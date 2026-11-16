package azugo

import (
	"bufio"
	"fmt"
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

func TestProxyPreservesEscapedRequestPath(t *testing.T) {
	type receivedRequest struct {
		target string
		host   string
	}

	requests := make(chan receivedRequest, 1)
	upstream := httptest.NewServer(nethttp.HandlerFunc(func(_ nethttp.ResponseWriter, r *nethttp.Request) {
		requests <- receivedRequest{r.RequestURI, r.Host}
	}))
	defer upstream.Close()
	target, err := url.Parse(upstream.URL + "/api%2Fv1")
	qt.Assert(t, qt.IsNil(err))

	a := NewTestApp()
	a.Config().Server.Path = "/base"
	a.Start(t)
	defer a.Stop()
	a.Use(func(next RequestHandler) RequestHandler {
		return func(ctx *Context) {
			_ = ctx.Query.StringOptional("x")
			next(ctx)
		}
	})
	a.Proxy("/proxy", ProxyUpstream(target))
	a.Group("/group").Proxy("/proxy", ProxyUpstream(target))
	root, err := url.Parse(upstream.URL)
	qt.Assert(t, qt.IsNil(err))
	a.Proxy("/plain", ProxyUpstream(root))

	for _, tt := range []struct {
		path string
		want string
	}{
		{"/base/proxy", "/api%2Fv1"},
		{"/base/proxy/a%2Fb?x=1", "/api%2Fv1/a%2Fb?x=1"},
		{"/base/proxy/a%3Fb?x=%2523&x=2", "/api%2Fv1/a%3Fb?x=%2523&x=2"},
		{"/base/proxy/a%23b?x=1", "/api%2Fv1/a%23b?x=1"},
		{"/base/proxy/a%25b/%E2%82%AC", "/api%2Fv1/a%25b/%E2%82%AC"},
		{"/base/proxy/%2e%2e/a%2fb", "/api%2Fv1/%2e%2e/a%2fb"},
		{"/base/proxy/a//b/../c", "/api%2Fv1/a//b/../c"},
		{"/BASE/proxy/a%2Fb", "/api%2Fv1/a%2Fb"},
		{"/base/group/proxy/a%3Fb?x=1", "/api%2Fv1/a%3Fb?x=1"},
		{"http://different.example/base/proxy/a%2Fb?x=1", "/api%2Fv1/a%2Fb?x=1"},
		{"/base/plain//http://example.test/a%2Fb?x=1", "//http://example.test/a%2Fb?x=1"},
	} {
		t.Run(tt.path, func(t *testing.T) {
			conn, err := a.ln.Dial()
			qt.Assert(t, qt.IsNil(err))
			defer conn.Close()
			_, err = fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: test\r\nConnection: close\r\n\r\n", tt.path)
			qt.Assert(t, qt.IsNil(err))
			resp, err := nethttp.ReadResponse(bufio.NewReader(conn), nil)
			qt.Assert(t, qt.IsNil(err))
			defer resp.Body.Close()
			qt.Assert(t, qt.Equals(resp.StatusCode, nethttp.StatusOK))
			qt.Check(t, qt.Equals(<-requests, receivedRequest{tt.want, target.Host}))
		})
	}
}

func TestProxyRequestBodyStreams(t *testing.T) {
	type received struct {
		body   string
		length int64
	}

	bodies := make(chan received, 10)
	upstream := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		body, err := io.ReadAll(r.Body)
		if err == nil {
			bodies <- received{string(body), r.ContentLength}
		}
		w.WriteHeader(nethttp.StatusOK)
	}))
	defer upstream.Close()
	target, err := url.Parse(upstream.URL)
	qt.Assert(t, qt.IsNil(err))

	a := NewTestApp()
	a.Start(t)
	defer a.Stop()
	a.Use(func(next RequestHandler) RequestHandler {
		return func(ctx *Context) {
			ctx.SetMaxBodySize(6)
			if ctx.Path() == "/proxy/partial" {
				r, err := ctx.Body.Reader()
				if err != nil {
					ctx.Error(err)
					return
				}
				_, err = io.ReadFull(r, make([]byte, 1))
				if err != nil {
					ctx.Error(err)
					return
				}
			}
			next(ctx)
		}
	})
	a.Proxy("/proxy", ProxyUpstream(target))

	tests := []struct {
		name   string
		path   string
		reader io.Reader
		length int
		status int
		want   *received
	}{
		{"known-length", "/proxy/test", strings.NewReader("upload"), 6, 200, &received{"upload", 6}},
		{"chunked", "/proxy/test", strings.NewReader("upload"), -1, 200, &received{"upload", -1}},
		{"known-too-large", "/proxy/test", strings.NewReader("uploads"), 7, 413, nil},
		{"chunked-too-large", "/proxy/test", strings.NewReader("uploads"), -1, 413, nil},
		{"source-error", "/proxy/test", failingBodyReader{io.ErrUnexpectedEOF}, -1, 400, nil},
		{"truncated-known-length", "/proxy/test", strings.NewReader("abc"), 6, 400, nil},
		{"partially-consumed", "/proxy/partial", strings.NewReader("upload"), 6, 400, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw fasthttp.RequestCtx
			raw.Request.Header.SetMethod("POST")
			raw.Request.SetRequestURI(tt.path)
			raw.Request.SetBodyStream(tt.reader, tt.length)
			defer raw.Request.Reset()
			a.Handler(&raw)
			qt.Assert(t, qt.Equals(raw.Response.StatusCode(), tt.status), qt.Commentf("%s", raw.Response.Body()))
			if tt.want != nil {
				qt.Check(t, qt.Equals(<-bodies, *tt.want))
			}
		})
	}
}

