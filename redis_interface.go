package sync_cache

import (
	"github.com/go-redis/redis"
	"time"
)

type Redis interface {
	Get(key string) *redis.StringCmd
	Set(key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(keys ...string) *redis.IntCmd
	FlushDB() *redis.StatusCmd
}
