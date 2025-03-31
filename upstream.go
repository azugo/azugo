package azugo

import (
	"bytes"
	"crypto/tls"
	"net/url"
	"strings"

	"azugo.io/azugo/internal/proxy"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

type proxyUpstream struct {
	Scheme  []byte
	Host    []byte
	Path    []byte
	BaseURL []byte
}

// Proxy is the proxy handler.
type Proxy struct {
	client        *fasthttp.Client
	options       *proxyOptions
	upstreamIndex uint
}

// ProxyOption is a proxy option.
type ProxyOption interface {
	apply(opts *proxyOptions)
}

type proxyOptions struct {
	BasePath           string
	InsecureSkipVerify bool
	BodyRewriter       *proxy.BodyRewriter
	Upstream           []*proxyUpstream
}

// ProxyUpstreamInsecureSkipVerify skips TLS certificate verification for upstream request.
type ProxyUpstreamInsecureSkipVerify bool

func (o ProxyUpstreamInsecureSkipVerify) apply(opts *proxyOptions) {
	opts.InsecureSkipVerify = bool(o)
}

type bodyRewriterRule struct {
	from, to string
}

func (o *bodyRewriterRule) apply(opts *proxyOptions) {
	opts.BodyRewriter.AddReplace([]byte(o.from), []byte(o.to))
}

// ProxyUpstreamBodyReplaceText replaces text in the response body.
func ProxyUpstreamBodyReplaceText(from, to string) ProxyOption {
	return &bodyRewriterRule{from, to}
}

// ProxyUpstreamBodyReplaceURL replaces URL in the response body.
type ProxyUpstreamBodyReplaceURL bool

func (o ProxyUpstreamBodyReplaceURL) apply(opts *proxyOptions) {
	opts.BodyRewriter.RewriteBaseURL = bool(o)
}

type proxyUpstreams []*proxyUpstream

func (o proxyUpstreams) apply(opts *proxyOptions) {
	opts.Upstream = append(opts.Upstream, o...)
}

// ProxyUpstream adds one or more upstream URLs.
func ProxyUpstream(upstream ...*url.URL) ProxyOption {
	upstr := make(proxyUpstreams, len(upstream))
	for i, v := range upstream {
		upstr[i] = &proxyUpstream{
			Scheme:  []byte(v.Scheme),
			Host:    []byte(v.Host),
			Path:    []byte(strings.TrimRight(v.Path, "/")),
			BaseURL: []byte(strings.TrimRight(v.String(), "/")),
		}
	}

	return upstr
}

// newUpstreamProxy creates a new proxy handler.
func (m *mux) newUpstreamProxy(basePath string, options ...ProxyOption) *Proxy {
	opt := &proxyOptions{
		BasePath:           strings.TrimRight(basePath, "/"),
		InsecureSkipVerify: true,
		BodyRewriter:       proxy.NewBodyRewriter(),
	}

	for _, option := range options {
		option.apply(opt)
	}

	return &Proxy{
		client: &fasthttp.Client{
			NoDefaultUserAgentHeader: true,
			TLSConfig: &tls.Config{
				//nolint:gosec
				InsecureSkipVerify: opt.InsecureSkipVerify,
			},
			ReadBufferSize:  m.app.ServerOptions.ResponseWriteBufferSize,
			WriteBufferSize: m.app.ServerOptions.RequestReadBufferSize,
		},
		options: opt,
	}
}

// Handler implements azugo.Handler to handle incoming request.
func (p *Proxy) Handler(ctx *Context) {
	if len(p.options.Upstream) == 0 {
		ctx.StatusCode(fasthttp.StatusBadGateway)
		ctx.Text(fasthttp.StatusMessage(fasthttp.StatusBadGateway))

		return
	}

	// This is not thread safe but we don't care if multiple requests goes to the same upstream.
	p.upstreamIndex = (p.upstreamIndex + 1) % uint(len(p.options.Upstream))
	upstream := p.options.Upstream[p.upstreamIndex]

	// Copy request from original
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	ctx.Request().CopyTo(req)

	resp := &ctx.Context().Response

	req.SetRequestURIBytes(upstream.Path)
	req.URI().SetSchemeBytes(upstream.Scheme)
	req.SetHostBytes(upstream.Host)
	// Downgrade HTTP/2 to HTTP/1.1
	if ctx.IsTLS() && bytes.Equal(req.Header.Protocol(), []byte("HTTP/2")) {
		req.Header.SetProtocolBytes([]byte("HTTP/1.1"))
	}

	proxy.StripHeaders(&req.Header)

	if err := p.client.Do(req, resp); err != nil {
		ctx.Log().With(zap.Error(err)).Warn("proxy upstream failed")
		ctx.StatusCode(fasthttp.StatusBadGateway)
		ctx.Text(fasthttp.StatusMessage(fasthttp.StatusBadGateway))

		return
	}

	proxy.StripHeaders(&resp.Header)
	proxy.RewriteCookies(ctx.IsTLS(), ctx.Host(), resp)

	if p.options.BodyRewriter != nil && p.options.BodyRewriter.Enabled() {
		p.options.BodyRewriter.RewriteResponse(append([]byte(ctx.BaseURL()), []byte(p.options.BasePath)...), upstream.BaseURL, resp)
	}
}
