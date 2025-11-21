// Package reddis
package reddis

import (
	"context"
	"time"

	"github.com/Wladim1r/auth/lib/getenv"
	"github.com/redis/go-redis/v9"
)

type RDB struct {
	rdb *redis.Client
}

func NewClient() *RDB {
	return &RDB{
		rdb: redis.NewClient(&redis.Options{
			Addr:     getenv.GetString("REDIS_ADDR", "localhost:6379"),
			Password: getenv.GetString("REDIS_PASSWORD", ""),
			DB:       getenv.GetInt("REDIS_DB", 0),
		}),
	}
}

func (r *RDB) Record(ctx context.Context, key string, value interface{}, t time.Duration) error {
	return r.rdb.Set(ctx, key, value, t).Err()
}

func (r *RDB) Receive(ctx context.Context, key string) (string, error) {
	return r.rdb.Get(ctx, key).Result()
}

func (r *RDB) Delete(ctx context.Context, keys ...string) error {
	return r.rdb.Del(ctx, keys...).Err()
}

func (r *RDB) Close() {
	r.rdb.Close()
}
