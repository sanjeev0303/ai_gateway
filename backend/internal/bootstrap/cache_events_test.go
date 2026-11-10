package bootstrap

import (
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/infrastructure/invalidation"
	"ai-governance-saas/backend/internal/interfaces/gatewayv2"
)

func TestDecodeCacheEvent(t *testing.T) {
	org := uuid.New()
	cred := uuid.New().String()

	cases := []struct {
		name    string
		channel string
		payload string
		want    gatewayv2.CacheEvent
		ok      bool
	}{
		{"policy", invalidation.PolicyInvalidationChannel, `{"org_id":"` + org.String() + `"}`, gatewayv2.CacheEvent{OrganizationID: org}, true},
		{"credential", invalidation.CredentialInvalidationChannel, `{"org_id":"` + org.String() + `","credential_id":"` + cred + `"}`, gatewayv2.CacheEvent{OrganizationID: org, CredentialID: cred}, true},
		{"policy without a valid org", invalidation.PolicyInvalidationChannel, `{"org_id":"nope"}`, gatewayv2.CacheEvent{}, false},
		{"credential without an id", invalidation.CredentialInvalidationChannel, `{"org_id":"` + org.String() + `"}`, gatewayv2.CacheEvent{}, false},
		{"garbage", invalidation.PolicyInvalidationChannel, `{`, gatewayv2.CacheEvent{}, false},
		{"unknown channel", "other", `{}`, gatewayv2.CacheEvent{}, false},
	}
	for _, tc := range cases {
		got, ok := decodeCacheEvent(tc.channel, tc.payload)
		if ok != tc.ok || got != tc.want {
			t.Errorf("%s: got (%+v, %v), want (%+v, %v)", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}
