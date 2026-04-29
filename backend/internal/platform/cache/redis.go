package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedis(addr string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: addr,
	})
}

type JSONCache struct {
	client *redis.Client
	prefix string
}

func NewJSONCache(client *redis.Client, prefix string) *JSONCache {
	return &JSONCache{client: client, prefix: prefix}
}

func (c *JSONCache) Get(ctx context.Context, key string, out any) (bool, error) {
	value, err := c.client.Get(ctx, c.prefix+key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if err := json.Unmarshal([]byte(value), out); err != nil {
		return false, err
	}

	return true, nil
}

func (c *JSONCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, c.prefix+key, body, ttl).Err()
}
