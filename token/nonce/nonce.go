package nonce

// Store represents a nonce store methods.
type Store interface {
	// Create a new nonce.
	Create() (string, error)

	// Verify checks if a nonce is valid.
	Verify(nonce string) (bool, error)
}
