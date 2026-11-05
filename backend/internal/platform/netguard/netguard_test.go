package netguard

import (
	"net"
	"testing"
)

func TestBlocked(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "10.0.0.1", "192.168.1.1", "169.254.169.254", // loopback/private/link-local
		"100.64.0.1",   // CGNAT
		"0.0.0.0",      // unspecified / this-host
		"192.0.0.1",    // IETF protocol
		"198.18.0.1",   // benchmarking
		"224.0.0.1",    // multicast
		"::1",          // IPv6 loopback
	}
	for _, s := range blocked {
		if !Blocked(net.ParseIP(s)) {
			t.Errorf("expected %s to be blocked", s)
		}
	}
	public := []string{"8.8.8.8", "1.1.1.1", "140.82.112.3"}
	for _, s := range public {
		if Blocked(net.ParseIP(s)) {
			t.Errorf("expected %s to be allowed", s)
		}
	}
}

func TestValidateUpstreamURL(t *testing.T) {
	blocked := []string{
		"http://169.254.169.254/latest/meta-data/", // cloud metadata
		"http://localhost:9099/",
		"https://metadata.google.internal/",
		"http://127.0.0.1/",
		"http://10.0.0.5/",
		"http://192.168.1.1/",
		"ftp://example.com/", // non-http scheme
		"file:///etc/passwd",
	}
	for _, u := range blocked {
		if err := ValidateUpstreamURL(u, false); err == nil {
			t.Errorf("expected %q to be rejected", u)
		}
	}

	allowed := []string{"", "https://api.anthropic.com", "https://api.openai.com/v1"}
	for _, u := range allowed {
		if err := ValidateUpstreamURL(u, false); err != nil {
			t.Errorf("expected %q to be allowed, got %v", u, err)
		}
	}

	// The dev escape hatch permits localhost.
	if err := ValidateUpstreamURL("http://localhost:9099/", true); err != nil {
		t.Errorf("allowPrivate should permit localhost, got %v", err)
	}
}
