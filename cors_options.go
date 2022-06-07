package azugo

import "strings"

var (
	defaultAllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	defaultCORSOptions    = CORSOptions{
		allowedMethods: strings.Join(defaultAllowedMethods, ", "),
	}
)

// CORSOptions is options for CORS middleware.
type CORSOptions struct {
	allowedMethods  string
	allowedHeaders  string
	allowAllOrigins bool
	allowedOrigins  map[string]struct{}
}

// SetMethods sets allowed CORS methods.
func (c *CORSOptions) SetMethods(methods ...string) *CORSOptions {
	c.allowedMethods = strings.Join(methods, ", ")
	return c
}

// Methods returns allowed CORS methods.
func (c *CORSOptions) Methods() string {
	return c.allowedMethods
}

// SetHeaders sets allowed CORS methods.
func (c *CORSOptions) SetHeaders(headers ...string) *CORSOptions {
	c.allowedHeaders = strings.Join(headers, ", ")
	return c
}

// Headers returns allowed CORS headers.
func (c *CORSOptions) Headers() string {
	return c.allowedHeaders
}

// SetOrigins sets allowed CORS origins. Set to `*` to allow all origins.
func (c *CORSOptions) SetOrigins(origins ...string) *CORSOptions {
	c.allowedOrigins = make(map[string]struct{}, len(origins))
	c.allowAllOrigins = false
	for _, origin := range origins {
		if origin == "*" {
			c.allowAllOrigins = true
			break
		}
		c.allowedOrigins[origin] = struct{}{}
	}
	return c
}

// ValidOrigins returns true if CORS origin is allowed.
func (c *CORSOptions) ValidOrigin(origin string) bool {
	if c.allowAllOrigins {
		return true
	}
	_, ok := c.allowedOrigins[origin]
	return ok
}
