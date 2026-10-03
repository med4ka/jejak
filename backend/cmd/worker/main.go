package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"jejak/internal/db"
	"jejak/internal/env"
)

// ---- Single-instance guard (2026-09-30) ----
//
// Rationale: workers could previously be started repeatedly without
// hindrance: two twin worker processes once ran concurrently. The consequence
// was not a crash but waste and confusing debugging: two consumers alternated
// popping the "click_events" list, so logs were split, process ordering was
// hard to follow, and every event was INSERTed twice for zero benefit (the
// queue itself stayed safe: LPop is atomic, so no event was stored twice;
// only worker effort was wasted).
// Mechanism: SETNX on key `jejak:worker:lock:click_events` stores the
// instance id with a 30-second TTL, refreshed (heartbeat) every 10 seconds.
// A second worker sees the key already set and exits instead of taking over.
// Trade-off (deliberately accepted): a 30s TTL against a 10s heartbeat gives
// a 3× margin: if the main process is frozen for more than 30 seconds (long
// GC, ctrl+z on Windows) the lock expires and a second worker may enter; when
// the first process wakes, its heartbeat finds the ownership lost and it is
// the one that exits (correct behavior: the earlier process yields rather
// than both running). This is not a perfect lock (that would require fencing
// tokens): it is sufficient for development and makes "forgot to stop the
// old worker" no longer end with two consumers. Alternatives considered:
// (a) documentation only: rejected because the trap had already occurred in
// practice; (b) a local PID file: rejected because it does not survive a
// force-killed process (stale file) and does not work across two machines;
// (c) Redis Stream consumer groups: a major change to consumption, out of
// scope for "prevent double-run".
const (
	workerLockKey   = "jejak:worker:lock:click_events"
	workerLockTTL   = 30 * time.Second
	workerHeartbeat = 10 * time.Second
)

// renewWorkerLock extends the TTL only while the key still holds this
// process's instance id (atomic GET+compare+PEXPIRE in Lua). A 0 return means
// another process has taken the lock (its TTL lapsed) and the old owner must
// exit.
var renewWorkerLock = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0`)

// releaseWorkerLock releases this process's own lock at shutdown: without an
// ownership condition, shutdown would delete a lock belonging to another
// worker that had taken over (a narrow race, and still safe: the only result
// would be "lock lost early" and the new worker simply acquires it again).
var releaseWorkerLock = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0`)

// workerInstanceID produces an id unique per process: hostname + PID. Enough
// to tell two workers on the same machine apart (the case that actually
// occurred).
func workerInstanceID() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "unknown-host"
	}
	return fmt.Sprintf("%s:%d", host, os.Getpid())
}

// Worker processes click events from the Redis queue asynchronously
type Worker struct {
	client *redis.Client
	store  db.ShardStore
	ctx    context.Context
	stopCh chan struct{}
}

// NewWorker creates a new click event worker. store persists each event
// (click_events row + click_count): without it the queue would drain into
// logs only and analytics tables would stay empty forever.
// The worker receives the COMPLETE *redis.Options (not just Addr as before):
// redis.ParseURL produces Options carrying Password and TLSConfig (for
// rediss:// / Upstash). If only opt.Addr were copied, the worker would connect
// to the correct host WITHOUT credentials and the AUTH / TLS handshake would
// fail, so the queue would never be processed. Accepting NewWorker(opt) with
// password & TLS makes the worker behave identically to the API server's Redis
// client (which has used ParseURL in main.go from the start); the full DSN is
// still read in main(), so the passwordless local default keeps working.
// Trade-off: the Options are created in main() and shared; the default
// go-redis pool (10 connections) is enough for this 1-second polling pattern.
// Alternatives: Redis Stream + consumer group (explicit per-message ack), or
// a Redis list with BRPOP. The list is kept because it is already consistent
// with the handler's LPush.
func NewWorker(redisOpt *redis.Options, store db.ShardStore) *Worker {
	return &Worker{
		client: redis.NewClient(redisOpt),
		store:  store,
		ctx:    context.Background(),
		stopCh: make(chan struct{}),
	}
}

// Start begins processing click events from the Redis queue
func (w *Worker) Start() {
	log.Println("Click event worker started")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			log.Println("Click event worker stopped")
			return
		case <-ticker.C:
			w.processBatch()
		}
	}
}

// Stop signals the worker to stop processing
func (w *Worker) Stop() {
	close(w.stopCh)
}

// processBatch processes the FULL backlog of click events from the Redis
// queue (loops LPop until empty), so throughput is NOT bounded to 1 event/sec.
// The original version popped ONE message per tick (1/second): in the
// full-stack load test (N=200) the queue held 200 events, so the worker
// needed ~200 seconds to drain it and analytics and counters looked stale for
// a long time. Looping until the list is empty within a SINGLE tick drains
// the backlog in under 1 second and the worker then idles on the following
// tick (an empty LPop returns quickly instead of busy-looping, so the effective
// poll sleep is near zero while the queue is empty). Trade-off: a large burst
// means many consecutive INSERT+UPDATEs in one tick (high instantaneous load),
// but that is exactly the right queue design: drain fast, go idle fast. No
// upper bound is imposed: growth stays bounded by the Redis list, which only
// grows by the incoming requests, and the database can catch up. Alternatives:
// blocking BRPOP with a limit of 1 (one event per iteration), or a batched
// pop; repeated LPop is the simplest and stays compatible.
func (w *Worker) processBatch() {
	for {
		ctx, cancel := context.WithTimeout(w.ctx, 100*time.Millisecond)
		msg, err := w.client.LPop(ctx, "click_events").Result()
		cancel()
		if err != nil || msg == "" {
			return // no more messages (empty queue or timeout)
		}
		w.processEvent(msg)
	}
}

