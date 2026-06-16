// Package middleware provides HTTP middleware for the azugo framework.
package middleware

import (
	"azugo.io/azugo"

	"github.com/valyala/fasthttp"
)

const (
	headerOrigin           string = "Origin"
	headerRequestMethod    string = "Access-Control-Request-Method"
	headerAllowOrigin      string = "Access-Control-Allow-Origin"
	headerAllowMethods     string = "Access-Control-Allow-Methods"
	headerAllowHeaders     string = "Access-Control-Allow-Headers"
	headerAllowCredentials string = "Access-Control-Allow-Credentials"
)

const userValueCORSPreflight = "__cors_preflight"

// CORS is a middleware for handling CORS requests.
func CORS(opts *azugo.CORSOptions) func(azugo.RequestHandler) azugo.RequestHandler {
	return func(h azugo.RequestHandler) azugo.RequestHandler {
		return func(ctx *azugo.Context) {
			origin := ctx.Header.Get(headerOrigin)
			if len(origin) == 0 || !opts.ValidOrigin(origin) {
				if h != nil {
					h(ctx)
				}

				return
			}

			ctx.Header.SetAlways(headerAllowOrigin, origin)
			ctx.Header.SetAlways(headerAllowMethods, opts.Methods())
			ctx.Header.SetAlways(headerAllowHeaders, opts.Headers())

			if opts.AllowCredentials() {
				ctx.Header.SetAlways(headerAllowCredentials, "true")
			}

			if ctx.Method() == fasthttp.MethodOptions && ctx.Header.Get(headerRequestMethod) != "" {
				ctx.SetUserValue(userValueCORSPreflight, true)
			}

			if h != nil {
				h(ctx)
			}
		}
	}
}
