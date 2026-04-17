package azugo

import "net"

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
