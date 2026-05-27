package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisClient(addr string) *RedisClient {
	return &RedisClient{
		client: redis.NewClient(&redis.Options{
			Addr:         addr,
			DialTimeout:  3 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		}),
		ctx: context.Background(),
	}
}

func (c *RedisClient) Ping() error {
	return c.client.Ping(c.ctx).Err()
}

func (c *RedisClient) Set(key, value string) error {
	return c.client.Set(c.ctx, key, value, 0).Err()
}

func (c *RedisClient) Get(key string) (string, error) {
	return c.client.Get(c.ctx, key).Result()
}
