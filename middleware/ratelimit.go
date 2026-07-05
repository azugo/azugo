package middleware

import (
	"errors"
	"iter"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"azugo.io/azugo"
	"azugo.io/azugo/config"

	"azugo.io/core/http"
	"azugo.io/core/ratelimit"
)

var errEmptyRateLimitKey = errors.New("rate limit key is empty")

// RateLimitError is returned by the RateLimit middleware when a request exceeds
// the configured limit.
type RateLimitError struct {
	// Result is the rate limiter outcome that triggered the rejection.
	Result      ratelimit.Result
	emitHeaders bool
}

// Error implements the error interface.
func (*RateLimitError) Error() string {
	return "rate limit exceeded"
}

// SafeError returns a message that can be safely returned to the client.
func (*RateLimitError) SafeError() string {
	return "rate limit exceeded"
}

// StatusCode returns the HTTP status code for the rate limit error.
func (*RateLimitError) StatusCode() int {
	return http.StatusTooManyRequests
}

// ErrorHeaders returns the RateLimit response headers to set on the response.
func (e *RateLimitError) ErrorHeaders() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		if e.emitHeaders && e.Result.RetryAfter > 0 {
			yield(http.HeaderRetryAfter, formatSeconds(e.Result.RetryAfter))
		}
	}
}

type rateLimitMiddleware struct {
	config       *config.RateLimit
	name         string
	emitHeaders  bool
	keyGenerator func(ctx *azugo.Context) (string, error)
	limitFunc    func(ctx *azugo.Context) int
	defaultLimit int
	policyPrefix string

	mu      sync.Mutex
	limiter atomic.Pointer[ratelimit.Limiter]
}

// RateLimitOption configures the rate limit middleware.
type RateLimitOption interface {
	apply(opt *rateLimitMiddleware)
}

type disableRateLimitHeadersOption struct{}

func (o disableRateLimitHeadersOption) apply(opt *rateLimitMiddleware) {
	opt.emitHeaders = false
}

// DisableRateLimitHeaders disables ratelimit response headers.
func DisableRateLimitHeaders() RateLimitOption {
	return disableRateLimitHeadersOption{}
}

// RateLimitKeyGenerator is a function that generates a rate limit key for a
// request.
type RateLimitKeyGenerator func(ctx *azugo.Context) (string, error)

func (o RateLimitKeyGenerator) apply(opt *rateLimitMiddleware) {
	opt.keyGenerator = o
}

// RateLimitName sets the limiter name used for key namespacing in the cache
// backend. Defaults to "global".
type RateLimitName string

func (o RateLimitName) apply(opt *rateLimitMiddleware) {
	opt.name = string(o)
}

// RateLimitResolver allows to prived custom resolver for context based custom
// limits.
// If zero is returned default configured limit is used.
type RateLimitResolver func(ctx *azugo.Context) int

func (o RateLimitResolver) apply(opt *rateLimitMiddleware) {
	opt.limitFunc = o
}

// RateLimit applies a request rate limit per client.
//
// CORS preflight requests flagged by the CORS middleware are exempt.
func RateLimit(c *config.RateLimit, opts ...RateLimitOption) azugo.RequestHandlerFunc {
	if c == nil || !c.Enabled {
		return func(next azugo.RequestHandler) azugo.RequestHandler {
			return next
		}
	}

	m := &rateLimitMiddleware{
		config:       c,
		name:         "global",
		emitHeaders:  true,
		keyGenerator: defaultRateLimitKey,
	}

	for _, opt := range opts {
		opt.apply(m)
	}

	switch c.Strategy {
	case "fixed-window":
		m.defaultLimit = c.Limit
		m.policyPrefix = "fixed-window;w=" + formatSeconds(c.Window) + ";q="
	case "token-bucket":
		m.defaultLimit = c.Burst
		m.policyPrefix = "token-bucket;rate=" + strconv.FormatFloat(c.Rate, 'f', -1, 64) + ";burst="
	}

	return m.handler
}

func (m *rateLimitMiddleware) getLimiter(ctx *azugo.Context) (ratelimit.Limiter, error) {
	if l := m.limiter.Load(); l != nil {
		return *l, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if l := m.limiter.Load(); l != nil {
		return *l, nil
	}

	limiter, err := m.config.New(ctx.App().Cache(), m.name,
		ratelimit.Instrumenter(ctx.App().Instrumenter()))
	if err != nil {
		return nil, err
	}

	m.limiter.Store(&limiter)

	return limiter, nil
}

func (m *rateLimitMiddleware) handler(next azugo.RequestHandler) azugo.RequestHandler {
	return func(ctx *azugo.Context) {
		if v, _ := ctx.UserValue(userValueCORSPreflight).(bool); v {
			next(ctx)

			return
		}

		limiter, err := m.getLimiter(ctx)
		if err != nil {
			ctx.Error(err)

			return
		}

		key, err := m.keyGenerator(ctx)
		if err != nil {
			ctx.Error(err)

			return
		}

		if key == "" {
			ctx.Error(errEmptyRateLimitKey)

			return
		}

		limit := 0
		if m.limitFunc != nil {
			limit = m.limitFunc(ctx)
		}

		res, err := limiter.AllowLimit(ctx, key, limit)
		if err != nil {
			ctx.Error(err)

			return
		}

		effLimit := m.defaultLimit
		if limit > 0 {
			effLimit = limit
		}

		if m.emitHeaders {
			m.setHeaders(ctx, res, effLimit)
		}

		if !res.Allowed {
			ctx.Error(&RateLimitError{Result: res, emitHeaders: m.emitHeaders})

			return
		}

		next(ctx)
	}
}

func (m *rateLimitMiddleware) setHeaders(ctx *azugo.Context, res ratelimit.Result, effLimit int) {
	if effLimit > 0 {
		limitStr := strconv.Itoa(effLimit)

		ctx.Header.SetAlways(http.HeaderRateLimitLimit, limitStr)

		if m.policyPrefix != "" {
			ctx.Header.SetAlways(http.HeaderRateLimitPolicy, m.policyPrefix+limitStr)
		}
	}

	ctx.Header.SetAlways(http.HeaderRateLimitRemaining, strconv.Itoa(max(res.Remaining, 0)))
	ctx.Header.SetAlways(http.HeaderRateLimitReset, formatSeconds(time.Until(res.ResetAt)))
}

func defaultRateLimitKey(ctx *azugo.Context) (string, error) {
	if u := ctx.User(); u != nil && u.Authorized() {
		if id := u.ID(); id != "" {
			return "user:" + id, nil
		}
	}

	return "ip:" + ctx.IP().String(), nil
}

func formatSeconds(d time.Duration) string {
	if d <= 0 {
		return "0"
	}

	s := d / time.Second
	if d%time.Second != 0 {
		s++
	}

	return strconv.FormatInt(int64(s), 10)
}
