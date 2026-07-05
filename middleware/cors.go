// Package middleware provides HTTP middleware for the azugo framework.
package middleware

import (
	"azugo.io/azugo"

	"azugo.io/core/http"
)

const userValueCORSPreflight = "__cors_preflight"

// CORS is a middleware for handling CORS requests.
func CORS(opts *azugo.CORSOptions) func(azugo.RequestHandler) azugo.RequestHandler {
	return func(h azugo.RequestHandler) azugo.RequestHandler {
		return func(ctx *azugo.Context) {
			origin := ctx.Header.Get(http.HeaderOrigin)
			if len(origin) == 0 || !opts.ValidOrigin(origin) {
				if h != nil {
					h(ctx)
				}

				return
			}

			ctx.Header.SetAlways(http.HeaderAccessControlAllowOrigin, origin)
			ctx.Header.SetAlways(http.HeaderAccessControlAllowMethods, opts.Methods())
			ctx.Header.SetAlways(http.HeaderAccessControlAllowHeaders, opts.Headers())

			if opts.AllowCredentials() {
				ctx.Header.SetAlways(http.HeaderAccessControlAllowCredentials, "true")
			}

			if ctx.Method() == http.MethodOptions && ctx.Header.Get(http.HeaderAccessControlRequestMethod) != "" {
				ctx.SetUserValue(userValueCORSPreflight, true)
			}

			if h != nil {
				h(ctx)
			}
		}
	}
}
