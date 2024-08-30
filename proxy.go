package azugo

import (
	"net"
	"strings"
)

type ProxyOptions struct {
	// ForwardLimit limits the number of entries in the headers that will be processed.
	// The default value is 1. Set to 0 to disable the limit.
	// Trusting all entries in the headers is a security risk.
	ForwardLimit int
	// TrustAll option sets to trust all proxies.
	TrustAll bool
	// TrustedIPs represents addresses of trusted proxies.
	TrustedIPs []net.IP
	// TrustedNetworks represents addresses of trusted networks.
	TrustedNetworks []*net.IPNet
	// TrustedHeaders represents headers that contain the client's real IP address and their preference.
	TrustedHeaders []string
}

var defaultProxyOptions = ProxyOptions{
	ForwardLimit: 1,
	TrustedIPs: []net.IP{
		net.IPv4(127, 0, 0, 1),
	},
}

// Clear clears trusted proxy list.
func (opts *ProxyOptions) Clear() {
	opts.TrustAll = false
	opts.TrustedIPs = make([]net.IP, 0)
	opts.TrustedNetworks = make([]*net.IPNet, 0)
}

// Add proxy IP or network in CIDR format to trusted proxy list.
// Specify "*" to trust all proxies.
func (opts *ProxyOptions) Add(ipnet string) {
	// Special option to trust all proxies if IP address is set as wildcard
	if ipnet == "*" {
		opts.TrustAll = true

		return
	}
	// CIDR format
	if strings.ContainsRune(ipnet, '/') {
		_, netmask, err := net.ParseCIDR(ipnet)
		if err != nil || netmask == nil {
			return
		}

		opts.TrustedNetworks = append(opts.TrustedNetworks, netmask)

		return
	}

	// Single IP address
	ipaddr := net.ParseIP(ipnet)
	if ipaddr == nil {
		return
	}

	opts.TrustedIPs = append(opts.TrustedIPs, ipaddr)
}

// IsTrustedProxy checks whether the proxy that request is coming from can be trusted.
func (c *Context) IsTrustedProxy() bool {
	if c.RouterOptions().Proxy.TrustAll {
		return true
	}

	ip := c.IP()
	if ip == nil {
		return false
	}

	for _, tip := range c.RouterOptions().Proxy.TrustedIPs {
		if tip.Equal(ip) {
			return true
		}
	}

	for _, tnet := range c.RouterOptions().Proxy.TrustedNetworks {
		if tnet.Contains(ip) {
			return true
		}
	}

	return false
}
