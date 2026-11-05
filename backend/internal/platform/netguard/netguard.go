// Package netguard blocks outbound connections to internal, loopback, and
// cloud-metadata addresses. It is the core defense against SSRF through the
// tenant-configurable upstream base URL: a tenant must not be able to aim the
// gateway (which forwards with the tenant's upstream key and streams the reply
// back) at 169.254.169.254, localhost, or an RFC1918 host.
package netguard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"syscall"
	"time"
)

// metadataHosts resolve to cloud-metadata or internal endpoints and are never a
// valid upstream, regardless of what they resolve to.
var metadataHosts = map[string]bool{
	"localhost":                true,
	"metadata.google.internal": true,
	"instance-data":            true,
	"metadata":                 true,
}

// extraBlocked are ranges net.IP's built-in predicates don't cover but which an
// upstream still must not reach: CGNAT, the 0.0.0.0/8 "this host" block,
// IETF-protocol space, and benchmarking space.
var extraBlocked = func() []*net.IPNet {
	cidrs := []string{"100.64.0.0/10", "0.0.0.0/8", "192.0.0.0/24", "198.18.0.0/15"}
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		if _, n, err := net.ParseCIDR(c); err == nil {
			out = append(out, n)
		}
	}
	return out
}()

// Blocked reports whether ip is an address an upstream must not point at.
func Blocked(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, n := range extraBlocked {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// ValidateUpstreamURL rejects an upstream base URL that would enable SSRF: a
// non-http(s) scheme, a metadata hostname, or a literal private/loopback
// address. An empty URL is allowed (the provider falls back to its own
// default). When allowPrivate is true (local development only) the address
// checks are skipped so dev harnesses can target localhost upstreams.
func ValidateUpstreamURL(rawURL string, allowPrivate bool) error {
	if rawURL == "" {
		return nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid upstream URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("upstream URL must use http or https scheme")
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("upstream URL must specify a hostname")
	}
	if allowPrivate {
		return nil
	}
	hostLower := strings.ToLower(host)
	if metadataHosts[hostLower] || strings.HasSuffix(hostLower, ".localhost") {
		return fmt.Errorf("prohibited upstream host %q", host)
	}
	if ip := net.ParseIP(hostLower); ip != nil && Blocked(ip) {
		return fmt.Errorf("prohibited upstream address %q", host)
	}
	return nil
}

// GuardedDialContext returns a DialContext that refuses connections to
// non-public addresses, for use on any http.Transport that may be pointed at a
// tenant-controlled upstream URL. When allowPrivate is true (local dev only) it
// is an ordinary dialer so harnesses can reach localhost upstreams.
func GuardedDialContext(allowPrivate bool) func(ctx context.Context, network, addr string) (net.Conn, error) {
	d := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	if !allowPrivate {
		d.Control = DialControl
	}
	return d.DialContext
}

// DialControl is a net.Dialer.Control hook that refuses connections to
// non-public addresses. It runs after DNS resolution on the concrete address,
// so it also closes DNS-rebinding that URL-string validation alone cannot.
func DialControl(network, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("netguard: bad dial address %q: %w", address, err)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("netguard: non-IP dial address %q", host)
	}
	if Blocked(ip) {
		return fmt.Errorf("netguard: blocked connection to non-public address %s", ip)
	}
	return nil
}
