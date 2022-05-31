package middleware

import (
	"bytes"
	"strconv"
	"sync"
	"time"

	"azugo.io/azugo"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

var requestHandlerPool sync.Pool

type metricsHandler struct {
	reqCnt            *prometheus.CounterVec
	reqDur            *prometheus.HistogramVec
	reqSize, respSize prometheus.Summary
	// Metrics path
	MetricsPath string
	// Subsystem
	Subsystem string
}

// Interface for metrics handler options
type MetricsOption interface {
	apply(*metricsHandler)
}

// MetricsSubsystem represents subsystem name for Prometheus metric structuring
type MetricsSubsystem string

func (m MetricsSubsystem) apply(p *metricsHandler) {
	p.Subsystem = string(m)
}

// Metrics initializes and returns Prometheus metrics middleware
func Metrics(path string, options ...MetricsOption) azugo.RequestHandlerFunc {
	p := &metricsHandler{MetricsPath: path}
	for _, opt := range options {
		opt.apply(p)
	}
	p.registerMetrics()
	return p.Handler
}

// Idea is from https://github.com/DanielHeckrath/gin-prometheus/blob/master/gin_prometheus.go and https://github.com/zsais/go-gin-prometheus/blob/master/middleware.go
func computeApproximateRequestSize(req *fasthttp.Request, out chan int) {
	s := 0
	if req.URI() != nil {
		s += len(req.URI().Path())
		s += len(req.URI().Host())
	}
	s += len(req.Header.Method())
	s += len("HTTP/1.1")
	req.Header.VisitAll(func(key, value []byte) {
		if !bytes.Equal(key, []byte("Host")) {
			s += len(key) + len(value)
		}
	})
	if req.Header.ContentLength() != -1 {
		s += req.Header.ContentLength()
	}
	out <- s
}

// Handler returns metrics RequestHandler function which handles requests to gather metric data,
// skipping paths from SkipPaths list
//
// Handles MetricsPath requests from trusted IPs and trusted networks
// which returns application metrics results
func (p *metricsHandler) Handler(h azugo.RequestHandler) azugo.RequestHandler {
	metricsHandler := fasthttpadaptor.NewFastHTTPHandler(promhttp.Handler())
	return func(ctx *azugo.Context) {
		if bytes.EqualFold(ctx.Context().Path(), []byte(p.MetricsPath)) && p.isTrusted(ctx) {
			metricsHandler(ctx.Context())
			return
		}
		for _, path := range ctx.App().MetricsOptions.SkipPaths {
			if bytes.HasPrefix(bytes.ToLower(ctx.Context().Path()), []byte(path)) {
				h(ctx)
				return
			}
		}
		reqSize := make(chan int)
		frc := acquireRequestFromPool()
		ctx.Request().CopyTo(frc)
		go func() {
			defer releaseRequestToPool(frc)
			computeApproximateRequestSize(frc, reqSize)
		}()
		h(ctx)
		status := ctx.Response().StatusCode()
		if status == fasthttp.StatusNotFound {
			return
		}
		elapsed := float64(time.Since(ctx.Context().ConnTime())) / float64(time.Second)
		respSize := float64(len(ctx.Response().Body()))
		p.reqDur.WithLabelValues(strconv.Itoa(status), ctx.Method(), ctx.Path()).Observe(elapsed)
		p.reqCnt.WithLabelValues(strconv.Itoa(status), ctx.Method(), ctx.Path()).Inc()
		p.reqSize.Observe(float64(<-reqSize))
		p.respSize.Observe(respSize)
	}
}

func (p *metricsHandler) isTrusted(ctx *azugo.Context) bool {
	opts := ctx.App().MetricsOptions
	if opts.TrustAll {
		return true
	}
	ip := ctx.IP()
	for _, tip := range opts.TrustedIPs {
		if tip.Equal(ip) {
			return true
		}
	}
	for _, tnet := range opts.TrustedNetworks {
		if tnet.Contains(ip) {
			return true
		}
	}
	return false
}

func (p *metricsHandler) registerMetrics() {
	RequestDurationBucket := []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 15, 20, 30, 40, 50, 60}
	p.reqCnt = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: p.Subsystem,
			Name:      "requests_total",
			Help:      "The HTTP request counts processed.",
		},
		[]string{"code", "method", "path"},
	)
	p.reqDur = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: p.Subsystem,
			Name:      "request_duration_seconds",
			Help:      "The HTTP request duration in seconds.",
			Buckets:   RequestDurationBucket,
		},
		[]string{"code", "method", "path"},
	)
	p.reqSize = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Subsystem: p.Subsystem,
			Name:      "request_size_bytes",
			Help:      "The HTTP request sizes in bytes.",
		},
	)
	p.respSize = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Subsystem: p.Subsystem,
			Name:      "response_size_bytes",
			Help:      "The HTTP response sizes in bytes.",
		},
	)
	prometheus.MustRegister(p.reqCnt, p.reqDur, p.reqSize, p.respSize)
}

func acquireRequestFromPool() *fasthttp.Request {
	v := requestHandlerPool.Get()
	if v == nil {
		return &fasthttp.Request{}
	}
	return v.(*fasthttp.Request)
}

func releaseRequestToPool(req *fasthttp.Request) {
	req.Reset()
	requestHandlerPool.Put(req)
}
