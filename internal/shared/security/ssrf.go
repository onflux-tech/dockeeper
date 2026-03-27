package security

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// blockedIPNets contains all private, loopback, link-local, and metadata IP ranges.
var blockedIPNets []*net.IPNet

func init() {
	cidrs := []string{
		"127.0.0.0/8",    // loopback
		"10.0.0.0/8",     // RFC 1918
		"172.16.0.0/12",  // RFC 1918
		"192.168.0.0/16", // RFC 1918
		"169.254.0.0/16", // link-local / cloud metadata
		"0.0.0.0/8",      // "this" network
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 unique local
		"fe80::/10",      // IPv6 link-local
	}
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil {
			blockedIPNets = append(blockedIPNets, network)
		}
	}
}

// ValidateWebhookURL validates a URL for SSRF safety.
// It resolves DNS and checks the resolved IP against blocked ranges.
func ValidateWebhookURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow http and https schemes.
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("SSRF: URL scheme %q not allowed, must be http or https", parsed.Scheme)
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return fmt.Errorf("SSRF: URL has no hostname")
	}

	// Resolve hostname to IPs.
	ips, err := net.LookupHost(hostname)
	if err != nil {
		return fmt.Errorf("SSRF: failed to resolve hostname %q: %w", hostname, err)
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		if isBlockedIP(ip) {
			return fmt.Errorf("SSRF: URL targets a restricted address (%s)", ipStr)
		}
	}

	return nil
}

// isBlockedIP checks if an IP falls within any blocked range.
func isBlockedIP(ip net.IP) bool {
	for _, network := range blockedIPNets {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
