package middleware

import (
	"azugo.io/azugo"
)

const (
	headerOrigin           string = "Origin"
	headerAllowOrigin      string = "Access-Control-Allow-Origin"
	headerAllowMethods     string = "Access-Control-Allow-Methods"
	headerAllowHeaders     string = "Access-Control-Allow-Headers"
	headerAllowCredentials string = "Access-Control-Allow-Credentials"
)

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

			ctx.Header.Set(headerAllowOrigin, origin)
			ctx.Header.Set(headerAllowMethods, opts.Methods())
			ctx.Header.Set(headerAllowHeaders, opts.Headers())

			if opts.AllowCredentials() {
				ctx.Header.Set(headerAllowCredentials, "true")
			}

			if h != nil {
				h(ctx)
			}
		}
	}
}
