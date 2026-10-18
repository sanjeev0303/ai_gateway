package gatewayv2

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

// ----------------------------------------------------------------------------
// Phase J: Gateway V2 Data Plane & Inference Routing End-to-End Test Suite
// ----------------------------------------------------------------------------

func TestPhaseJ_GatewayInferenceRoutingAndGovernance(t *testing.T) {
	t.Log("Execution Mode: Deterministic Local Mock Gateway Pipeline (Hermetic Test Environment)")

	orgID := uuid.MustParse("aaaaaaaa-1111-4111-8111-111111111111")
	user1ID := uuid.MustParse("11111111-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	user2ID := uuid.MustParse("22222222-bbbb-4bbb-8bbb-bbbbbbbbbbbb")
	user3ID := uuid.MustParse("33333333-cccc-4ccc-8ccc-cccccccccccc")

	projCoreID := uuid.MustParse("cccccccc-1111-4111-8111-111111111111")
	projResearchID := uuid.MustParse("dddddddd-2222-4222-8222-222222222222")
	projInnovationID := uuid.MustParse("eeeeeeee-3333-4333-8333-333333333333")

	g, stubs := newGatewayForPipeline()

	// 1. User 1: Anthropic Bound Credential (claude-3-5-sonnet)
	t.Run("User 1: Anthropic Non-streaming and Streaming Routing", func(t *testing.T) {
		stubs.auth.cred.OrganizationID = orgID
		stubs.auth.cred.SubjectType = "user"
		stubs.auth.cred.SubjectID = user1ID
		stubs.auth.cred.ProjectID = &projCoreID
		stubs.auth.cred.AllowedProviders = []string{"anthropic"}
		stubs.auth.cred.AllowedModels = []string{"claude-3-5-sonnet"}

		// Non-streaming
		stubs.inspect.hints = InspectorHints{Model: "claude-3-5-sonnet", Provider: "anthropic", Stream: false}
		req := pipelineRequest("user1-key", `{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"hello"}]}`)
		res := g.executePipeline(context.Background(), req, "req-user1-anthropic-ns")
		requireTrue(t, res.Allowed, "user 1 anthropic non-streaming must be allowed: %s", res.ErrorMessage)
		requireTrue(t, res.Context != nil, "context must be populated")
		requireEqual(t, res.Context.InspectorHints.Model, "claude-3-5-sonnet")
		requireEqual(t, res.Context.InspectorHints.Provider, "anthropic")

		// Streaming
		stubs.inspect.hints = InspectorHints{Model: "claude-3-5-sonnet", Provider: "anthropic", Stream: true}
		reqStream := pipelineRequest("user1-key", `{"model":"claude-3-5-sonnet","stream":true,"messages":[{"role":"user","content":"stream hello"}]}`)
		resStream := g.executePipeline(context.Background(), reqStream, "req-user1-anthropic-stream")
		requireTrue(t, resStream.Allowed, "user 1 anthropic streaming must be allowed: %s", resStream.ErrorMessage)
		requireTrue(t, resStream.Context.InspectorHints.Stream, "streaming flag must be preserved")
	})

	// 2. User 2: Gemini Bound Credential (gemini-1.5-flash)
	t.Run("User 2: Gemini Routing and Model Scoping", func(t *testing.T) {
		stubs.auth.cred.OrganizationID = orgID
		stubs.auth.cred.SubjectType = "user"
		stubs.auth.cred.SubjectID = user2ID
		stubs.auth.cred.ProjectID = &projResearchID
		stubs.auth.cred.AllowedProviders = []string{"gemini"}
		stubs.auth.cred.AllowedModels = []string{"gemini-1.5-flash"}

		stubs.inspect.hints = InspectorHints{Model: "gemini-1.5-flash", Provider: "gemini"}
		req := pipelineRequest("user2-key", `{"contents":[{"parts":[{"text":"flash hello"}]}]}`)
		res := g.executePipeline(context.Background(), req, "req-user2-gemini-ok")
		requireTrue(t, res.Allowed, "user 2 gemini-1.5-flash must be allowed: %s", res.ErrorMessage)
		requireEqual(t, res.Context.InspectorHints.Model, "gemini-1.5-flash")
		requireEqual(t, res.Context.InspectorHints.Provider, "gemini")

		// Denied unpermitted model gemini-1.5-pro
		stubs.inspect.hints = InspectorHints{Model: "gemini-1.5-pro", Provider: "gemini"}
		reqPro := pipelineRequest("user2-key", `{"contents":[{"parts":[{"text":"pro hello"}]}]}`)
		resPro := g.executePipeline(context.Background(), reqPro, "req-user2-gemini-pro-forbidden")
		requireTrue(t, !resPro.Allowed, "gemini-1.5-pro must be forbidden for user 2")
		requireEqual(t, resPro.StatusCode, http.StatusForbidden)
	})

	// 3. User 3: OpenAI and Hugging Face Bound Credential
	t.Run("User 3: OpenAI and Hugging Face Multi-Provider Routing", func(t *testing.T) {
		stubs.auth.cred.OrganizationID = orgID
		stubs.auth.cred.SubjectType = "user"
		stubs.auth.cred.SubjectID = user3ID
		stubs.auth.cred.ProjectID = &projInnovationID
		stubs.auth.cred.AllowedProviders = []string{"openai", "huggingface"}
		stubs.auth.cred.AllowedModels = []string{"gpt-4o", "meta-llama/Llama-3.3-70B-Instruct"}

		// OpenAI gpt-4o
		stubs.inspect.hints = InspectorHints{Model: "gpt-4o", Provider: "openai"}
		reqOpenAI := pipelineRequest("user3-key", `{"model":"gpt-4o","messages":[{"role":"user","content":"openai hello"}]}`)
		resOpenAI := g.executePipeline(context.Background(), reqOpenAI, "req-user3-openai-ok")
		requireTrue(t, resOpenAI.Allowed, "user 3 gpt-4o must be allowed: %s", resOpenAI.ErrorMessage)
		requireEqual(t, resOpenAI.Context.InspectorHints.Provider, "openai")

		// Hugging Face Llama-3.3-70B
		stubs.inspect.hints = InspectorHints{Model: "meta-llama/Llama-3.3-70B-Instruct", Provider: "huggingface"}
		reqHF := pipelineRequest("user3-key", `{"model":"meta-llama/Llama-3.3-70B-Instruct"}`)
		resHF := g.executePipeline(context.Background(), reqHF, "req-user3-hf-ok")
		requireTrue(t, resHF.Allowed, "user 3 huggingface must be allowed: %s", resHF.ErrorMessage)
		requireEqual(t, resHF.Context.InspectorHints.Provider, "huggingface")

		// User 3 cannot access Anthropic (unallowed provider)
		stubs.inspect.hints = InspectorHints{Model: "claude-3-5-sonnet", Provider: "anthropic"}
		reqAnthropic := pipelineRequest("user3-key", `{"model":"claude-3-5-sonnet"}`)
		resAnthropic := g.executePipeline(context.Background(), reqAnthropic, "req-user3-anthropic-forbidden")
		requireTrue(t, !resAnthropic.Allowed, "anthropic must be forbidden for user 3")
		requireEqual(t, resAnthropic.StatusCode, http.StatusForbidden)
	})

	// 4. Parameter Tampering & Path/Query Escapes
	t.Run("API Parameter Tampering Detection", func(t *testing.T) {
		stubs.auth.cred.OrganizationID = orgID
		stubs.auth.cred.SubjectType = "user"
		stubs.auth.cred.SubjectID = user1ID
		stubs.auth.cred.AllowedProviders = []string{"anthropic"}
		stubs.auth.cred.AllowedModels = []string{"claude-3-5-sonnet"}

		stubs.inspect.hints = InspectorHints{Model: "claude-3-5-sonnet", Provider: "anthropic"}
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions?model=gpt-4o", strings.NewReader(`{"model":"claude-3-5-sonnet"}`))
		req.Header.Set("x-api-key", "user1-key")

		res := g.executePipeline(context.Background(), req, "req-tampered-query")
		requireTrue(t, !res.Allowed, "conflicting model query param must be rejected as parameter tampering")
		requireEqual(t, res.StatusCode, http.StatusBadRequest)
		requireEqual(t, res.ErrorType, "parameter_tampering")
	})

	// 5. Governance: Policy Denial, Budget Limits, Rate Limits
	t.Run("Policy, Budget and Rate Limit Enforcement", func(t *testing.T) {
		stubs.auth.cred.AllowedProviders = nil
		stubs.auth.cred.AllowedModels = nil
		stubs.inspect.hints = InspectorHints{Model: "claude-3-5-sonnet", Provider: "anthropic"}

		// Policy blocklist rule denial
		stubs.policy.snapshot.Rules = []PolicyRule{modelRule(domain.RuleTypeBlocklist, "claude-3-5-sonnet")}
		reqPolicy := pipelineRequest("user1-key", `{"model":"claude-3-5-sonnet"}`)
		resPolicy := g.executePipeline(context.Background(), reqPolicy, "req-policy-denied")
		requireTrue(t, !resPolicy.Allowed, "policy denial must reject request")
		requireEqual(t, resPolicy.StatusCode, http.StatusForbidden)
		requireEqual(t, resPolicy.ErrorType, "policy_denied")
		stubs.policy.snapshot.Rules = nil

		// Rate limit denial
		stubs.rate.result = RateLimitResult{
			Allowed:    false,
			Limit:      100,
			Remaining:  0,
			RetryAfter: time.Minute,
			Scope:      "credential",
			Key:        "cred:user1-key",
		}
		reqRate := pipelineRequest("user1-key", `{"model":"claude-3-5-sonnet"}`)
		resRate := g.executePipeline(context.Background(), reqRate, "req-rate-limited")
		requireTrue(t, !resRate.Allowed, "rate limited request must be rejected")
		requireEqual(t, resRate.StatusCode, http.StatusTooManyRequests)
		requireEqual(t, resRate.ErrorType, "rate_limit_exceeded")
		stubs.rate.result = RateLimitResult{Allowed: true}

		// Budget limit denial
		stubs.budget.result = BudgetResult{
			Allowed: false,
			Limit:   100.0,
			Current: 100.0,
			Scope:   "organization",
			Key:     "org:budget",
		}
		reqBudget := pipelineRequest("user1-key", `{"model":"claude-3-5-sonnet"}`)
		resBudget := g.executePipeline(context.Background(), reqBudget, "req-budget-exceeded")
		requireTrue(t, !resBudget.Allowed, "budget exceeded request must be rejected")
		requireEqual(t, resBudget.StatusCode, http.StatusPaymentRequired)
		requireEqual(t, resBudget.ErrorType, "budget_exceeded")
		stubs.budget.result = BudgetResult{Allowed: true}
	})

	// 6. Provider Failure Simulation and Recovery
	t.Run("Provider Failure Handling and Recovery", func(t *testing.T) {
		stubs.inspect.hints = InspectorHints{Model: "claude-3-5-sonnet", Provider: "anthropic"}
		stubs.auth.cred.AllowedProviders = nil
		stubs.auth.cred.AllowedModels = nil

		// Simulate upstream provider outage (500)
		stubs.tenant.err = errors.New("upstream provider connection timed out after 30s")
		reqFail := pipelineRequest("user1-key", `{"model":"claude-3-5-sonnet"}`)
		resFail := g.executePipeline(context.Background(), reqFail, "req-provider-fail")
		requireTrue(t, !resFail.Allowed, "failed provider must not be allowed")
		requireEqual(t, resFail.StatusCode, http.StatusServiceUnavailable)
		requireEqual(t, resFail.ErrorType, "upstream_unavailable")

		// Recovery: provider becomes healthy again
		stubs.tenant.err = nil
		stubs.tenant.upstream = upstreamFixture()
		reqRecover := pipelineRequest("user1-key", `{"model":"claude-3-5-sonnet"}`)
		resRecover := g.executePipeline(context.Background(), reqRecover, "req-provider-recovered")
		requireTrue(t, resRecover.Allowed, "recovered provider must succeed: %s", resRecover.ErrorMessage)
	})

	// 7. Immediate Credential Revocation
	t.Run("Immediate Credential Revocation Enforcement", func(t *testing.T) {
		stubs.auth.err = ErrCredentialRevoked
		reqRevoked := pipelineRequest("revoked-key", `{"model":"claude-3-5-sonnet"}`)
		resRevoked := g.executePipeline(context.Background(), reqRevoked, "req-revoked-key")
		requireTrue(t, !resRevoked.Allowed, "revoked credential must be rejected")
		requireEqual(t, resRevoked.StatusCode, http.StatusUnauthorized)
		requireEqual(t, resRevoked.ErrorType, "authentication_error")
		stubs.auth.err = nil
	})

	_ = time.Second
}
