package middleware

import (
	"azugo.io/azugo"
)

const (
	headerOrigin       string = "Origin"
	headerAllowOrigin  string = "Access-Control-Allow-Origin"
	headerAllowMethods string = "Access-Control-Allow-Methods"
	headerAllowHeaders string = "Access-Control-Allow-Headers"
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
			if h != nil {
				h(ctx)
			}
		}
	}
}
