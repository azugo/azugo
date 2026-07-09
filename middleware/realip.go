package middleware

import (
	"bytes"
	"net"
	"strings"

	"azugo.io/azugo"
	"azugo.io/azugo/internal/utils"

	"azugo.io/core/http"
	"github.com/valyala/bytebufferpool"
	"go.uber.org/zap"
)

var xForwardedForSep = []byte(", ")

func realIP(ctx *azugo.Context, header string) net.IP {
	if xrip := ctx.Header.Get(header); len(xrip) > 0 {
		return net.ParseIP(xrip)
	}

	return nil
}

func forwardedFor(ctx *azugo.Context) net.IP {
	values := ctx.Request().Header.PeekAll(http.HeaderXForwardedFor)

	var xff []byte

	switch len(values) {
	case 0:
		return nil
	case 1:
		xff = values[0]
	default:
		buf := bytebufferpool.Get()
		defer bytebufferpool.Put(buf)

		for i, v := range values {
			if i > 0 {
				_, _ = buf.Write(xForwardedForSep)
			}

			_, _ = buf.Write(v)
		}

		xff = buf.Bytes()
	}

	if len(xff) > 0 {
		p := 0
		c := 0

		for i := ctx.RouterOptions().Proxy.ForwardLimit; i > 0; i-- {
			if p-c > 0 {
				xff = xff[:p-c]
			}

			p = bytes.LastIndex(xff, []byte(","))
			if p < 0 {
				p = 0

				break
			}

			p++
			c = 1

			for ; p < len(xff) && xff[p] == ' '; p++ {
				// skip spaces
				c++
			}
		}

		for p < len(xff) && xff[p] == ' ' {
			// skip spaces
			p++
		}

		if ip := net.ParseIP(utils.B2S(xff[p:])); ip != nil {
			return ip
		}
	}

	return nil
}

func realIPOrForwardedFor(ctx *azugo.Context) net.Addr {
	remoteAddr := ctx.Context().RemoteAddr()
	if !ctx.IsTrustedProxy() {
		return remoteAddr
	}

	for _, header := range ctx.RouterOptions().Proxy.TrustedHeaders {
		if strings.EqualFold(header, http.HeaderXForwardedFor) {
			if ip := forwardedFor(ctx); ip != nil {
				return &net.TCPAddr{
					IP: ip,
				}
			}

			continue
		}

		if ip := realIP(ctx, header); ip != nil {
			return &net.TCPAddr{
				IP: ip,
			}
		}
	}

	return remoteAddr
}

// RealIP middleware updates request remote IP address based on X-Real-IP and X-Forwarded-For headers.
func RealIP(next azugo.RequestHandler) azugo.RequestHandler {
	return func(ctx *azugo.Context) {
		ctx.Context().SetRemoteAddr(realIPOrForwardedFor(ctx))
		// Update logger with new source IP field
		_ = ctx.AddLogFields(zap.String("source.ip", ctx.IP().String()))
		next(ctx)
	}
}
