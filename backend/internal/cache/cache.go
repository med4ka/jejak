package cache

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// LEARN:
//   Kenapa: Interface ini mendefinisikan operasi cache-aside pattern untuk URL shortener.
//   Konsep design yang terkait: cache-aside vs write-through - pola di mana aplikasi
//   cek cache dulu, kalau miss baru query DB lalu isi cache. Paling umum dipakai.
//   Trade-off: Cache-aside (lazy loading) lebih sederhana, latency read jadi 2x (cache miss
//   -> DB query), tapi write lebih cepat. Write-through malah menambah latency write.
//   Alternatif: Bisa pakai write-through untuk data kritis, tapi overhead write jadi besar
//   untuk URL shortener yang read-ceptional.
type Cache interface {
	Get(shortCode string) (string, bool)
	Set(shortCode, originalURL string, ttlSeconds int64) error
	Delete(shortCode string) error
}

// RedisCache implements Cache using Redis
type RedisCache struct {
	client *redis.Client
	ctx    context.Context
}

// LEARN:
//   Kenapa: Membungkus *redis.Client yang sudah ada (dipakai juga untuk queue)
//   supaya cache dan queue berbagi 1 koneksi pool, bukan buka koneksi ganda.
//   Trade-off: Cache ikut mati kalau client queue bermasalah (shared fate),
//   tapi hemat koneksi dan config cukup 1 REDIS_URL.
//   Alternatif: Client terpisah untuk cache vs queue (isolasi), tapi 2x koneksi.
func NewCacheFromClient(client *redis.Client) *RedisCache {
	return &RedisCache{
		client: client,
		ctx:    context.Background(),
	}
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(addr string, password string, db int) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &RedisCache{
		client: client,
		ctx:    context.Background(),
	}
}

// Get retrieves a value from cache. Returns (value, found).
func (c *RedisCache) Get(shortCode string) (string, bool) {
	val, err := c.client.Get(c.ctx, shortCode).Result()
	if err != nil {
		log.Printf("Cache miss for key %s: %v", shortCode, err)
		return "", false
	}
	return val, true
}

// Set stores a value in cache with a TTL (in seconds).
func (c *RedisCache) Set(shortCode, originalURL string, ttlSeconds int64) error {
	err := c.client.Set(c.ctx, shortCode, originalURL, time.Duration(ttlSeconds)*time.Second).Err()
	if err != nil {
		return err
	}
	return nil
}

// Delete removes a value from cache.
func (c *RedisCache) Delete(shortCode string) error {
	err := c.client.Del(c.ctx, shortCode).Err()
	if err != nil {
		return err
	}
	return nil
}