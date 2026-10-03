package cache

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache defines the cache-aside operations for the URL shortener.
// Rationale: cache-aside versus write-through: the application checks the
// cache first and only on a miss queries the database, then fills the cache;
// this is by far the most common pattern. Trade-off: cache-aside (lazy
// loading) is simpler and writes stay fast, but a miss doubles read latency
// (cache miss → database query). Write-through adds write latency instead.
// Alternative: write-through for critical data, but the write overhead is too
// high for a URL shortener that is overwhelmingly read-heavy.
type Cache interface {
	// Get returns (value, true) on a hit; a miss or backend error returns
	// ("", false) - never a hard error (the caller falls through to the DB).
	Get(shortCode string) (string, bool)
	// Set stores the value with a TTL; returns the backend error when it
	// fails (the entry then does not exist).
	Set(shortCode, originalURL string, ttlSeconds int64) error
	// Delete evicts one key; a key that does not exist is not an error.
	Delete(shortCode string) error
}

// RedisCache implements Cache using Redis
type RedisCache struct {
	client *redis.Client
	ctx    context.Context
}

// NewCacheFromClient wraps an existing *redis.Client (also used for the
// queue) so cache and queue share one connection pool instead of opening
// separate connections. Trade-off: the cache dies with a faulty queue client
// (shared fate), but connections are saved and configuration needs only one
// REDIS_URL. Alternative: a separate client for cache versus queue (isolation)
// at the cost of twice the connections.
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

// Set stores a value in cache with a TTL (in seconds). Returns the Redis
// error when SET fails (the entry then does not exist, so the next read is a
// miss); a key that already exists is simply overwritten, never an error.
func (c *RedisCache) Set(shortCode, originalURL string, ttlSeconds int64) error {
	err := c.client.Set(c.ctx, shortCode, originalURL, time.Duration(ttlSeconds)*time.Second).Err()
	if err != nil {
		return err
	}
	return nil
}

// Delete removes a value from cache. Returns the Redis error when DEL fails;
// a key that does not exist is not an error (DEL of an absent key succeeds).
func (c *RedisCache) Delete(shortCode string) error {
	err := c.client.Del(c.ctx, shortCode).Err()
	if err != nil {
		return err
	}
	return nil
}
