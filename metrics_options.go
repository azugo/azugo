package azugo

import "net"

const DefaultMetricPath string = "/metrics"

type MetricsOptions struct {
	// TrustAll option sets to trust all IP addresses.
	TrustAll bool
	// TrustedIPs represents list of trusted IP addresses.
	TrustedIPs []net.IP
	// TrustedNetworks represents addresses of trusted networks.
	TrustedNetworks []*net.IPNet
	// SkipPaths represents paths to bypass metrics handler
	SkipPaths []string
}

var defaultMetricsOptions = MetricsOptions{
	TrustedIPs: []net.IP{
		net.IPv4(127, 0, 0, 1),
	},
}
