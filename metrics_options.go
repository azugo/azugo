package azugo

import (
	"net"
	"strings"
)

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

// Clear trusted metrics client list.
func (opts *MetricsOptions) Clear() *MetricsOptions {
	opts.TrustAll = false
	opts.TrustedIPs = make([]net.IP, 0)
	opts.TrustedNetworks = make([]*net.IPNet, 0)

	return opts
}

// Add IP or network in CIDR format to trusted metrics client list.
// Specify "*" to trust all sources.
func (opts *MetricsOptions) Add(ipnet string) *MetricsOptions {
	// Special option to trust all sources if IP address is set as wildcard
	if ipnet == "*" {
		opts.TrustAll = true

		return opts
	}
	// CIDR format
	if strings.ContainsRune(ipnet, '/') {
		_, netmask, err := net.ParseCIDR(ipnet)
		if err != nil || netmask == nil {
			return opts
		}

		opts.TrustedNetworks = append(opts.TrustedNetworks, netmask)

		return opts
	}
	// Single IP address
	ipaddr := net.ParseIP(ipnet)
	if ipaddr == nil {
		return opts
	}

	opts.TrustedIPs = append(opts.TrustedIPs, ipaddr)

	return opts
}
