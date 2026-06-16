package middleware

import (
	"errors"
	"testing"
	"time"

	"azugo.io/azugo"
	"azugo.io/azugo/config"
	"azugo.io/azugo/token"
	"azugo.io/azugo/user"
	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

func TestRateLimit(t *testing.T) {
	a := azugo.NewTestApp()

	a.Use(RateLimit(&config.RateLimit{
		Enabled:  true,
		Strategy: "fixed-window",
		Limit:    1,
		Window:   time.Minute,
	}))

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.StatusCode(fasthttp.StatusOK)
		ctx.Text("ok")
	})

	a.Start(t)
	defer a.Stop()

	resp, err := a.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
	qt.Check(t, qt.Not(qt.Equals(string(resp.Header.Peek("RateLimit-Limit")), "")))
	qt.Check(t, qt.Not(qt.Equals(string(resp.Header.Peek("RateLimit-Remaining")), "")))
	qt.Check(t, qt.Not(qt.Equals(string(resp.Header.Peek("RateLimit-Reset")), "")))
	qt.Check(t, qt.Not(qt.Equals(string(resp.Header.Peek("RateLimit-Policy")), "")))
	fasthttp.ReleaseResponse(resp)

	resp, err = a.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusTooManyRequests))
	qt.Check(t, qt.Not(qt.Equals(string(resp.Header.Peek("Retry-After")), "")))
	qt.Check(t, qt.Not(qt.Equals(string(resp.Header.Peek("RateLimit-Remaining")), "")))
	fasthttp.ReleaseResponse(resp)
}

func TestRateLimitPlainOptionsIsLimited(t *testing.T) {
	a := azugo.NewTestApp()

	a.Use(RateLimit(&config.RateLimit{
		Enabled:  true,
		Strategy: "fixed-window",
		Limit:    1,
		Window:   time.Minute,
	}))

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.StatusCode(fasthttp.StatusOK)
		ctx.Text("ok")
	})

	a.Start(t)
	defer a.Stop()

	// A plain OPTIONS request (without CORS preflight headers) consumes quota.
	resp, err := a.TestClient().Options("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Not(qt.Equals(resp.StatusCode(), fasthttp.StatusTooManyRequests)))
	fasthttp.ReleaseResponse(resp)

	// The next plain OPTIONS request is rejected by the limiter.
	resp, err = a.TestClient().Options("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusTooManyRequests))
	fasthttp.ReleaseResponse(resp)
}

func TestRateLimitCORSPreflightBypassesLimit(t *testing.T) {
	a := azugo.NewTestApp()

	// CORS runs before the limiter and flags preflights from allowed origins.
	a.Use(CORS((&azugo.CORSOptions{}).SetOrigins("https://example.com")))
	a.Use(RateLimit(&config.RateLimit{
		Enabled:  true,
		Strategy: "fixed-window",
		Limit:    1,
		Window:   time.Minute,
	}))

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.StatusCode(fasthttp.StatusOK)
		ctx.Text("ok")
	})

	a.Start(t)
	defer a.Stop()

	// Genuine CORS preflight requests from an allowed origin are exempt, so
	// repeated preflights are never rejected even though the limit is one
	// request per window.
	for range 3 {
		resp, err := a.TestClient().Options("/test",
			a.TestClient().WithHeader(headerOrigin, "https://example.com"),
			a.TestClient().WithHeader(headerRequestMethod, fasthttp.MethodGet),
		)
		qt.Assert(t, qt.IsNil(err))
		qt.Check(t, qt.Not(qt.Equals(resp.StatusCode(), fasthttp.StatusTooManyRequests)))
		fasthttp.ReleaseResponse(resp)
	}
}

func TestRateLimitCORSPreflightDisallowedOriginIsLimited(t *testing.T) {
	a := azugo.NewTestApp()

	a.Use(CORS((&azugo.CORSOptions{}).SetOrigins("https://example.com")))
	a.Use(RateLimit(&config.RateLimit{
		Enabled:  true,
		Strategy: "fixed-window",
		Limit:    1,
		Window:   time.Minute,
	}))

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.StatusCode(fasthttp.StatusOK)
		ctx.Text("ok")
	})

	a.Start(t)
	defer a.Stop()

	// A preflight-looking request from a non allow-listed origin is not flagged
	// by CORS, so it cannot be used to bypass the limit by spoofing headers.
	resp, err := a.TestClient().Options("/test",
		a.TestClient().WithHeader(headerOrigin, "https://evil.example"),
		a.TestClient().WithHeader(headerRequestMethod, fasthttp.MethodGet),
	)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Not(qt.Equals(resp.StatusCode(), fasthttp.StatusTooManyRequests)))
	fasthttp.ReleaseResponse(resp)

	resp, err = a.TestClient().Options("/test",
		a.TestClient().WithHeader(headerOrigin, "https://evil.example"),
		a.TestClient().WithHeader(headerRequestMethod, fasthttp.MethodGet),
	)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusTooManyRequests))
	fasthttp.ReleaseResponse(resp)
}

