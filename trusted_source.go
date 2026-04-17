package azugo

import (
	"net"
	"strings"
)

// defaultHealthzTrustedSource is pre-populated with localhost and private
// network ranges, suitable for Kubernetes and Docker environments.
var defaultHealthzTrustedSource TrustedSource

func init() {
	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"::1/128",        // IPv6 loopback
		"10.0.0.0/8",     // RFC 1918
		"172.16.0.0/12",  // RFC 1918 (includes Docker bridge 172.17.0.0/16)
		"192.168.0.0/16", // RFC 1918
		"fc00::/7",       // IPv6 unique local
	} {
		_, network, _ := net.ParseCIDR(cidr)
		defaultHealthzTrustedSource.TrustedNetworks = append(defaultHealthzTrustedSource.TrustedNetworks, network)
	}
}

// TrustedSource holds a list of trusted IP addresses and networks
// and reports whether a given IP is among them.
type TrustedSource struct {
	// TrustAll trusts all IP addresses.
	TrustAll bool
	// TrustedIPs is the list of trusted IP addresses.
	TrustedIPs []net.IP
	// TrustedNetworks is the list of trusted networks.
	TrustedNetworks []*net.IPNet
}

// Clear resets all trusted entries.
func (t *TrustedSource) Clear() *TrustedSource {
	t.TrustAll = false
	t.TrustedIPs = make([]net.IP, 0)
	t.TrustedNetworks = make([]*net.IPNet, 0)

	return t
}

// Add adds an IP address or CIDR network to the trusted list.
// Specify "*" to trust all sources.
func (t *TrustedSource) Add(ipnet string) *TrustedSource {
	if ipnet == "*" {
		t.TrustAll = true

		return t
	}

	if strings.ContainsRune(ipnet, '/') {
		_, network, err := net.ParseCIDR(ipnet)
		if err != nil || network == nil {
			return t
		}

		t.TrustedNetworks = append(t.TrustedNetworks, network)

		return t
	}

	ip := net.ParseIP(ipnet)
	if ip == nil {
		return t
	}

	t.TrustedIPs = append(t.TrustedIPs, ip)

	return t
}

// IsTrusted reports whether the given IP is in the trusted list.
func (t *TrustedSource) IsTrusted(ip net.IP) bool {
	if t.TrustAll {
		return true
	}

	for _, tip := range t.TrustedIPs {
		if tip.Equal(ip) {
			return true
		}
	}

	for _, tnet := range t.TrustedNetworks {
		if tnet.Contains(ip) {
			return true
		}
	}

	return false
}
