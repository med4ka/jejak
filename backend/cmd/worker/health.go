package main

// Link health monitor (migration 17, 2026-10-04): the worker checks up to
// 50 destinations per run - once shortly after boot, then every 6 hours -
// with 1.2s spacing (<= 50 requests/minute, the spec budget). The Redis
// worker lock already guarantees a single worker instance, so no separate
// health lock is needed (same single-instance assumption as the click
// queue). The shared Redis client also backs the redirect cache, letting
// CheckOne evict a payload whose health_status changed (a status flip then
// takes effect immediately instead of after the 300s TTL).

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"jejak/internal/cache"
	"jejak/internal/db"
	"jejak/internal/health"
)

// startHealthMonitor runs checker batches for the lifetime of ctx: an
// immediate run on boot (a deploy restart therefore refreshes every status
// within one batch window), then health.BatchInterval ticks. Each run logs
// its result; a failing run never stops the loop (the next tick retries).
func startHealthMonitor(ctx context.Context, store db.ShardStore, rdb *redis.Client) {
	checker := &health.Checker{
		Store:  store,
		Cache:  cache.NewCacheFromClient(rdb),
		Logger: log.Default(),
	}

	run := func() {
		start := time.Now()
		done, err := checker.RunBatch(ctx)
		if err != nil && ctx.Err() == nil {
			log.Printf("health batch: %d link dicek (%s), error: %v", done, time.Since(start).Round(time.Millisecond), err)
			return
		}
		log.Printf("health batch: %d link dicek dalam %s", done, time.Since(start).Round(time.Millisecond))
	}

	log.Printf("health monitor aktif (interval %s, maks %d link/run)", health.BatchInterval, health.BatchLimit)
	run()

	ticker := time.NewTicker(health.BatchInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("health monitor berhenti")
			return
		case <-ticker.C:
			run()
		}
	}
}
