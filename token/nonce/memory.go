package nonce

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"azugo.io/core/cache"
	"github.com/oklog/ulid/v2"
)

// CacheNonceStore is a nonce store that stores nonces in cache.
type CacheNonceStore struct {
	cache   cache.CacheInstance[bool]
	entropy *ulid.MonotonicEntropy
}

// NewCacheNonceStore creates a new nonce store that stores nonces in cache.
func NewCacheNonceStore(c cache.CacheInstance[bool]) *CacheNonceStore {
	return &CacheNonceStore{
		entropy: ulid.Monotonic(rand.Reader, 0),
		cache:   c,
	}
}

func (s *CacheNonceStore) Create(ctx context.Context) (string, error) {
	id, err := ulid.New(ulid.Timestamp(time.Now().UTC()), s.entropy)
	if err != nil {
		return "", err
	}

	key := id.String()
	if err := s.cache.Set(ctx, key, true); err != nil {
		return "", fmt.Errorf("nonce can not be stored in cache: %w", err)
	}

	return key, nil
}

func (s *CacheNonceStore) Verify(ctx context.Context, nonce string) (bool, error) {
	i, err := s.cache.Get(ctx, nonce)
	if err != nil {
		return false, err
	}
	if i {
		// Ignore error if nonce can not be deleted from cache
		_ = s.cache.Delete(ctx, nonce)
	}
	return i, nil
}
