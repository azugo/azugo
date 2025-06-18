package azugo

import (
	"strings"
)

const (
	HeaderOrigin           string = "Origin"
	HeaderAllowOrigin      string = "Access-Control-Allow-Origin"
	HeaderAllowMethods     string = "Access-Control-Allow-Methods"
	HeaderAllowHeaders     string = "Access-Control-Allow-Headers"
	HeaderAllowCredentials string = "Access-Control-Allow-Credentials"
)

var (
	defaultAllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	defaultCORSOptions    = CORSOptions{
		allowedMethods:   strings.Join(defaultAllowedMethods, ", "),
		allowCredentials: false,
	}
)

// CORSOptions is options for CORS middleware.
type CORSOptions struct {
	allowedMethods   string
	allowedHeaders   string
	allowAllOrigins  bool
	allowedOrigins   map[string]struct{}
	allowCredentials bool
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

// AllowCredentials returns flag if CORS credentials are alloweds.
func (c *CORSOptions) AllowCredentials() bool {
	return c.allowCredentials
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

// SetCORSHeaders applies the CORS headers to the response based on the provided options and origin.
func SetCORSHeaders(ctx *Context, opts *CORSOptions, origin string) {
	if len(origin) == 0 || !opts.ValidOrigin(origin) {
		return
	}

	ctx.Header.Set(HeaderAllowOrigin, origin)
	ctx.Header.Set(HeaderAllowMethods, opts.Methods())
	ctx.Header.Set(HeaderAllowHeaders, opts.Headers())

	if opts.AllowCredentials() {
		ctx.Header.Set(HeaderAllowCredentials, "true")
	}
}
