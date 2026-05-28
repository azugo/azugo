package middleware

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"azugo.io/azugo"

	"github.com/VictoriaMetrics/metrics"
	"github.com/valyala/fasthttp"
)

var requestDurationBuckets = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 15, 20, 30, 40, 50, 60}

type metricsHandler struct {
	reqSize     *metrics.Summary
	respSize    *metrics.Summary
	metricsPath string
	subsystem   string
}

// MetricsOption is an interface for metrics handler options.
type MetricsOption interface {
	apply(h *metricsHandler)
}

// MetricsSubsystem represents subsystem name for metric structuring.
type MetricsSubsystem string

func (m MetricsSubsystem) apply(p *metricsHandler) {
	p.subsystem = string(m)
}

// Metrics initializes and returns metrics middleware.
func Metrics(path string, options ...MetricsOption) azugo.RequestHandlerFunc {
	p := &metricsHandler{metricsPath: path}
	for _, opt := range options {
		opt.apply(p)
	}

	metrics.ExposeMetadata(true)

	p.reqSize = metrics.GetOrCreateSummary(p.metricName("request_size_bytes"))
	p.respSize = metrics.GetOrCreateSummary(p.metricName("response_size_bytes"))

	return p.Handler
}

func (p *metricsHandler) metricName(base string) string {
	if p.subsystem != "" {
		return p.subsystem + "_" + base
	}

	return base
}

// Handler returns metrics RequestHandler function which handles requests to gather metric data,
// skipping paths from SkipPaths list.
//
// Handles MetricsPath requests from trusted IPs and trusted networks
// which returns application metrics results.
func (p *metricsHandler) Handler(h azugo.RequestHandler) azugo.RequestHandler {
	return func(ctx *azugo.Context) {
		if strings.EqualFold(ctx.Path(), p.metricsPath) {
			if p.isTrusted(ctx) {
				p.serveMetrics(ctx)
			} else {
				h(ctx)
			}

			return
		}

		for _, skip := range ctx.App().MetricsOptions.SkipPaths {
			if strings.HasPrefix(strings.ToLower(ctx.Path()), skip) {
				h(ctx)

				return
			}
		}

		h(ctx)

		if skip, ok := ctx.UserValue("__skip_metrics").(bool); ok && skip {
			return
		}

		status := ctx.Response().StatusCode()
		if status == fasthttp.StatusNotFound {
			return
		}

		elapsed := float64(time.Since(ctx.Context().ConnTime())) / float64(time.Second)

		var respSize float64

		if l := ctx.Response().Header.ContentLength(); l > 0 {
			respSize = float64(l)
		} else if !ctx.Response().IsBodyStream() {
			respSize = float64(len(ctx.Response().Body()))
		}

		path := ctx.RouterPath()
		if path == "" {
			path = ctx.Path()
		}

		labels := fmt.Sprintf(`{code=%q,method=%q,path=%q}`, strconv.Itoa(status), ctx.Method(), path)
		metrics.GetOrCreateCounter(p.metricName("requests_total") + labels).Inc()
		metrics.GetOrCreatePrometheusHistogramExt(p.metricName("request_duration_seconds")+labels, requestDurationBuckets).Update(elapsed)

		p.reqSize.Update(float64(computeApproximateRequestSize(ctx.Request())))
		p.respSize.Update(respSize)
	}
}

func (p *metricsHandler) serveMetrics(ctx *azugo.Context) {
	accept := string(ctx.Request().Header.Peek("Accept"))
	w := ctx.Response().BodyWriter()

	if negotiateOpenMetrics(accept) {
		ctx.Response().Header.Set("Content-Type", "application/openmetrics-text; version=1.0.0; charset=utf-8")
		metrics.WritePrometheus(w, true)
		fmt.Fprint(w, "# EOF\n")
	} else {
		ctx.Response().Header.Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		metrics.WritePrometheus(w, true)
	}
}

// negotiateOpenMetrics returns true if the Accept header prefers OpenMetrics format.
func negotiateOpenMetrics(accept string) bool {
	for part := range strings.SplitSeq(accept, ",") {
		part = strings.TrimSpace(part)
		mediaType, params, _ := strings.Cut(part, ";")

		if strings.TrimSpace(mediaType) != "application/openmetrics-text" {
			continue
		}

		for param := range strings.SplitSeq(params, ";") {
			if strings.TrimSpace(param) == "q=0" {
				return false
			}
		}

		return true
	}

	return false
}

func (p *metricsHandler) isTrusted(ctx *azugo.Context) bool {
	return ctx.App().MetricsOptions.IsTrusted(ctx.IP())
}

func computeApproximateRequestSize(req *fasthttp.Request) int {
	s := 0
	if req.URI() != nil {
		s += len(req.URI().Path())
		s += len(req.URI().Host())
	}

	s += len(req.Header.Method())
	s += len(req.Header.Protocol())

	for key, value := range req.Header.All() {
		if string(key) != "Host" {
			s += len(key) + len(value)
		}
	}

	if req.Header.ContentLength() != -1 {
		s += req.Header.ContentLength()
	}

	return s
}
