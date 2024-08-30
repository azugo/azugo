package middleware

import (
	"bytes"
	"net"
	"strings"

	"azugo.io/azugo"
	"azugo.io/azugo/internal/utils"

	"github.com/valyala/bytebufferpool"
	"go.uber.org/zap"
)

const (
	xForwardedFor = "X-Forwarded-For"
)

var xForwardedForSep = []byte(", ")

func realIP(ctx *azugo.Context, header string) net.IP {
	if xrip := ctx.Header.Get(header); len(xrip) > 0 {
		return net.ParseIP(xrip)
	}

	return nil
}

func forwardedFor(ctx *azugo.Context) net.IP {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	ctx.Context().Request.Header.VisitAllInOrder(func(key, value []byte) {
		if strings.EqualFold(utils.B2S(key), xForwardedFor) {
			if buf.Len() > 0 {
				_, _ = buf.Write(xForwardedForSep)
			}

			_, _ = buf.Write(value)
		}
	})

	xff := buf.Bytes()
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
		if strings.EqualFold(header, xForwardedFor) {
			if ip := forwardedFor(ctx); ip != nil {
				return &net.TCPAddr{
					IP: ip,
				}
			}
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
