package cache

import (
	"context"
	"errors"

	"github.com/go-redis/redis/v9"
)

// Cache represents a cache.
type Cache struct {
	options []CacheOption
	cache   map[string]any
}

// New creates a new cache with specified type.
func New(opts ...CacheOption) *Cache {
	c := &Cache{
		options: opts,
		cache:   make(map[string]any),
	}
	return c
}

// CacheInstance represents a cache instance.
type CacheInstance[T any] interface {
	// Get value from cache. If value is not found, it will return default value.
	Get(ctx context.Context, key string, opts ...ItemOption[T]) (T, error)
	// Set value in cache.
	Set(ctx context.Context, key string, value T, opts ...ItemOption[T]) error
	// Delete value from cache.
	Delete(ctx context.Context, key string) error
}

// Get returns pre-configured cache instance by name.
func Get[T any](cache *Cache, name string) (CacheInstance[T], error) {
	i, ok := cache.cache[name]
	if !ok {
		return nil, errors.New("cache not found")
	}
	r, ok := i.(CacheInstance[T])
	if !ok {
		return nil, errors.New("invalid cache type")
	}
	return r, nil
}

// Create new cache instance with specified name and options.
func Create[T any](cache *Cache, name string, opts ...CacheOption) (CacheInstance[T], error) {
	opt := append(append([]CacheOption{}, cache.options...), opts...)

	o := newCacheOptions(opt...)

	switch o.Type {
	case MemoryCache:
		return newMemoryCache[T](opt...)
	case RedisCache:
		return newRedisCache[T](name, opt...)
	}
	return nil, errors.New("unsupported cache type")
}

// ValidateConnectionString validates connection string for specific cache type.
func ValidateConnectionString(typ CacheType, connStr string) error {
	if typ == RedisCache {
		if len(connStr) == 0 {
			return errors.New("Redis connection string can not be empty")
		}
		if _, err := redis.ParseURL(connStr); err != nil {
			return err
		}
		return nil
	}
	return nil
}
