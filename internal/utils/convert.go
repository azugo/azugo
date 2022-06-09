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
	return *(*string)(unsafe.Pointer(&b))
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
