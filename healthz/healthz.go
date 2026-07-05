// Package healthz provides a health check endpoint handler following the
// IETF Health Check Response Format for HTTP APIs draft specification.
//
// ref: https://datatracker.ietf.org/doc/html/draft-inadarei-api-health-check
package healthz

import (
	"azugo.io/azugo"

	"azugo.io/core/http"
)

// Status indicates the health status of the service.
//
// ref: https://datatracker.ietf.org/doc/html/draft-inadarei-api-health-check#section-3.1
type Status string

const (
	// Pass indicates the service is healthy. HTTP 2xx is returned.
	Pass Status = "pass"
	// Fail indicates the service is unhealthy. HTTP 5xx is returned.
	Fail Status = "fail"
	// Warn indicates the service is healthy with some concerns. HTTP 2xx is returned.
	Warn Status = "warn"
)

// statusRank returns the severity rank of a status (higher is worse).
func statusRank(s Status) int {
	switch s {
	case Fail:
		return 2
	case Warn:
		return 1
	case Pass:
		return 0
	}

	return 0
}

// Response is the data returned by the health check endpoint.
type Response struct {
	// Status indicates whether the service status is acceptable or not.
	Status Status `json:"status"`
	// Description is a human-friendly description of the service.
	Description string `json:"description,omitempty"`
}

// CheckFunc is a function that performs a health check and returns a Response.
// Returning nil is treated as pass.
type CheckFunc func(*azugo.Context) *Response

// Handler returns an azugo.RequestHandler for the health check endpoint.
// Allowed source IPs are controlled by app.HealthzOptions, configured via
// the healthz section in the application configuration.
// All provided checks are run; the worst status wins. When multiple checks
// share the worst status the description from the first one is used.
//
// Basic usage:
//
//	app.Get("/healthz", healthz.Handler())
//
// With checks:
//
//	app.Get("/healthz", healthz.Handler(
//	    func(ctx *azugo.Context) *healthz.Response {
//	        if !db.Ping() {
//	            return &healthz.Response{Status: healthz.Fail, Description: "database unreachable"}
//	        }
//	        return nil
//	    },
//	))
func Handler(checks ...CheckFunc) azugo.RequestHandler {
	return func(ctx *azugo.Context) {
		ctx.SkipRequestLog()
		ctx.SkipMetrics()

		if !ctx.App().HealthzOptions.IsTrusted(ctx.IP()) {
			ctx.NotFound()

			return
		}

		resp := &Response{Status: Pass}

		for _, fn := range checks {
			if fn == nil {
				continue
			}

			r := fn(ctx)
			if r == nil {
				continue
			}

			if statusRank(r.Status) > statusRank(resp.Status) {
				resp = r
			}
		}

		if resp.Status == Fail {
			ctx.StatusCode(http.StatusServiceUnavailable)
		}

		ctx.JSON(resp)
	}
}
