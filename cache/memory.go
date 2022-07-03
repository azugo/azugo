package cache

import (
	"context"
	"fmt"

	"github.com/lafriks/ttlcache/v3"
)

type memoryCache[T any] struct {
	cache  *ttlcache.Cache[string, T]
	loader func(ctx context.Context, key string) (interface{}, error)
}

func newMemoryCache[T any](opts ...CacheOption) (CacheInstance[T], error) {
	opt := newCacheOptions(opts...)
	c := ttlcache.New(ttlcache.WithTTL[string, T](opt.TTL))
	return &memoryCache[T]{
		cache:  c,
		loader: opt.Loader,
	}, nil
}

func (c *memoryCache[T]) getLoader(ctx context.Context, opts ...ItemOption[T]) ttlcache.LoaderFunc[string, T] {
	return func(cache *ttlcache.Cache[string, T], key string) (*ttlcache.Item[string, T], error) {
		opt := newItemOptions(opts...)
		ttl := opt.TTL
		if ttl == 0 {
			ttl = ttlcache.DefaultTTL
		}

		v, err := c.loader(ctx, key)
		if err != nil {
			return nil, err
		}
		vv, ok := v.(T)
		if !ok {
			return nil, fmt.Errorf("invalid value from loader: %v", v)
		}
		return cache.Set(key, vv, ttl), nil
	}
}

func (c *memoryCache[T]) Get(ctx context.Context, key string, opts ...ItemOption[T]) (T, error) {
	var val T
	if c.cache == nil {
		return val, ErrCacheClosed
	}

	cacheOpts := make([]ttlcache.Option[string, T], 0)

	if c.loader != nil {
		cacheOpts = append(cacheOpts, ttlcache.WithLoader[string, T](c.getLoader(ctx, opts...)))
	}

	i, err := c.cache.Get(key, cacheOpts...)
	if err != nil || i == nil {
		return val, err
	}
	if i.IsExpired() {
		return val, nil
	}
	return i.Value(), nil
}

func (c *memoryCache[T]) Set(ctx context.Context, key string, value T, opts ...ItemOption[T]) error {
	if c.cache == nil {
		return ErrCacheClosed
	}
	opt := newItemOptions(opts...)
	ttl := opt.TTL
	if ttl == 0 {
		ttl = ttlcache.DefaultTTL
	}
	_ = c.cache.Set(key, value, ttl)
	return nil
}

func (c *memoryCache[T]) Delete(ctx context.Context, key string) error {
	if c.cache == nil {
		return ErrCacheClosed
	}
	c.cache.Delete(key)
	return nil
}

func (c *memoryCache[T]) Close() {
	if c.cache == nil {
		return
	}
	c.cache.DeleteAll()
	c.cache = nil
}