func (w *Worker) processEvent(msg string) {
	var event map[string]string
	if err := json.Unmarshal([]byte(msg), &event); err != nil {
		log.Printf("Warning: failed to unmarshal click event: %v", err)
		return
	}

	shortCode, ok := event["short_code"]
	if !ok {
		log.Printf("Warning: click event missing short_code")
		return
	}

	// Phase 14: referrer_domain + is_unique + clicked_at are sent by the
	// handler through the queue (see logClickAsync). The worker only forwards
	// those values to LogClick so per-domain breakdown and unique counts stay
	// accurate. clicked_at defaults to now when the field is absent (defensive).
	clickedAt := time.Now()
	if ts := event["clicked_at"]; ts != "" {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			clickedAt = t
		} else {
			log.Printf("Warning: click event has invalid clicked_at %q: %v", ts, err)
		}
	}
	unique := event["is_unique"] == "true"

	// Persist: one transaction writes the event row + bumps the counter.
	// At-most-once: failure here loses the event (logged, not retried):
	// same fire-and-forget philosophy as the redirect path.
	// Analytics dimensions (migration 16) are forwarded unchanged: device/
	// referrer classification already happens in the handler AT event-write
	// time (deviation A), so the worker does not repeat it: if it also
	// classified, two places with different rules could produce different
	// buckets for the same click.
	if err := w.store.LogClick(db.ClickEvent{
		ShortCode:      shortCode,
		Referrer:       event["referrer"],
		ReferrerDomain: event["referrer_domain"],
		ReferrerType:   event["referrer_type"],
		UserAgent:      event["user_agent"],
		DeviceType:     event["device_type"],
		IsUnique:       unique,
		ClickedAt:      clickedAt,
	}); err != nil {
		log.Printf("Warning: failed to persist click event for %s: %v", shortCode, err)
		return
	}
	log.Printf("Processed click event for short_code: %s", shortCode)
}

func main() {
	// .env (if present) is loaded first; OS environment variables that are
	// already set always win.
	env.LoadDotEnv()

	// Redis connection options from env; ParseURL the full DSN so password &
	// TLS (rediss://, Upstash) are preserved: NOT just the Addr. The default
	// "localhost:6379" points at the local native Redis (the redisd.default()
	// result) without credentials; when REDIS_URL is set, the parsed result is
	// always used.
	redisOpt := &redis.Options{Addr: "localhost:6379"}
	if u := os.Getenv("REDIS_URL"); u != "" {
		if opt, err := redis.ParseURL(u); err == nil {
			redisOpt = opt
		} else {
			log.Printf("Warning: invalid REDIS_URL, using localhost:6379: %v", err)
		}
	}

	// ---- Single-instance guard (SETNX + heartbeat, see the rationale above) ----
	rdb := redis.NewClient(redisOpt)
	defer rdb.Close()

	bootCtx, bootCancel := context.WithTimeout(context.Background(), 5*time.Second)
	instanceID := workerInstanceID()
	acquired, err := rdb.SetNX(bootCtx, workerLockKey, instanceID, workerLockTTL).Result()
	bootCancel()
	if err != nil {
		log.Fatalf("Worker lock gagal dicek (Redis): %v", err)
	}
	if !acquired {
		// Another worker already holds the lock: do NOT take it over, exit
		// immediately.
		log.Println("Worker sudah jalan, keluar.")
		os.Exit(1)
	}
	log.Printf("Worker lock diambil: %s (TTL %s, heartbeat tiap %s)", instanceID, workerLockTTL, workerHeartbeat)

	// Signal-aware context: Ctrl+C / SIGTERM releases the lock before exit so
	// a restart does not have to wait out the 30-second TTL.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Heartbeat: refresh the TTL every 10 seconds. If another process has
	// already taken the key (the TTL lapsed) → THIS process exits, not the
	// new one.
	go func() {
		t := time.NewTicker(workerHeartbeat)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				n, err := renewWorkerLock.Run(ctx, rdb, []string{workerLockKey}, instanceID, workerLockTTL.Milliseconds()).Int()
				if err != nil {
					// Transient Redis trouble: do not kill the worker (the TTL
					// is still alive); retry on the next tick. If the key is
					// truly gone, the next tick detects it via the 0 return.
					log.Printf("Warning: heartbeat lock gagal: %v", err)
					continue
				}
				if n == 0 {
					log.Println("Worker lock diambil proses lain: keluar.")
					os.Exit(1)
				}
			}
		}
	}()

	// Database for persisting events (compose already passes DATABASE_URL).
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required (worker persists click events)")
	}
	store, err := db.NewSingleStore(dbURL, "")
	if err != nil {
		log.Fatal("Failed to create store:", err)
	}

	worker := NewWorker(redisOpt, store)

	go worker.Start()

	// Block until the shutdown signal arrives, then release our own lock.
	<-ctx.Done()
	log.Println("Shutting down worker...")
	relCtx, relCancel := context.WithTimeout(context.Background(), 3*time.Second)
	if _, err := releaseWorkerLock.Run(relCtx, rdb, []string{workerLockKey}, instanceID).Result(); err != nil {
		log.Printf("Warning: lepas lock gagal (akan kedaluwarsa sendiri dalam %s): %v", workerLockTTL, err)
	}
	relCancel()
}
