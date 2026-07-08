package middleware

import (
	"bytes"
	"os"
	"strconv"
	"sync"
	"time"

	"azugo.io/azugo"
	"azugo.io/azugo/internal/utils"

	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

var (
	protocolHTTP  = []byte("http")
	protocolHTTPS = []byte("https")
)

// RequestLogger is a middleware that logs HTTP requests.
func RequestLogger(next azugo.RequestHandler) azugo.RequestHandler {
	var (
		init   sync.Once
		logger *zap.Logger
	)

	enabled := os.Getenv("ACCESS_LOG_ENABLED") == "" || os.Getenv("ACCESS_LOG_ENABLED") == "true"

	return func(ctx *azugo.Context) {
		init.Do(func() {
			logger = ctx.App().Log().Named("http")
		})
		ctx.SetUserValue("__log_request", enabled)

		next(ctx)

		ns := time.Since(ctx.Time()).Nanoseconds()

		if ctx.IsSkipRequestLog() {
			return
		}

		method := ctx.Method()
		path := ctx.Path()
		cleanedPath := path

		basePath := ctx.BasePath()
		if len(basePath) > 0 && len(basePath) < len(path) && basePath == path[:len(basePath)] {
			cleanedPath = path[len(basePath):]
		}

		query := ctx.Query.Raw()

		referer := ctx.Referer()
		userAgent := ctx.UserAgent()

		remoteIP := ctx.IP().String()

		msg := bytebufferpool.Get()
		defer bytebufferpool.Put(msg)

		// Remote IP address
		_, _ = msg.WriteString(remoteIP)
		// TODO: what is this?
		_, _ = msg.WriteString(" - -")

		// Request time
		_, _ = msg.Write([]byte(" ["))
		msg.B = ctx.Time().AppendFormat(msg.B, "02/Jan/2006:15:04:05 -0700")
		_, _ = msg.Write([]byte("] \""))

		// Method
		_, _ = msg.WriteString(method.String())
		// Path
		_ = msg.WriteByte(' ')
		_, _ = msg.WriteString(cleanedPath)
		// Query string
		if len(query) > 0 {
			_ = msg.WriteByte('?')
			_, _ = msg.WriteString(query)
		}
		// HTTP protocol
		_ = msg.WriteByte(' ')
		_, _ = msg.Write(ctx.Response().Header.Protocol())
		_ = msg.WriteByte('"')

		// Status Code
		_ = msg.WriteByte(' ')
		_, _ = msg.WriteString(strconv.Itoa(ctx.Response().StatusCode()))

		// Response body size
		_ = msg.WriteByte(' ')
		_, _ = msg.WriteString(strconv.Itoa(ctx.Response().Header.ContentLength()))

		// Referrer
		_, _ = msg.Write([]byte(" \""))
		if len(referer) > 0 {
			_, _ = msg.WriteString(referer)
		} else {
			_ = msg.WriteByte('-')
		}

		_ = msg.WriteByte('"')

		// User agent
		_, _ = msg.Write([]byte(" \""))
		if len(userAgent) > 0 {
			_, _ = msg.WriteString(userAgent)
		} else {
			_ = msg.WriteByte('-')
		}

		_ = msg.WriteByte('"')

		ctxFields := ctx.LogFields()

		fields := make([]zap.Field, 0, 20+len(ctxFields))

		// Request
		fields = append(fields,
			zap.String("http.version", ctx.Protocol()),
			zap.String("http.request.id", ctx.ID()),
			zap.String("http.request.method", method.String()),
		)

		if len(referer) > 0 {
			fields = append(fields, zap.String("http.request.referer", referer))
		}

		if ct := ctx.Request().Header.ContentType(); len(ct) > 0 {
			fields = append(fields, zap.String("http.request.mime_type", utils.B2S(ct)))
		}

		if len(userAgent) > 0 {
			fields = append(fields, zap.String("user_agent.original", userAgent))
		}

		// URL
		u := ctx.Request().URI()

		scheme := u.Scheme()
		if bytes.Equal(scheme, protocolHTTP) && ctx.IsTLS() {
			scheme = protocolHTTPS
		} else if bytes.Equal(scheme, protocolHTTPS) && !ctx.IsTLS() {
			scheme = protocolHTTP
		}

		fields = append(fields,
			zap.String("url.full", buildFullURI(ctx, cleanedPath, u)),
			zap.String("url.original", utils.B2S(u.Path())),
			zap.String("url.scheme", utils.B2S(scheme)),
			zap.String("url.domain", utils.B2S(u.Host())),
			zap.String("url.path", cleanedPath),
		)

		if usr := u.Username(); len(usr) > 0 {
			fields = append(fields, zap.String("url.username", utils.B2S(usr)))
		}

		if len(query) > 0 {
			fields = append(fields, zap.String("url.query", query))
		}

		if h := u.Hash(); len(h) > 0 {
			fields = append(fields, zap.String("url.fragment", utils.B2S(h)))
		}

		// Response
		fields = append(fields,
			zap.Int("http.response.status_code", ctx.Response().StatusCode()),
		)

		if ct := ctx.Response().Header.ContentType(); len(ct) > 0 {
			fields = append(fields, zap.String("http.response.mime_type", utils.B2S(ct)))
		}

		// Event
		fields = append(fields,
			zap.String("event.action", "http-request"),
			zap.String("event.category", "web"),
			zap.Int64("event.duration", ns),
		)

		// Source
		fields = append(fields,
			zap.String("source.ip", remoteIP),
		)

		for _, f := range ctxFields {
			duplicate := false

			for i := range fields {
				if fields[i].Key == f.Key {
					duplicate = true

					break
				}
			}

			if !duplicate {
				fields = append(fields, f)
			}
		}

		logger.Info(msg.String(), fields...)
	}
}

func buildFullURI(ctx *azugo.Context, path string, u *fasthttp.URI) string {
	fullURI := bytebufferpool.Get()
	defer bytebufferpool.Put(fullURI)

	_, _ = fullURI.WriteString(ctx.BaseURL())
	_, _ = fullURI.WriteString(path)

	if q := u.QueryString(); len(q) > 0 {
		_ = fullURI.WriteByte('?')
		_, _ = fullURI.Write(q)
	}

	if h := u.Hash(); len(h) > 0 {
		_ = fullURI.WriteByte('#')
		_, _ = fullURI.Write(h)
	}

	return fullURI.String()
}