func TestRateLimitPreservesCORSHeadersOnReject(t *testing.T) {
	a := azugo.NewTestApp()

	a.Use(CORS((&azugo.CORSOptions{}).SetOrigins("https://example.com")))
	a.Use(RateLimit(&config.RateLimit{
		Enabled:  true,
		Strategy: "fixed-window",
		Limit:    1,
		Window:   time.Minute,
	}))

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.StatusCode(fasthttp.StatusOK)
		ctx.Text("ok")
	})

	a.Start(t)
	defer a.Stop()

	origin := a.TestClient().WithHeader(headerOrigin, "https://example.com")

	// First cross-origin request succeeds and carries the CORS origin header.
	resp, err := a.TestClient().Get("/test", origin)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
	qt.Check(t, qt.Equals(string(resp.Header.Peek(headerAllowOrigin)), "https://example.com"))
	fasthttp.ReleaseResponse(resp)

	// The rate-limited response must keep the CORS header so the browser can
	// read the 429 instead of having it blocked by the same-origin policy.
	resp, err = a.TestClient().Get("/test", origin)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusTooManyRequests))
	qt.Check(t, qt.Equals(string(resp.Header.Peek(headerAllowOrigin)), "https://example.com"))
	fasthttp.ReleaseResponse(resp)
}

func TestRateLimitJSONErrorResponse(t *testing.T) {
	a := azugo.NewTestApp()

	a.Use(RateLimit(&config.RateLimit{
		Enabled:  true,
		Strategy: "fixed-window",
		Limit:    1,
		Window:   time.Minute,
	}))

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.StatusCode(fasthttp.StatusOK)
		ctx.Text("ok")
	})

	a.Start(t)
	defer a.Stop()

	resp, err := a.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
	fasthttp.ReleaseResponse(resp)

	// A JSON client receives a content-negotiated error body, and the RateLimit
	// response headers survive the framework error handling.
	resp, err = a.TestClient().Get("/test", a.TestClient().WithHeader("Accept", "application/json"))
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusTooManyRequests))
	qt.Check(t, qt.StringContains(string(resp.Header.ContentType()), "application/json"))
	qt.Check(t, qt.StringContains(string(resp.Body()), "rate limit exceeded"))
	qt.Check(t, qt.Not(qt.Equals(string(resp.Header.Peek("Retry-After")), "")))
	qt.Check(t, qt.Not(qt.Equals(string(resp.Header.Peek("RateLimit-Reset")), "")))
	fasthttp.ReleaseResponse(resp)
}

func TestRateLimitCustomErrorHandler(t *testing.T) {
	a := azugo.NewTestApp()

	a.RouterOptions().ErrorHandler = func(ctx *azugo.Context, err error) bool {
		var rle *RateLimitError
		if !errors.As(err, &rle) {
			return false
		}

		ctx.StatusCode(fasthttp.StatusTooManyRequests)
		ctx.Text("custom: slow down")

		return true
	}

	a.Use(RateLimit(&config.RateLimit{
		Enabled:  true,
		Strategy: "fixed-window",
		Limit:    1,
		Window:   time.Minute,
	}))

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.StatusCode(fasthttp.StatusOK)
		ctx.Text("ok")
	})

	a.Start(t)
	defer a.Stop()

	resp, err := a.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
	fasthttp.ReleaseResponse(resp)

	// The exported RateLimitError lets a custom handler fully override the response.
	resp, err = a.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusTooManyRequests))
	qt.Check(t, qt.Equals(string(resp.Body()), "custom: slow down"))
	fasthttp.ReleaseResponse(resp)
}

func TestRateLimitDisabledIsPassthrough(t *testing.T) {
	a := azugo.NewTestApp()

	// A disabled (or zero-valued) configuration must not build a limiter, so
	// requests pass through instead of failing with a 500 on every request.
	a.Use(RateLimit(&config.RateLimit{Enabled: false}))

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.StatusCode(fasthttp.StatusOK)
		ctx.Text("ok")
	})

	a.Start(t)
	defer a.Stop()

	for range 3 {
		resp, err := a.TestClient().Get("/test")
		qt.Assert(t, qt.IsNil(err))
		qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
		qt.Check(t, qt.Equals(string(resp.Header.Peek("RateLimit-Limit")), ""))
		fasthttp.ReleaseResponse(resp)
	}
}

func TestRateLimitNilConfigIsPassthrough(t *testing.T) {
	a := azugo.NewTestApp()

	a.Use(RateLimit(nil))

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.StatusCode(fasthttp.StatusOK)
		ctx.Text("ok")
	})

	a.Start(t)
	defer a.Stop()

	resp, err := a.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
	fasthttp.ReleaseResponse(resp)
}

