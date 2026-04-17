// Package radix provides a radix tree implementation for HTTP routing.
package radix

const (
	root nodeType = iota
	static
	param
	wildcard
)
