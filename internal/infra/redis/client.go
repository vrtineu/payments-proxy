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
		Addr:            addr,
		PoolSize:        10,
		MinIdleConns:    5,
		PoolTimeout:     2 * time.Second,
		MaxRetries:      3,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
	})

	return &RedisClient{
		Client: rdb,
	}
}
