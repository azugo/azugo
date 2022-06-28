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

func newRedisCache[T any](prefix string, con *redis.Client, opts ...CacheOption) (CacheInstance[T], error) {
	opt := newCacheOptions(opts...)

	return &redisCache[T]{
		con:    con,
		prefix: opt.KeyPrefix + prefix + ":",
		ttl:    opt.TTL,
	}, nil
}

func newRedisClient(constr, password string) (*redis.Client, error) {
	redisOptions, err := redis.ParseURL(constr)
	if err != nil {
		return nil, err
	}
	// If password is provided override provided in connection string.
	if len(password) != 0 {
		redisOptions.Password = password
	}

	return redis.NewClient(redisOptions), nil
}

func (c *redisCache[T]) Get(ctx context.Context, key string, opts ...ItemOption[T]) (T, error) {
	var val T
	if c.con == nil {
		return val, ErrCacheClosed
	}
	s := c.con.Get(ctx, c.prefix+key)
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
	if c.con == nil {
		return ErrCacheClosed
	}
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
	if c.con == nil {
		return ErrCacheClosed
	}
	s := c.con.Del(ctx, c.prefix+key)
	if s.Err() != nil {
		return s.Err()
	}
	return nil
}

func (c *redisCache[T]) Close() {
	if c.con == nil {
		return
	}
	_ = c.con.Close()
	c.con = nil
}
