package middleware

import (
	"bytes"
	"net"
	"strings"

	"azugo.io/azugo"
	"azugo.io/azugo/internal/utils"

	"github.com/valyala/bytebufferpool"
)

const (
	xForwardedFor = "X-Forwarded-For"
	xRealIP       = "X-Real-IP"
)

var xForwardedForSep = []byte(", ")

func realIP(ctx *azugo.Context) net.IP {
	if xrip := ctx.Header.Get(xRealIP); len(xrip) > 0 {
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
		for i := ctx.App().RouterOptions.ProxyOptions.ForwardLimit; i > 0; i-- {
			if p > 0 {
				xff = xff[:p-2]
			}
			p = bytes.LastIndex(xff, []byte(", "))
			if p < 0 {
				p = 0
				break
			} else {
				p += 2
			}
		}
		if ip := net.ParseIP(utils.B2S(xff[p:])); ip != nil {
			return ip
		}
	}
	return nil
}

func realIPOrForwardedFor(ctx *azugo.Context) net.Addr {
	remoteAddr := ctx.Context().RemoteAddr()
	_, ok := remoteAddr.(*net.TCPAddr)
	if !ok {
		return remoteAddr
	}
	if !ctx.IsTrustedProxy() {
		return remoteAddr
	}
	if ip := realIP(ctx); ip != nil {
		remoteAddr = &net.TCPAddr{
			IP: ip,
		}
	} else if ip := forwardedFor(ctx); ip != nil {
		remoteAddr = &net.TCPAddr{
			IP: ip,
		}
	}
	return remoteAddr
}

// RealIP middleware updates request remote IP address based on X-Real-IP and X-Forwarded-For headers.
func RealIP(next azugo.RequestHandler) azugo.RequestHandler {
	return func(ctx *azugo.Context) {
		ctx.Context().SetRemoteAddr(realIPOrForwardedFor(ctx))
		next(ctx)
	}
}
