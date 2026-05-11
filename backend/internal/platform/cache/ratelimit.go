package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisRateLimiter struct {
	client *redis.Client
	prefix string
}

func NewRedisRateLimiter(client *redis.Client, prefix string) *RedisRateLimiter {
	return &RedisRateLimiter{client: client, prefix: strings.TrimSpace(prefix)}
}

func (l *RedisRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, time.Duration, error) {
	if limit <= 0 {
		return true, 0, nil
	}
	if window <= 0 {
		window = time.Minute
	}

	redisKey := l.key(key)
	count, err := l.client.Incr(ctx, redisKey).Result()
	if err != nil {
		return false, 0, err
	}
	if count == 1 {
		if err := l.client.Expire(ctx, redisKey, window).Err(); err != nil {
			return false, 0, err
		}
	}
	if count <= int64(limit) {
		return true, 0, nil
	}

	ttl, err := l.client.TTL(ctx, redisKey).Result()
	if err != nil {
		return false, 0, err
	}
	if ttl <= 0 {
		ttl = window
		_ = l.client.Expire(ctx, redisKey, window).Err()
	}

	return false, ttl, nil
}

func (l *RedisRateLimiter) key(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return l.prefix + hex.EncodeToString(sum[:])
}
