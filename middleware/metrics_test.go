package middleware

import (
	"bytes"
	"net"
	"net/netip"
	"strings"
	"testing"

	"azugo.io/azugo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		ctx.StatusCode(fasthttp.StatusOK).Text("Hello, world!")
	})
	a.Get("/stream", func(ctx *azugo.Context) {
		ctx.ContentType("application/json")
		ctx.Header.Set("Content-Length", "4")
		ctx.Response().SetBodyStream(bytes.NewReader([]byte("test")), 4)
	})

	a.Start(t)
	defer a.Stop()

	resp, err := a.TestClient().Get("/")
	require.NoError(t, err)
	fasthttp.ReleaseResponse(resp)

	resp, err = a.TestClient().Get("/test")
	require.NoError(t, err)
	fasthttp.ReleaseResponse(resp)

	resp, err = a.TestClient().Get("/stream")
	require.NoError(t, err)
	fasthttp.ReleaseResponse(resp)

	resp, err = a.TestClient().Get("/skip")
	require.NoError(t, err)
	fasthttp.ReleaseResponse(resp)

	resp, err = a.TestClient().Get("/metrics")
	require.NoError(t, err)
	i := strings.Index(string(resp.Body()), "requests_total")
	fasthttp.ReleaseResponse(resp)

	assert.True(t, i > -1, "metrics handler not returning expected metrics")
}
