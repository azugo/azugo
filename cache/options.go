package cache

import (
	"time"
)

type cacheOptions struct {
	Type             CacheType
	TTL              time.Duration
	ConnectionString string
	KeyPrefix        string
}

// CacheOption is an option for the cache instance.
type CacheOption interface {
	applyCache(*cacheOptions)
}

func newCacheOptions(opts ...CacheOption) *cacheOptions {
	opt := &cacheOptions{}
	for _, o := range opts {
		o.applyCache(opt)
	}
	return opt
}

type itemOptions[T any] struct {
	TTL          time.Duration
	DefaultValue T
}

// ItemOption is an option for the cached item.
type ItemOption[T any] interface {
	applyItem(*itemOptions[T])
}

func newItemOptions[T any](opts ...ItemOption[T]) *itemOptions[T] {
	opt := &itemOptions[T]{}
	for _, o := range opts {
		o.applyItem(opt)
	}
	return opt
}

// CacheType represents a cache type.
type CacheType string

const (
	// MemoryCache store data in memory.
	MemoryCache CacheType = "memory"
	// RedisCache store data in Redis database.
	RedisCache CacheType = "redis"
)

func (t CacheType) applyCache(c *cacheOptions) {
	c.Type = t
}

// DefaultTTL is an default TTL for items in cache instance.
type DefaultTTL time.Duration

func (t DefaultTTL) applyCache(c *cacheOptions) {
	c.TTL = time.Duration(t)
}

// TTL represents time to keep item in cache.
type TTL[T any] time.Duration

func (t TTL[T]) applyItem(c *itemOptions[T]) {
	c.TTL = time.Duration(t)
}

// ConnectionString is a connection string for the cache instance.
type ConnectionString string

func (cs ConnectionString) applyCache(c *cacheOptions) {
	c.ConnectionString = string(cs)
}

// KeyPrefix is a prefix for the cache keys.
type KeyPrefix string

func (kp KeyPrefix) applyCache(c *cacheOptions) {
	c.KeyPrefix = string(kp)
}
