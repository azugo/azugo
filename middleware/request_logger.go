package middleware

import (
	"bytes"
	"strconv"
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

func RequestLogger(next azugo.RequestHandler) azugo.RequestHandler {
	var logger *zap.Logger
	return func(ctx *azugo.Context) {
		if logger == nil {
			logger = ctx.App().Log().Named("http")
		}
		ctx.SetUserValue("log_request", true)

		t1 := time.Now()

		next(ctx)

		ns := time.Since(t1).Nanoseconds()

		if !ctx.UserValue("log_request").(bool) {
			return
		}

		method := ctx.Method()
		path := ctx.Path()
		cleanedPath := path
		basePath := ctx.BasePath()
		if len(basePath) > 0 && len(basePath) < len(path) && basePath == path[:len(basePath)] {
			cleanedPath = path[len(basePath):]
		}

		query := ctx.Request().URI().QueryString()

		referer := ctx.Header.Get("Referer")
		userAgent := ctx.Header.Get("User-Agent")

		msg := bytebufferpool.Get()
		defer bytebufferpool.Put(msg)

		// Remote IP address
		_, _ = msg.WriteString(ctx.IP().String())
		// TODO: what is this?
		_, _ = msg.WriteString(" - -")

		// Request time
		_, _ = msg.Write([]byte(" ["))
		_, _ = msg.WriteString(t1.Format("02/Jan/2006:15:04:05 -0700"))
		_, _ = msg.Write([]byte("] \""))

		// Method
		_, _ = msg.WriteString(method)
		// Path
		_ = msg.WriteByte(' ')
		_, _ = msg.WriteString(cleanedPath)
		// Query string
		if len(query) > 0 {
			_ = msg.WriteByte('?')
			_, _ = msg.Write(query)
		}
		// HTTP protocol
		_ = msg.WriteByte(' ')
		_, _ = msg.Write(ctx.Context().Response.Header.Protocol())
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

		fields := make([]zap.Field, 0, 10)

		// Request
		fields = append(fields,
			zap.String("http.version", utils.B2S(ctx.Context().Request.Header.Protocol())),
			zap.String("http.request.id", ctx.ID()),
			zap.String("http.request.method", method),
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
		if q := u.QueryString(); len(q) > 0 {
			fields = append(fields, zap.String("url.query", utils.B2S(q)))
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
			zap.String("source.ip", ctx.IP().String()),
		)

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
