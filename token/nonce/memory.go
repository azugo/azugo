// Package nonce provides nonce store implementations for WS-Federation.
package nonce

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"azugo.io/core/cache"
	"github.com/oklog/ulid/v2"
)

// CacheNonceStore is a nonce store that stores nonces in cache.
type CacheNonceStore struct {
	cache   cache.Instance[bool]
	entropy *ulid.MonotonicEntropy
}

// NewCacheNonceStore creates a new nonce store that stores nonces in cache.
func NewCacheNonceStore(c cache.Instance[bool]) *CacheNonceStore {
	return &CacheNonceStore{
		entropy: ulid.Monotonic(rand.Reader, 0),
		cache:   c,
	}
}

// Create creates a new nonce and stores it in the cache.
func (s *CacheNonceStore) Create(ctx context.Context) (string, error) {
	id, err := ulid.New(ulid.Timestamp(time.Now().UTC()), s.entropy)
	if err != nil {
		return "", err
	}

	key := id.String()
	if err := s.cache.Set(ctx, key, true); err != nil {
		return "", fmt.Errorf("nonce can not be stored in cache: %w", err)
	}

	if err := s.cache.Sync(ctx); err != nil {
		return "", fmt.Errorf("nonce can not be stored in cache: %w", err)
	}

	return key, nil
}

// Verify checks and consumes a nonce from the cache.
func (s *CacheNonceStore) Verify(ctx context.Context, nonce string) (bool, error) {
	i, err := s.cache.Pop(ctx, nonce)
	if err != nil {
		var knf cache.KeyNotFoundError
		if errors.As(err, &knf) {
			return false, nil
		}

		return false, err
	}

	return i, nil
}
