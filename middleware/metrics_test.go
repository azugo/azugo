package middleware

import (
	"bytes"
	"net"
	"net/netip"
	"strings"
	"testing"

	"azugo.io/azugo"

	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

func TestMetricsHandler(t *testing.T) {
	a := azugo.NewTestApp()

	a.Use(func(next azugo.RequestHandler) azugo.RequestHandler {
		return func(ctx *azugo.Context) {
			ctx.Context().SetRemoteAddr(net.TCPAddrFromAddrPort(netip.MustParseAddrPort("1.1.1.1:30003")))
			next(ctx)
		}
	})

	opts := &a.MetricsOptions
	opts.SkipPaths = []string{"/skip"}
	opts.TrustedIPs = append(opts.TrustedIPs, net.IPv4(1, 1, 1, 1))
	a.Use(Metrics(azugo.DefaultMetricPath, MetricsSubsystem("subsystem")))

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.ContentType("application/json")
		ctx.StatusCode(fasthttp.StatusOK)
		ctx.Text("Hello, world!")
	})
	a.Get("/stream", func(ctx *azugo.Context) {
		ctx.ContentType("application/json")
		ctx.Header.Set("Content-Length", "4")
		ctx.Response().SetBodyStream(bytes.NewReader([]byte("test")), 4)
	})

	a.Start(t)
	defer a.Stop()

	resp, err := a.TestClient().Get("/")
	qt.Assert(t, qt.IsNil(err))
	fasthttp.ReleaseResponse(resp)

	resp, err = a.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	fasthttp.ReleaseResponse(resp)

	resp, err = a.TestClient().Get("/stream")
	qt.Assert(t, qt.IsNil(err))
	fasthttp.ReleaseResponse(resp)

	resp, err = a.TestClient().Get("/skip")
	qt.Assert(t, qt.IsNil(err))
	fasthttp.ReleaseResponse(resp)

	resp, err = a.TestClient().Get("/metrics")

	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))

	i := strings.Index(string(resp.Body()), "requests_total")
	fasthttp.ReleaseResponse(resp)

	qt.Check(t, qt.IsTrue(i > -1), qt.Commentf("metrics handler not returning expected metrics"))
}
