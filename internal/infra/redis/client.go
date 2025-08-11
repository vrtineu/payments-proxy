package redis

import (
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client *redis.Client
}

func NewRedisClient() *RedisClient {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		PoolSize:     128,
		MinIdleConns: 16,
		PoolTimeout:  1 * time.Second,
	})

	return &RedisClient{
		Client: rdb,
	}
}
