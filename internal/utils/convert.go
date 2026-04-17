// Package utils provides internal utility helpers.
package utils

import (
	"fmt"
	"net/url"
	"strings"
	"unsafe"
)

// B2S converts byte slice to a string without memory allocation.
// See https://groups.google.com/forum/#!msg/Golang-Nuts/ENgbUzYvCuU/90yGx7GUAgAJ .
func B2S(b []byte) string {
	return *(*string)(unsafe.Pointer(&b)) //nolint:gosec
}

// ParseBoolValue reports whether the string value represents a truthy boolean.
// Accepted truthy values: "true" (case-insensitive) and "1".
func ParseBoolValue(v string) bool {
	return strings.EqualFold(v, "true") || v == "1"
}

// MapToURLValues converts a map to url.Values.
func MapToURLValues(m map[string]any) string {
	p := &url.Values{}

	for key, value := range m {
		var val string

		switch v := value.(type) {
		case []string:
			val = strings.Join(v, ",")
		default:
			val = fmt.Sprintf("%v", v)
		}

		p.Add(key, val)
	}

	return p.Encode()
}
