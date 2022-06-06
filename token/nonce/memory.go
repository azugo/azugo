package nonce

import (
	"crypto/rand"
	"errors"
	"time"

	"github.com/lafriks/ttlcache/v3"
	"github.com/oklog/ulid/v2"
)

// MemoryNonceStore is a nonce store that stores nonces in memory.
type MemoryNonceStore struct {
	cache   *ttlcache.Cache[string, struct{}]
	ttl     time.Duration
	entropy *ulid.MonotonicEntropy
}

// NewMemoryNonceStore creates a new nonce store that stores nonces in memory.
func NewMemoryNonceStore(ttl time.Duration) *MemoryNonceStore {
	return &MemoryNonceStore{
		entropy: ulid.Monotonic(rand.Reader, 0),
		cache: ttlcache.New(
			ttlcache.WithTTL[string, struct{}](ttl),
		),
	}
}

func (s *MemoryNonceStore) Create() (string, error) {
	id, err := ulid.New(ulid.Timestamp(time.Now().UTC()), s.entropy)
	if err != nil {
		return "", err
	}

	i := s.cache.Set(id.String(), struct{}{}, s.ttl)
	if i == nil {
		return "", errors.New("nonce can not be stored in store")
	}

	return i.Key(), nil
}

func (s *MemoryNonceStore) Verify(nonce string) (bool, error) {
	i, err := s.cache.Get(nonce)
	if err != nil {
		return false, err
	}
	if i != nil {
		s.cache.Delete(i.Key())
	}
	return i != nil && !i.IsExpired(), nil
}
