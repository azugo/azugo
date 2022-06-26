package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/goccy/go-json"
)

type redisCache[T any] struct {
	con    *redis.Client
	prefix string
	ttl    time.Duration
}

func newRedisCache[T any](prefix string, opts ...CacheOption) (CacheInstance[T], error) {
	opt := newCacheOptions(opts...)
	redisOptions, err := redis.ParseURL(opt.ConnectionString)
	if err != nil {
		return nil, err
	}
	return &redisCache[T]{
		con:    redis.NewClient(redisOptions),
		prefix: opt.KeyPrefix + prefix + ":",
		ttl:    opt.TTL,
	}, nil
}

func (c *redisCache[T]) Get(ctx context.Context, key string, opts ...ItemOption[T]) (T, error) {
	s := c.con.Get(ctx, c.prefix+key)
	var val T
	if s.Err() == redis.Nil {
		return val, nil
	}
	if s.Err() != nil {
		return val, s.Err()
	}
	if err := json.Unmarshal([]byte(s.Val()), &val); err != nil {
		return val, fmt.Errorf("invalid cache value: %w", err)
	}
	return val, nil
}

func (c *redisCache[T]) Set(ctx context.Context, key string, value T, opts ...ItemOption[T]) error {
	buf, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("invalid cache value: %w", err)
	}
	opt := newItemOptions(opts...)
	ttl := c.ttl
	if opt.TTL != 0 {
		ttl = opt.TTL
	}
	s := c.con.Set(ctx, c.prefix+key, string(buf), ttl)
	if s.Err() != nil {
		return s.Err()
	}
	return nil
}

func (c *redisCache[T]) Delete(ctx context.Context, key string) error {
	s := c.con.Del(ctx, c.prefix+key)
	if s.Err() != nil {
		return s.Err()
	}
	return nil
}