func TestRateLimitHeadersDisabled(t *testing.T) {
	a := azugo.NewTestApp()

	a.Use(RateLimit(&config.RateLimit{
		Enabled:  true,
		Strategy: "fixed-window",
		Limit:    1,
		Window:   time.Minute,
	}, DisableRateLimitHeaders()))

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.StatusCode(fasthttp.StatusOK)
		ctx.Text("ok")
	})

	a.Start(t)
	defer a.Stop()

	resp, err := a.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
	qt.Check(t, qt.Equals(string(resp.Header.Peek("RateLimit-Limit")), ""))
	qt.Check(t, qt.Equals(string(resp.Header.Peek("RateLimit-Remaining")), ""))
	qt.Check(t, qt.Equals(string(resp.Header.Peek("RateLimit-Reset")), ""))
	qt.Check(t, qt.Equals(string(resp.Header.Peek("RateLimit-Policy")), ""))
	fasthttp.ReleaseResponse(resp)

	resp, err = a.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusTooManyRequests))
	qt.Check(t, qt.Equals(string(resp.Header.Peek("Retry-After")), ""))
	fasthttp.ReleaseResponse(resp)
}

func TestRateLimitUsesUserIDWhenAuthorized(t *testing.T) {
	a := azugo.NewTestApp()

	a.Use(func(next azugo.RequestHandler) azugo.RequestHandler {
		return func(ctx *azugo.Context) {
			id := ctx.Header.Get("X-User-ID")
			if id != "" {
				ctx.SetUser(user.New(map[string]token.ClaimStrings{
					"sub": {id},
				}))
			}

			next(ctx)
		}
	})

	a.Use(RateLimit(&config.RateLimit{
		Enabled:  true,
		Strategy: "fixed-window",
		Limit:    1,
		Window:   time.Minute,
	}))

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.StatusCode(fasthttp.StatusOK)
		ctx.Text("ok")
	})

	a.Start(t)
	defer a.Stop()

	resp, err := a.TestClient().Get("/test", a.TestClient().WithHeader("X-User-ID", "u1"))
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
	fasthttp.ReleaseResponse(resp)

	resp, err = a.TestClient().Get("/test", a.TestClient().WithHeader("X-User-ID", "u2"))
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
	fasthttp.ReleaseResponse(resp)

	resp, err = a.TestClient().Get("/test", a.TestClient().WithHeader("X-User-ID", "u1"))
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusTooManyRequests))
	fasthttp.ReleaseResponse(resp)
}

func TestRateLimitCustomKeyGenerator(t *testing.T) {
	a := azugo.NewTestApp()

	a.Use(RateLimit(&config.RateLimit{
		Enabled:  true,
		Strategy: "fixed-window",
		Limit:    1,
		Window:   time.Minute,
	}, RateLimitKeyGenerator(func(_ *azugo.Context) (string, error) {
		return "custom-key", nil
	})))

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.StatusCode(fasthttp.StatusOK)
		ctx.Text("ok")
	})

	a.Start(t)
	defer a.Stop()

	resp, err := a.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
	fasthttp.ReleaseResponse(resp)

	resp, err = a.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusTooManyRequests))
	fasthttp.ReleaseResponse(resp)
}

func TestRateLimitCustomKeyGeneratorError(t *testing.T) {
	a := azugo.NewTestApp()

	a.Use(RateLimit(&config.RateLimit{
		Enabled:  true,
		Strategy: "fixed-window",
		Limit:    1,
		Window:   time.Minute,
	}, RateLimitKeyGenerator(func(_ *azugo.Context) (string, error) {
		return "", errors.New("key generator failed")
	})))

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.StatusCode(fasthttp.StatusOK)
		ctx.Text("ok")
	})

	a.Start(t)
	defer a.Stop()

	resp, err := a.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusInternalServerError))
	fasthttp.ReleaseResponse(resp)
}

func TestRateLimitNameIsolation(t *testing.T) {
	cfg := &config.RateLimit{
		Enabled:  true,
		Strategy: "fixed-window",
		Limit:    1,
		Window:   time.Minute,
	}

	fixedKey := RateLimitKeyGenerator(func(_ *azugo.Context) (string, error) {
		return "same-key", nil
	})

	// Two separate apps simulate independent route groups with different limiter names.
	// Use() applies globally so per-group isolation is tested via separate apps.
	appA := azugo.NewTestApp()
	appA.Use(RateLimit(cfg, RateLimitName("isolation-a"), fixedKey))
	appA.Get("/test", func(ctx *azugo.Context) { ctx.StatusCode(fasthttp.StatusOK); ctx.Text("ok") })
	appA.Start(t)
	defer appA.Stop()

	appB := azugo.NewTestApp()
	appB.Use(RateLimit(cfg, RateLimitName("isolation-b"), fixedKey))
	appB.Get("/test", func(ctx *azugo.Context) { ctx.StatusCode(fasthttp.StatusOK); ctx.Text("ok") })
	appB.Start(t)
	defer appB.Stop()

	// Exhaust group-a's quota.
	resp, err := appA.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
	fasthttp.ReleaseResponse(resp)

	resp, err = appA.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusTooManyRequests))
	fasthttp.ReleaseResponse(resp)

	// group-b must still have its own full quota despite the same key string.
	resp, err = appB.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
	fasthttp.ReleaseResponse(resp)
}
