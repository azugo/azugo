package middleware

import (
	"time"

	"azugo.io/azugo"
)

// ReadTimeout gives the guarded routes duration to read the request body instead of the server ReadTimeout.
func ReadTimeout(d time.Duration) func(azugo.RequestHandler) azugo.RequestHandler {
	return func(h azugo.RequestHandler) azugo.RequestHandler {
		return func(ctx *azugo.Context) {
			if err := ctx.SetReadDeadline(time.Now().Add(d)); err != nil {
				ctx.Error(err)

				return
			}

			h(ctx)
		}
	}
}

// WriteTimeout gives the guarded routes duration to write the response instead of the server WriteTimeout.
func WriteTimeout(d time.Duration) func(azugo.RequestHandler) azugo.RequestHandler {
	return func(h azugo.RequestHandler) azugo.RequestHandler {
		return func(ctx *azugo.Context) {
			if err := ctx.SetWriteDeadline(time.Now().Add(d)); err != nil {
				ctx.Error(err)

				return
			}

			h(ctx)
		}
	}
}
