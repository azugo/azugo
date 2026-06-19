package server

import (
	"testing"

	"azugo.io/azugo"
	"azugo.io/azugo/config"
	"azugo.io/azugo/middleware"
	"github.com/go-quicktest/qt"
	"github.com/spf13/cobra"
	"github.com/valyala/fasthttp"
)

// newRateLimitedApp builds an app through server.New with the rate limiter
// enabled via configuration (limit of one request per minute) so that the
// automatic middleware wiring in New is exercised end-to-end.
func newRateLimitedApp(t *testing.T, opts ...Option) *azugo.TestApp {
	t.Helper()

	t.Setenv("RATELIMIT_ENABLED", "true")
	t.Setenv("RATELIMIT_STRATEGY", "fixed-window")
	t.Setenv("RATELIMIT_LIMIT", "1")
	t.Setenv("RATELIMIT_WINDOW", "1m")

	opts = append(opts, Options{
		AppName:       "Azugo TestApp",
		Configuration: config.New(),
	})

	a, err := New(&cobra.Command{Use: "test"}, opts...)
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.IsTrue(a.Config().RateLimit.Enabled))

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.StatusCode(fasthttp.StatusOK)
		ctx.Text("ok")
	})

	return azugo.NewTestApp(a)
}

func TestAutoRateLimitEnabled(t *testing.T) {
	ta := newRateLimitedApp(t)

	ta.Start(t)
	defer ta.Stop()

	// First request is within the limit.
	resp, err := ta.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
	fasthttp.ReleaseResponse(resp)

	// Second request is rejected by the auto-wired rate limit middleware.
	resp, err = ta.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusTooManyRequests))
	fasthttp.ReleaseResponse(resp)
}

func TestAutoRateLimitResolver(t *testing.T) {
	// The configured limit is 1, but the resolver raises it to 5 for the auto
	// middleware, proving RateLimitOptions reach it.
	ta := newRateLimitedApp(t, RateLimitOptions(
		middleware.RateLimitResolver(func(_ *azugo.Context) int { return 5 }),
	))

	ta.Start(t)
	defer ta.Stop()

	for range 5 {
		resp, err := ta.TestClient().Get("/test")
		qt.Assert(t, qt.IsNil(err))
		qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
		qt.Check(t, qt.Equals(string(resp.Header.Peek("RateLimit-Limit")), "5"))
		fasthttp.ReleaseResponse(resp)
	}

	resp, err := ta.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusTooManyRequests))
	fasthttp.ReleaseResponse(resp)
}

func TestRateLimitInvalidConfigFailsLoad(t *testing.T) {
	// An enabled but invalid rate limit configuration must fail app loading
	// rather than building a limiter that errors on every request.
	t.Setenv("RATELIMIT_ENABLED", "true")
	t.Setenv("RATELIMIT_STRATEGY", "bogus-strategy")

	_, err := New(&cobra.Command{Use: "test"}, Options{
		AppName:       "Azugo TestApp",
		Configuration: config.New(),
	})
	qt.Check(t, qt.IsNotNil(err))
}

func TestDisableAutoRateLimit(t *testing.T) {
	ta := newRateLimitedApp(t, DisableAutoRateLimit())

	ta.Start(t)
	defer ta.Stop()

	// The rate limiter is enabled in configuration, but DisableAutoRateLimit
	// keeps the middleware out of the global stack, so no request is rejected.
	resp, err := ta.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
	fasthttp.ReleaseResponse(resp)

	resp, err = ta.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
	fasthttp.ReleaseResponse(resp)
}
