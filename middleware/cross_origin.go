package middleware

import (
	"net/url"
	"strings"

	"azugo.io/azugo"

	"azugo.io/core/http"
)

// CrossOriginError is returned by the CrossOriginProtection middleware for a request that a page
// of another origin sent.
type CrossOriginError struct{}

// Error implements the error interface.
func (CrossOriginError) Error() string {
	return "cross-origin request"
}

// SafeError returns a message that can be safely returned to the client.
func (CrossOriginError) SafeError() string {
	return "cross-origin request"
}

// StatusCode returns the HTTP status code for the cross-origin error.
func (CrossOriginError) StatusCode() int {
	return http.StatusForbidden
}

type crossOriginProtection struct {
	trusted map[string]struct{}
}

// CrossOriginOption configures CrossOriginProtection.
type CrossOriginOption interface {
	apply(m *crossOriginProtection)
}

// TrustedOrigins are origins, "scheme://host[:port]", whose cross-origin requests are allowed.
type TrustedOrigins []string

func (o TrustedOrigins) apply(m *crossOriginProtection) {
	for _, origin := range o {
		u, err := url.Parse(origin)
		if err != nil || u.Scheme == "" || u.Host == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
			panic("invalid trusted origin " + origin + ": must be scheme://host[:port]")
		}

		m.trusted[strings.ToLower(origin)] = struct{}{}
	}
}

// CrossOriginProtection refuses state-changing requests that the browser marks as sent by a page
// of another origin (see azugo.Context.IsCrossOrigin) with a CrossOriginError. GET, HEAD and
// OPTIONS requests pass, and so do requests without Sec-Fetch-Site and Origin, which come from
// non-browser clients.
func CrossOriginProtection(opts ...CrossOriginOption) azugo.RequestHandlerFunc {
	m := &crossOriginProtection{trusted: make(map[string]struct{})}

	for _, opt := range opts {
		opt.apply(m)
	}

	return func(next azugo.RequestHandler) azugo.RequestHandler {
		return func(ctx *azugo.Context) {
			method := ctx.Method()
			if method.IsGet() || method.IsHead() || method.IsOptions() || !ctx.IsCrossOrigin() {
				next(ctx)

				return
			}

			if _, ok := m.trusted[strings.ToLower(ctx.Header.Get(http.HeaderOrigin))]; ok {
				next(ctx)

				return
			}

			ctx.Error(CrossOriginError{})
		}
	}
}
