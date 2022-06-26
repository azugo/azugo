package nonce

import (
	"context"
)

// Store represents a nonce store methods.
type Store interface {
	// Create a new nonce.
	Create(ctx context.Context) (string, error)

	// Verify checks if a nonce is valid.
	Verify(ctx context.Context, nonce string) (bool, error)
}
