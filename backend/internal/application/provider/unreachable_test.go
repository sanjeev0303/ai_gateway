package provider

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"
)

func TestIsUnreachable(t *testing.T) {
	dns := &net.DNSError{Err: "server misbehaving", Name: "api.mistral.ai", IsTemporary: true}
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"dns failure", dns, true},
		{"wrapped in url.Error", fmt.Errorf("openai: %w", &url.Error{Op: "Get", URL: "https://api.mistral.ai/v1/models", Err: dns}), true},
		{"timeout", fmt.Errorf("validate: %w", context.DeadlineExceeded), true},
		{"provider answered", errors.New("unexpected status 500"), false},
		{"bad key", ErrInvalidProviderCredential, false},
	}
	for _, tc := range cases {
		if got := isUnreachable(tc.err); got != tc.want {
			t.Errorf("%s: isUnreachable = %v, want %v", tc.name, got, tc.want)
		}
	}
}