func TestProxyRealStreamedUploads(t *testing.T) {
	payload := strings.Repeat("x", fasthttp.DefaultMaxRequestBodySize+17)
	bodies := make(chan string, 2)
	upstream := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read upstream body: %v", err)
		}
		bodies <- string(body)
	}))
	defer upstream.Close()
	target, err := url.Parse(upstream.URL)
	qt.Assert(t, qt.IsNil(err))

	a := NewTestApp()
	a.Start(t)
	defer a.Stop()
	streamed := make(chan bool, 2)
	a.Use(func(next RequestHandler) RequestHandler {
		return func(ctx *Context) {
			ctx.SetMaxBodySize(len(payload))
			streamed <- ctx.Request().IsBodyStream()
			next(ctx)
		}
	})
	a.Proxy("/proxy", ProxyUpstream(target))

	for _, length := range []int{len(payload), -1} {
		var req fasthttp.Request
		var resp fasthttp.Response
		req.Header.SetMethod("POST")
		req.SetRequestURI("http://localhost/proxy")
		req.SetBodyStream(strings.NewReader(payload), length)
		err = a.TestClient().client.Do(&req, &resp)
		qt.Assert(t, qt.IsNil(err))
		qt.Assert(t, qt.Equals(resp.StatusCode(), nethttp.StatusOK))
		qt.Check(t, qt.IsTrue(<-streamed))
		qt.Check(t, qt.Equals(<-bodies, payload))
		req.Reset()
		resp.Reset()
	}
}

func TestProxyUpstreamFailure(t *testing.T) {
	upstream := httptest.NewServer(nethttp.NotFoundHandler())
	target, err := url.Parse(upstream.URL)
	qt.Assert(t, qt.IsNil(err))
	upstream.Close()

	a := NewTestApp()
	a.Start(t)
	defer a.Stop()
	a.Proxy("/proxy", ProxyUpstream(target))

	var raw fasthttp.RequestCtx
	raw.Request.Header.SetMethod("POST")
	raw.Request.SetRequestURI("/proxy")
	raw.Request.SetBodyStream(strings.NewReader("body"), -1)
	defer raw.Request.Reset()
	a.Handler(&raw)
	qt.Check(t, qt.Equals(raw.Response.StatusCode(), nethttp.StatusBadGateway))
}
