package nonce

import (
	"context"
	"sync"
	"testing"

	"azugo.io/core/cache"
	"github.com/go-quicktest/qt"
)

func newCacheStore(t *testing.T) *CacheNonceStore {
	t.Helper()

	c := cache.New(cache.MemoryCache)
	qt.Assert(t, qt.IsNil(c.Start(context.Background())))
	t.Cleanup(c.Close)

	inst, err := cache.Create[bool](c, "nonce")
	qt.Assert(t, qt.IsNil(err))

	return NewCacheNonceStore(inst)
}

func TestCacheNonceCreateAndVerify(t *testing.T) {
	store := newCacheStore(t)
	ctx := context.Background()

	nonce, err := store.Create(ctx)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(nonce != ""))

	ok, err := store.Verify(ctx, nonce)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(ok))

	// A nonce is single use, so the second presentation is refused.
	ok, err = store.Verify(ctx, nonce)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(ok))
}

func TestCacheNonceVerifyUnknown(t *testing.T) {
	store := newCacheStore(t)

	ok, err := store.Verify(context.Background(), "never-issued")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsFalse(ok))
}

func TestCacheNonceConcurrentVerifyAcceptsOnce(t *testing.T) {
	store := newCacheStore(t)
	ctx := context.Background()

	nonce, err := store.Create(ctx)
	qt.Assert(t, qt.IsNil(err))

	const racers = 16

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		accepted int
	)

	wg.Add(racers)

	for range racers {
		go func() {
			defer wg.Done()

			ok, err := store.Verify(ctx, nonce)
			qt.Check(t, qt.IsNil(err))

			if ok {
				mu.Lock()
				accepted++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	qt.Check(t, qt.Equals(accepted, 1))
}

func TestCacheNonceCreateIsUnique(t *testing.T) {
	store := newCacheStore(t)
	ctx := context.Background()

	seen := make(map[string]struct{}, 64)

	for range 64 {
		nonce, err := store.Create(ctx)
		qt.Assert(t, qt.IsNil(err))

		_, dup := seen[nonce]
		qt.Check(t, qt.IsFalse(dup))

		seen[nonce] = struct{}{}
	}
}
