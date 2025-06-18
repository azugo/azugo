package middleware

import (
	"azugo.io/azugo"
)

// CORS is a middleware for handling CORS requests.
func CORS(opts *azugo.CORSOptions) func(azugo.RequestHandler) azugo.RequestHandler {
	return func(h azugo.RequestHandler) azugo.RequestHandler {
		return func(ctx *azugo.Context) {
			origin := ctx.Header.Get(azugo.HeaderOrigin)
			if len(origin) == 0 || !opts.ValidOrigin(origin) {
				if h != nil {
					h(ctx)
				}

				return
			}

			azugo.SetCORSHeaders(ctx, opts, origin)

			if h != nil {
				h(ctx)
			}
		}
	}
}
