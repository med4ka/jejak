package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"

	"jejak/internal/db"
	"jejak/internal/env"
)

// Worker processes click events from the Redis queue asynchronously
type Worker struct {
	client *redis.Client
	store  db.ShardStore
	ctx    context.Context
	stopCh chan struct{}
}

// NewWorker creates a new click event worker. store persists each event
// (click_events row + click_count) — without it the queue would drain into
// logs only and analytics tables would stay empty forever.
// LEARN:
//
//	Kenapa: worker menerima *redis.Options LENGKAP (bukan hanya Addr seperti
//	sebelumnya). redis.ParseURL menghasilkan Options yang membawa Password dan
//	TLSConfig (untuk rediss:// / Upstash). Kalau hanya opt.Addr yang disalin,
//	worker terhubung ke host yang benar TANPA kredensial -> AUTH / TLS handshake
//	gagal, antrian tidak pernah ter-proses. Dengan NewWorker(opt) sekaligus
//	menerima password & TLS, worker berperilaku identik dengan client Redis API
//	server (yang sejak awal memakai ParseURL di main.go). DSN lengkap tetap
//	dibaca di main() -> masih support default lokal tanpa password.
//	Trade-off: Options dibuat di main() dan dibagi; pooling go-redis default
//	(10 koneksi) cukup untuk pola poll-1-detik ini.
//	Alternatif: Redis Stream + consumer group (message-ack robust), Redis list
//	dengan BRPOP. List tetap dipakai karena sudah konsisten dgn LPush handler.
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
			// Process one batch of click events
			w.processBatch()
		}
	}
}

// Stop signals the worker to stop processing
func (w *Worker) Stop() {
	close(w.stopCh)
}

// processBatch processes the FULL backlog of click events from the Redis
// queue (loop LPop until empty), so throughput is NOT bounded to 1 event/sec.
// LEARN:
//
//	Kenapa: versi asli meng-LPop SATU pesan per tick (1/detik). Di load test
//	full-stack (N=200) antrian memuat 200 event -> worker butuh ~200 detik
//	untuk menguras; analitik & counter terlihat "basi" sangat lama. Dengan
//	meng-loop sampai list kosong dalam SATU tick, worker menguras backlog
//	dalam <1 detik dan tinggal idle di tick berikutnya (LPop kosong -> cepat
//	return, bukan busy-loop; sleep poll interval nyaris sm nol ketika kosong).
//	Trade-off: Burst besar = banyak INSERT+UPDATE beruntun dalam satu tick
//	(beban sesaat tinggi), tapi ini justru desain queue yang benar: cepat
//	kuras, cepat luang. Batas atas tak dibuat: bloom tetap dibatasi Redis list
//	yang hanya bertambah sebesar request masuk; DB bisa mengejar.
//	Alternatif: BRPOP blocking dengan batas 1 (per-invoice worker), atau
//	batch RPOPCOUNT. LPop-berulang paling sederhana dan tetap kompatibel.
func (w *Worker) processBatch() {
	for {
		ctx, cancel := context.WithTimeout(w.ctx, 100*time.Millisecond)
		msg, err := w.client.LPop(ctx, "click_events").Result()
		cancel()
		if err != nil || msg == "" {
			return // No more messages (kosong atau timeout)
		}
		w.processEvent(msg)
	}
}

// processEvent parses and persists ONE click event from the queue.
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

	// Fase 14: referrer_domain + is_unique + clicked_at dikirim handler via
	// queue (lihat logClickAsync). Worker hanya meneruskan nilai itu ke
	// LogClick supaya analitik breakdown per-domain & unique count akurat.
	// clicked_at default ke now kalau field tidak ada (defensive).
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
	// At-most-once: failure here loses the event (logged, not retried) —
	// same fire-and-forget philosophy as the redirect path.
	if err := w.store.LogClick(db.ClickEvent{
		ShortCode:      shortCode,
		Referrer:       event["referrer"],
		ReferrerDomain: event["referrer_domain"],
		IsUnique:       unique,
		ClickedAt:      clickedAt,
	}); err != nil {
		log.Printf("Warning: failed to persist click event for %s: %v", shortCode, err)
		return
	}
	log.Printf("Processed click event for short_code: %s", shortCode)
}

func main() {
	// .env (kalau ada) dimuat dulu; env OS yang sudah di-set selalu menang.
	env.LoadDotEnv()

	// Redis connection options from env; ParseURL the full DSN so password &
	// TLS (rediss://, Upstash) are preserved — NOT just the Addr. Nilai default
	// "localhost:6379" menunjuk Redis native lokal (hasil redisd.default())
	// tanpa kredensial; kalau REDIS_URL diisi, selalu pakai hasil parse.
	redisOpt := &redis.Options{Addr: "localhost:6379"}
	if u := os.Getenv("REDIS_URL"); u != "" {
		if opt, err := redis.ParseURL(u); err == nil {
			redisOpt = opt
		} else {
			log.Printf("Warning: invalid REDIS_URL, using localhost:6379: %v", err)
		}
	}

	// Database for persisting events (compose already passes DATABASE_URL).
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required (worker persists click events)")
	}
	store, err := db.NewSingleStore(dbURL, "")
	if err != nil {
		log.Fatal("Failed to create store:", err)
	}

	// Create worker with Redis connection options
	worker := NewWorker(redisOpt, store)

	// Start worker in a goroutine
	go worker.Start()

	// Keep the main goroutine running
	select {}
}
