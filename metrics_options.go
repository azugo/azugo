package azugo

import "net"

// DefaultMetricPath is the default path for the Prometheus metrics endpoint.
const DefaultMetricPath string = "/metrics"

// MetricsOptions configures the metrics endpoint.
type MetricsOptions struct {
	TrustedSource

	// SkipPaths represents paths to bypass metrics handler.
	SkipPaths []string
}

var defaultMetricsOptions = MetricsOptions{
	TrustedSource: TrustedSource{
		TrustedIPs: []net.IP{
			net.IPv4(127, 0, 0, 1),
		},
	},
}
