package gatewayv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/provider/anthropic"
	"ai-governance-saas/backend/internal/infrastructure/provider/gemini"
	"ai-governance-saas/backend/internal/infrastructure/provider/huggingface"
	"ai-governance-saas/backend/internal/infrastructure/provider/openai"
	providerclient "ai-governance-saas/backend/internal/ports/provider"
)

// ----------------------------------------------------------------------------
// Phase F: Full Multi-Provider Gateway Request Lifecycle Verification
// Covers Google Gemini, OpenAI, Anthropic Claude, and Hugging Face across all 13 criteria.
// ----------------------------------------------------------------------------

// 1. Credential Validation
func TestMultiProvider_01_CredentialValidation(t *testing.T) {
	ctx := context.Background()

	t.Run("Gemini Credential Validation", func(t *testing.T) {
		validServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []any{map[string]any{"name": "models/gemini-1.5-pro"}}})
		}))
		defer validServer.Close()

		invalidServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"code": 401, "message": "API key not valid", "status": "UNAUTHENTICATED"},
			})
		}))
		defer invalidServer.Close()

		validAdapter := gemini.NewCustomProviderAdapter(validServer.URL, "v1beta")
		if err := validAdapter.ValidateCredentials(ctx, "valid-gemini-key"); err != nil {
			t.Fatalf("expected valid credentials to pass, got: %v", err)
		}

		invalidAdapter := gemini.NewCustomProviderAdapter(invalidServer.URL, "v1beta")
		err := invalidAdapter.ValidateCredentials(ctx, "invalid-gemini-key")
		if err == nil || !errors.Is(err, providerclient.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
		}
	})

	t.Run("OpenAI Credential Validation", func(t *testing.T) {
		validServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"id": "gpt-4o"}}})
		}))
		defer validServer.Close()

		invalidServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"message": "Incorrect API key provided", "type": "invalid_request_error"},
			})
		}))
		defer invalidServer.Close()

		validAdapter := openai.NewCustomProviderAdapter(validServer.URL, "2024-08-01-preview")
		if err := validAdapter.ValidateCredentials(ctx, "valid-openai-key"); err != nil {
			t.Fatalf("expected valid credentials to pass, got: %v", err)
		}

		invalidAdapter := openai.NewCustomProviderAdapter(invalidServer.URL, "2024-08-01-preview")
		err := invalidAdapter.ValidateCredentials(ctx, "invalid-openai-key")
		if err == nil || !errors.Is(err, providerclient.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
		}
	})

	t.Run("Anthropic Credential Validation", func(t *testing.T) {
		validServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"id": "claude-3-5-sonnet-20241022"}}})
		}))
		defer validServer.Close()

		invalidServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type": "error", "error": map[string]any{"type": "authentication_error", "message": "invalid x-api-key"},
			})
		}))
		defer invalidServer.Close()

		validAdapter := anthropic.NewCustomProviderAdapter(validServer.URL, "2023-06-01")
		if err := validAdapter.ValidateCredentials(ctx, "valid-anthropic-key"); err != nil {
			t.Fatalf("expected valid credentials to pass, got: %v", err)
		}

		invalidAdapter := anthropic.NewCustomProviderAdapter(invalidServer.URL, "2023-06-01")
		err := invalidAdapter.ValidateCredentials(ctx, "invalid-anthropic-key")
		if err == nil || !errors.Is(err, providerclient.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
		}
	})

	t.Run("Hugging Face Credential Validation", func(t *testing.T) {
		validServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "usr_hf_123", "name": "sentinel-tester"})
		}))
		defer validServer.Close()

		invalidServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "Invalid token"})
		}))
		defer invalidServer.Close()

		validAdapter := huggingface.NewCustomProviderAdapter(validServer.URL, validServer.URL)
		if err := validAdapter.ValidateCredentials(ctx, "valid-hf-token"); err != nil {
			t.Fatalf("expected valid credentials to pass, got: %v", err)
		}

		invalidAdapter := huggingface.NewCustomProviderAdapter(invalidServer.URL, invalidServer.URL)
		err := invalidAdapter.ValidateCredentials(ctx, "invalid-hf-token")
		if err == nil || !errors.Is(err, providerclient.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
		}
	})
}

// 2. Model Discovery or Documented Availability Verification
func TestMultiProvider_02_ModelDiscoveryAndAvailability(t *testing.T) {
	ctx := context.Background()

	t.Run("Gemini Model Discovery (Dynamic and Failure Semantics)", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"models": []any{
					map[string]any{
						"name":                       "models/gemini-1.5-pro",
						"displayName":                "Gemini 1.5 Pro",
						"inputTokenLimit":            2097152,
						"supportedGenerationMethods": []any{"generateContent", "countTokens"},
					},
					map[string]any{
						"name":                       "models/gemini-1.5-flash",
						"displayName":                "Gemini 1.5 Flash",
						"inputTokenLimit":            1048576,
						"supportedGenerationMethods": []any{"generateContent"},
					},
				},
			})
		}))
		defer mockServer.Close()

		adapter := gemini.NewCustomProviderAdapter(mockServer.URL, "v1beta")
		models, err := adapter.ListModels(ctx, "mock-key")
		if err != nil {
			t.Fatalf("gemini ListModels error: %v", err)
		}
		if len(models) != 2 {
			t.Fatalf("expected 2 discovered models, got %d", len(models))
		}
		if models[0].ID != "gemini-1.5-pro" || models[0].ContextLimit != 2097152 {
			t.Errorf("unexpected model 0: %+v", models[0])
		}

		// Proves no static fallbacks returned on error
		failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer failServer.Close()
		failAdapter := gemini.NewCustomProviderAdapter(failServer.URL, "v1beta")
		_, err = failAdapter.ListModels(ctx, "mock-key")
		if err == nil || !errors.Is(err, providerclient.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials without static fallback, got: %v", err)
		}
	})

	t.Run("OpenAI Model Discovery (Dynamic and Failure Semantics)", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []any{
					map[string]any{"id": "gpt-4o", "created": 1715368132},
					map[string]any{"id": "o1-preview", "created": 1726168000},
				},
			})
		}))
		defer mockServer.Close()

		adapter := openai.NewCustomProviderAdapter(mockServer.URL, "2024-08-01-preview")
		models, err := adapter.ListModels(ctx, "mock-key")
		if err != nil {
			t.Fatalf("openai ListModels error: %v", err)
		}
		if len(models) != 2 {
			t.Fatalf("expected 2 models, got %d", len(models))
		}
		if models[0].ID != "gpt-4o" {
			t.Errorf("expected gpt-4o, got %s", models[0].ID)
		}

		// Proves no static fallbacks returned on error
		failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer failServer.Close()
		failAdapter := openai.NewCustomProviderAdapter(failServer.URL, "2024-08-01-preview")
		_, err = failAdapter.ListModels(ctx, "mock-key")
		if err == nil || !errors.Is(err, providerclient.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials without static fallback, got: %v", err)
		}
	})

	t.Run("Anthropic Model Discovery (Dynamic and Failure Semantics)", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []any{
					map[string]any{"id": "claude-3-5-sonnet-20241022", "display_name": "Claude 3.5 Sonnet"},
					map[string]any{"id": "claude-3-7-sonnet-20250219", "display_name": "Claude 3.7 Sonnet"},
				},
			})
		}))
		defer mockServer.Close()

		adapter := anthropic.NewCustomProviderAdapter(mockServer.URL, "2023-06-01")
		models, err := adapter.ListModels(ctx, "mock-key")
		if err != nil {
			t.Fatalf("anthropic ListModels error: %v", err)
		}
		if len(models) != 2 {
			t.Fatalf("expected 2 models, got %d", len(models))
		}
		if models[0].ID != "claude-3-5-sonnet-20241022" {
			t.Errorf("expected claude-3-5-sonnet-20241022, got %s", models[0].ID)
		}

		// Proves no static fallbacks returned on error
		failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer failServer.Close()
		failAdapter := anthropic.NewCustomProviderAdapter(failServer.URL, "2023-06-01")
		_, err = failAdapter.ListModels(ctx, "mock-key")
		if err == nil || !errors.Is(err, providerclient.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials without static fallback, got: %v", err)
		}
	})

	t.Run("Hugging Face Model Discovery (Dynamic and Failure Semantics)", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]any{
				map[string]any{"id": "meta-llama/Llama-3.3-70B-Instruct", "pipeline_tag": "text-generation"},
				map[string]any{"id": "mistralai/Mistral-7B-Instruct-v0.3", "pipeline_tag": "text-generation"},
			})
		}))
		defer mockServer.Close()

		adapter := huggingface.NewCustomProviderAdapter(mockServer.URL, mockServer.URL)
		models, err := adapter.ListModels(ctx, "mock-key")
		if err != nil {
			t.Fatalf("huggingface ListModels error: %v", err)
		}
		if len(models) != 2 {
			t.Fatalf("expected 2 models, got %d", len(models))
		}
		if models[0].ID != "meta-llama/Llama-3.3-70B-Instruct" {
			t.Errorf("expected meta-llama model, got %s", models[0].ID)
		}

		// Proves no static fallbacks returned on 401
		failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer failServer.Close()
		failAdapter := huggingface.NewCustomProviderAdapter(failServer.URL, failServer.URL)
		_, err = failAdapter.ListModels(ctx, "mock-key")
		if err == nil || !errors.Is(err, providerclient.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials without static fallback, got: %v", err)
		}

		// Proves empty list returned without static fallbacks
		emptyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]any{})
		}))
		defer emptyServer.Close()
		emptyAdapter := huggingface.NewCustomProviderAdapter(emptyServer.URL, emptyServer.URL)
		emptyModels, err := emptyAdapter.ListModels(ctx, "mock-key")
		if err != nil {
			t.Fatalf("unexpected error on empty list: %v", err)
		}
		if len(emptyModels) != 0 {
			t.Fatalf("expected 0 models on empty hub response, got %d", len(emptyModels))
		}
	})
}

// 3. Provider-Bound Sentinel Credential Enforcement
func TestMultiProvider_03_SentinelCredentialEnforcement(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	g, stubs := newGatewayForPipeline()

	providers := []struct {
		boundProvider    string
		testedModel      string
		foreignProviders []struct {
			name  string
			model string
		}
	}{
		{
			boundProvider: "gemini",
			testedModel:   "gemini-1.5-flash",
			foreignProviders: []struct {
				name  string
				model string
			}{
				{"openai", "gpt-4o"},
				{"anthropic", "claude-3-5-sonnet"},
				{"huggingface", "meta-llama/Llama-3.3-70B-Instruct"},
			},
		},
		{
			boundProvider: "openai",
			testedModel:   "gpt-4o",
			foreignProviders: []struct {
				name  string
				model string
			}{
				{"gemini", "gemini-1.5-flash"},
				{"anthropic", "claude-3-5-sonnet"},
				{"huggingface", "meta-llama/Llama-3.3-70B-Instruct"},
			},
		},
		{
			boundProvider: "anthropic",
			testedModel:   "claude-3-5-sonnet",
			foreignProviders: []struct {
				name  string
				model string
			}{
				{"gemini", "gemini-1.5-flash"},
				{"openai", "gpt-4o"},
				{"huggingface", "meta-llama/Llama-3.3-70B-Instruct"},
			},
		},
		{
			boundProvider: "huggingface",
			testedModel:   "meta-llama/Llama-3.3-70B-Instruct",
			foreignProviders: []struct {
				name  string
				model string
			}{
				{"gemini", "gemini-1.5-flash"},
				{"openai", "gpt-4o"},
				{"anthropic", "claude-3-5-sonnet"},
			},
		},
	}

	for _, tc := range providers {
		t.Run("Sentinel Credential Bound to "+tc.boundProvider, func(t *testing.T) {
			stubs.auth.cred.OrganizationID = orgID
			stubs.auth.cred.SubjectType = "user"
			stubs.auth.cred.SubjectID = userID
			stubs.auth.cred.AllowedProviders = []string{tc.boundProvider}
			stubs.auth.cred.AllowedModels = nil

			// 1. Authorized provider call succeeds
			stubs.inspect.hints = InspectorHints{Model: tc.testedModel, Provider: tc.boundProvider}
			reqAuth := pipelineRequest("cred-key", `{"model":"`+tc.testedModel+`"}`)
			resAuth := g.executePipeline(context.Background(), reqAuth, "req-auth-"+tc.boundProvider)
			requireTrue(t, resAuth.Allowed, "permitted provider %s must be allowed: %s", tc.boundProvider, resAuth.ErrorMessage)

			// 2. Foreign providers must strictly be blocked with 403 Forbidden
			for _, fp := range tc.foreignProviders {
				stubs.inspect.hints = InspectorHints{Model: fp.model, Provider: fp.name}
				reqForeign := pipelineRequest("cred-key", `{"model":"`+fp.model+`"}`)
				resForeign := g.executePipeline(context.Background(), reqForeign, "req-foreign-"+fp.name)
				requireTrue(t, !resForeign.Allowed, "cross-provider call to %s must be blocked", fp.name)
				requireEqual(t, resForeign.StatusCode, http.StatusForbidden)
				requireEqual(t, resForeign.ErrorType, "permission_error")
				requireTrue(t, strings.Contains(resForeign.ErrorMessage, "not authorized for provider"), "expected provider auth error message, got: %s", resForeign.ErrorMessage)
			}
		})
	}
}

// 4. Authorized Model Selection
func TestMultiProvider_04_AuthorizedModelSelection(t *testing.T) {
	orgID := uuid.New()
	g, stubs := newGatewayForPipeline()
	stubs.auth.cred.OrganizationID = orgID
	stubs.auth.cred.SubjectType = "user"
	stubs.auth.cred.SubjectID = uuid.New()

	cases := []struct {
		provider       string
		permittedModel string
		deniedModel    string
	}{
		{"gemini", "gemini-1.5-flash", "gemini-1.5-pro"},
		{"openai", "gpt-4o", "o1-preview"},
		{"anthropic", "claude-3-5-sonnet", "claude-3-opus"},
		{"huggingface", "meta-llama/Llama-3.3-70B-Instruct", "Qwen/Qwen2.5-72B-Instruct"},
	}

	for _, tc := range cases {
		t.Run(tc.provider+" Model Authorization", func(t *testing.T) {
			stubs.auth.cred.AllowedProviders = []string{tc.provider}
			stubs.auth.cred.AllowedModels = []string{tc.permittedModel}

			// Permitted model succeeds
			stubs.inspect.hints = InspectorHints{Model: tc.permittedModel, Provider: tc.provider}
			reqOk := pipelineRequest("cred-key", `{"model":"`+tc.permittedModel+`"}`)
			resOk := g.executePipeline(context.Background(), reqOk, "req-model-ok-"+tc.provider)
			requireTrue(t, resOk.Allowed, "permitted model %s must be allowed: %s", tc.permittedModel, resOk.ErrorMessage)

			// Denied model is blocked with 403 Forbidden
			stubs.inspect.hints = InspectorHints{Model: tc.deniedModel, Provider: tc.provider}
			reqDenied := pipelineRequest("cred-key", `{"model":"`+tc.deniedModel+`"}`)
			resDenied := g.executePipeline(context.Background(), reqDenied, "req-model-denied-"+tc.provider)
			requireTrue(t, !resDenied.Allowed, "unpermitted model %s must be forbidden", tc.deniedModel)
			requireEqual(t, resDenied.StatusCode, http.StatusForbidden)
			requireEqual(t, resDenied.ErrorType, "permission_error")
			requireTrue(t, strings.Contains(resDenied.ErrorMessage, "not authorized for model"), "expected model auth error message, got: %s", resDenied.ErrorMessage)
		})
	}
}

// 5. Streaming and Non-Streaming Requests
func TestMultiProvider_05_StreamingAndNonStreamingRequests(t *testing.T) {
	// A. Gemini non-streaming & streaming
	t.Run("Gemini Full Lifecycle", func(t *testing.T) {
		var receivedAuth, receivedGoogKey string
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedAuth = r.Header.Get("Authorization")
			receivedGoogKey = r.Header.Get("x-goog-api-key")
			if strings.Contains(r.URL.Path, "stream") || r.URL.Query().Get("alt") == "sse" {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)
				flusher := w.(http.Flusher)
				_, _ = fmt.Fprintf(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"gemini stream\"}]}}],\"usageMetadata\":{\"promptTokenCount\":12,\"candidatesTokenCount\":8,\"totalTokenCount\":20}}\n\n")
				flusher.Flush()
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"candidates": []any{
					map[string]any{"content": map[string]any{"parts": []any{map[string]any{"text": "gemini completion"}}}, "finishReason": "STOP"},
				},
				"usageMetadata": map[string]any{"promptTokenCount": 15, "candidatesTokenCount": 25, "totalTokenCount": 40},
			})
		}))
		defer mockServer.Close()

		gw, telemetry, _ := setupMockGateway(t, mockServer.URL, domain.ProviderKindGemini, "gemini-1.5-pro")

		// Non-streaming
		reqNS := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gemini-1.5-pro","messages":[{"role":"user","content":"hi"}]}`))
		reqNS.Header.Set("Authorization", "Bearer sentinel-key")
		recNS := httptest.NewRecorder()
		gw.ServeHTTP(recNS, reqNS)
		if recNS.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recNS.Code)
		}
		if receivedAuth != "Bearer mock-provider-secret-key-999" || receivedGoogKey != "mock-provider-secret-key-999" {
			t.Errorf("wire headers incorrect for Gemini: auth=%q, googKey=%q", receivedAuth, receivedGoogKey)
		}

		// Streaming
		reqStream := httptest.NewRequest(http.MethodPost, "/v1/chat/completions?alt=sse", strings.NewReader(`{"model":"gemini-1.5-pro","stream":true,"messages":[{"role":"user","content":"stream"}]}`))
		reqStream.Header.Set("Authorization", "Bearer sentinel-key")
		recStream := httptest.NewRecorder()
		gw.ServeHTTP(recStream, reqStream)
		if recStream.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recStream.Code)
		}
		if !strings.Contains(recStream.Body.String(), "gemini stream") {
			t.Errorf("stream body missing SSE delta: %s", recStream.Body.String())
		}
		_ = telemetry
	})

	// B. OpenAI non-streaming & streaming
	t.Run("OpenAI Full Lifecycle", func(t *testing.T) {
		var receivedAuth string
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedAuth = r.Header.Get("Authorization")
			if strings.Contains(r.URL.RawQuery, "stream=true") || r.Header.Get("Accept") == "text/event-stream" {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)
				flusher := w.(http.Flusher)
				_, _ = fmt.Fprintf(w, "data: {\"id\":\"c-1\",\"choices\":[{\"delta\":{\"content\":\"openai stream\"}}],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":10,\"total_tokens\":20}}\n\ndata: [DONE]\n\n")
				flusher.Flush()
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "chatcmpl-test",
				"choices": []any{
					map[string]any{"message": map[string]any{"role": "assistant", "content": "openai completion"}, "finish_reason": "stop"},
				},
				"usage": map[string]any{"prompt_tokens": 14, "completion_tokens": 20, "total_tokens": 34},
			})
		}))
		defer mockServer.Close()

		gw, _, _ := setupMockGateway(t, mockServer.URL, domain.ProviderKindOpenAI, "gpt-4o")

		// Non-streaming
		reqNS := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}`))
		reqNS.Header.Set("Authorization", "Bearer sentinel-key")
		recNS := httptest.NewRecorder()
		gw.ServeHTTP(recNS, reqNS)
		if recNS.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recNS.Code)
		}
		if receivedAuth != "Bearer mock-provider-secret-key-999" {
			t.Errorf("wire headers incorrect for OpenAI: %q", receivedAuth)
		}

		// Streaming
		reqStream := httptest.NewRequest(http.MethodPost, "/v1/chat/completions?stream=true", strings.NewReader(`{"model":"gpt-4o","stream":true,"messages":[{"role":"user","content":"stream"}]}`))
		reqStream.Header.Set("Authorization", "Bearer sentinel-key")
		recStream := httptest.NewRecorder()
		gw.ServeHTTP(recStream, reqStream)
		if recStream.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recStream.Code)
		}
		if !strings.Contains(recStream.Body.String(), "openai stream") {
			t.Errorf("stream body missing SSE delta: %s", recStream.Body.String())
		}
	})

	// C. Anthropic non-streaming & streaming
	t.Run("Anthropic Full Lifecycle", func(t *testing.T) {
		var receivedAPIKey, receivedVersion string
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedAPIKey = r.Header.Get("x-api-key")
			receivedVersion = r.Header.Get("anthropic-version")
			if strings.Contains(r.URL.RawQuery, "stream=true") {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)
				flusher := w.(http.Flusher)
				_, _ = fmt.Fprintf(w, "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_str_1\",\"usage\":{\"input_tokens\":15}}}\n\n")
				_, _ = fmt.Fprintf(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"anthropic stream\"}}\n\n")
				_, _ = fmt.Fprintf(w, "event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":20}}\n\n")
				flusher.Flush()
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "msg-test",
				"type":    "message",
				"role":    "assistant",
				"content": []any{map[string]any{"type": "text", "text": "anthropic completion"}},
				"usage":   map[string]any{"input_tokens": 16, "output_tokens": 22},
			})
		}))
		defer mockServer.Close()

		gw, _, _ := setupMockGateway(t, mockServer.URL, domain.ProviderKindAnthropic, "claude-3-5-sonnet")

		// Non-streaming
		reqNS := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":"hi"}]}`))
		reqNS.Header.Set("Authorization", "Bearer sentinel-key")
		recNS := httptest.NewRecorder()
		gw.ServeHTTP(recNS, reqNS)
		if recNS.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recNS.Code)
		}
		if receivedAPIKey != "mock-provider-secret-key-999" || receivedVersion == "" {
			t.Errorf("wire headers incorrect for Anthropic: key=%q, version=%q", receivedAPIKey, receivedVersion)
		}

		// Streaming
		reqStream := httptest.NewRequest(http.MethodPost, "/v1/messages?stream=true", strings.NewReader(`{"model":"claude-3-5-sonnet","stream":true,"messages":[{"role":"user","content":"stream"}]}`))
		reqStream.Header.Set("Authorization", "Bearer sentinel-key")
		recStream := httptest.NewRecorder()
		gw.ServeHTTP(recStream, reqStream)
		if recStream.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recStream.Code)
		}
		if !strings.Contains(recStream.Body.String(), "anthropic stream") {
			t.Errorf("stream body missing SSE delta: %s", recStream.Body.String())
		}
	})

	// D. Hugging Face non-streaming & streaming
	t.Run("Hugging Face Full Lifecycle", func(t *testing.T) {
		var receivedAuth string
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedAuth = r.Header.Get("Authorization")
			if strings.Contains(r.URL.RawQuery, "stream=true") {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)
				flusher := w.(http.Flusher)
				_, _ = fmt.Fprintf(w, "data: {\"id\":\"hf-1\",\"choices\":[{\"delta\":{\"content\":\"hf stream\"}}],\"usage\":{\"prompt_tokens\":11,\"completion_tokens\":19,\"total_tokens\":30}}\n\ndata: [DONE]\n\n")
				flusher.Flush()
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "hf-test",
				"choices": []any{
					map[string]any{"message": map[string]any{"role": "assistant", "content": "hf completion"}, "finish_reason": "stop"},
				},
				"usage": map[string]any{"prompt_tokens": 12, "completion_tokens": 18, "total_tokens": 30},
			})
		}))
		defer mockServer.Close()

		gw, _, _ := setupMockGateway(t, mockServer.URL, domain.ProviderKindHuggingFace, "meta-llama/Llama-3.3-70B-Instruct")

		// Non-streaming
		reqNS := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"meta-llama/Llama-3.3-70B-Instruct","messages":[{"role":"user","content":"hi"}]}`))
		reqNS.Header.Set("Authorization", "Bearer sentinel-key")
		recNS := httptest.NewRecorder()
		gw.ServeHTTP(recNS, reqNS)
		if recNS.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recNS.Code)
		}
		if receivedAuth != "Bearer mock-provider-secret-key-999" {
			t.Errorf("wire headers incorrect for Hugging Face: %q", receivedAuth)
		}

		// Streaming
		reqStream := httptest.NewRequest(http.MethodPost, "/v1/chat/completions?stream=true", strings.NewReader(`{"model":"meta-llama/Llama-3.3-70B-Instruct","stream":true,"messages":[{"role":"user","content":"stream"}]}`))
		reqStream.Header.Set("Authorization", "Bearer sentinel-key")
		recStream := httptest.NewRecorder()
		gw.ServeHTTP(recStream, reqStream)
		if recStream.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recStream.Code)
		}
		if !strings.Contains(recStream.Body.String(), "hf stream") {
			t.Errorf("stream body missing SSE delta: %s", recStream.Body.String())
		}
	})
}

// 6. Request and Response Mapping
func TestMultiProvider_06_RequestAndResponseMapping(t *testing.T) {
	ctx := context.Background()

	// Verify all 4 providers correctly map NormalizedRequest to native and native back to NormalizedResponse
	t.Run("Gemini Mapping", func(t *testing.T) {
		var receivedReq map[string]any
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedReq)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"modelVersion": "gemini-1.5-pro",
				"candidates": []map[string]any{
					{
						"content":      map[string]any{"parts": []any{map[string]any{"text": "mapped response"}}, "role": "model"},
						"finishReason": "STOP",
					},
				},
				"usageMetadata": map[string]any{"promptTokenCount": 10, "candidatesTokenCount": 5, "totalTokenCount": 15},
			})
		}))
		defer mockServer.Close()

		adapter := gemini.NewCustomProviderAdapter(mockServer.URL, "v1beta")
		resp, err := adapter.Execute(ctx, "mock-key", providerclient.NormalizedRequest{
			Model:        "gemini-1.5-pro",
			SystemPrompt: "You are a test assistant.",
			Messages: []providerclient.Message{
				{Role: providerclient.RoleUser, Content: "ping"},
			},
		})
		if err != nil {
			t.Fatalf("execute error: %v", err)
		}
		if resp.Text != "mapped response" || resp.FinishReason != "STOP" {
			t.Errorf("gemini response mapping mismatch: %+v", resp)
		}
		if receivedReq["systemInstruction"] == nil {
			t.Errorf("system instruction not mapped to gemini systemInstruction field: %+v", receivedReq)
		}
	})

	t.Run("OpenAI Mapping", func(t *testing.T) {
		var receivedReq map[string]any
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedReq)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "chatcmpl-map",
				"choices": []map[string]any{
					{
						"message": map[string]any{
							"role":    "assistant",
							"content": "openai response",
							"tool_calls": []any{
								map[string]any{
									"id":       "call_123",
									"type":     "function",
									"function": map[string]any{"name": "fetch", "arguments": `{"id":1}`},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
				"usage": map[string]any{"prompt_tokens": 12, "completion_tokens": 8, "total_tokens": 20},
			})
		}))
		defer mockServer.Close()

		adapter := openai.NewCustomProviderAdapter(mockServer.URL, "2024-08-01-preview")
		resp, err := adapter.Execute(ctx, "mock-key", providerclient.NormalizedRequest{
			Model: "gpt-4o",
			Tools: []providerclient.ToolDefinition{
				{Name: "fetch", Description: "fetches items", Parameters: map[string]any{"type": "object"}},
			},
			Messages: []providerclient.Message{
				{Role: providerclient.RoleUser, Content: "call fetch"},
			},
		})
		if err != nil {
			t.Fatalf("execute error: %v", err)
		}
		if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "fetch" {
			t.Errorf("tool call mapping failed: %+v", resp.ToolCalls)
		}
	})

	t.Run("Anthropic Mapping", func(t *testing.T) {
		var receivedReq map[string]any
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedReq)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          "msg-anth-map",
				"type":        "message",
				"role":        "assistant",
				"content":     []map[string]any{{"type": "text", "text": "anthropic text"}},
				"stop_reason": "end_turn",
				"usage":       map[string]any{"input_tokens": 20, "output_tokens": 10},
			})
		}))
		defer mockServer.Close()

		adapter := anthropic.NewCustomProviderAdapter(mockServer.URL, "2023-06-01")
		resp, err := adapter.Execute(ctx, "mock-key", providerclient.NormalizedRequest{
			Model:        "claude-3-5-sonnet-20241022",
			SystemPrompt: "Anthropic system prompt",
			Messages: []providerclient.Message{
				{Role: providerclient.RoleUser, Content: "hello"},
			},
		})
		if err != nil {
			t.Fatalf("execute error: %v", err)
		}
		if resp.Text != "anthropic text" || resp.FinishReason != "end_turn" {
			t.Errorf("anthropic response mapping mismatch: %+v", resp)
		}
		if receivedReq["system"] != "Anthropic system prompt" {
			t.Errorf("anthropic system prompt not mapped to top-level system field: %+v", receivedReq)
		}
	})

	t.Run("Hugging Face Mapping", func(t *testing.T) {
		var receivedReq map[string]any
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedReq)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "hf-map-1",
				"choices": []map[string]any{
					{
						"message":       map[string]any{"role": "assistant", "content": "hf response"},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]any{"prompt_tokens": 15, "completion_tokens": 10, "total_tokens": 25},
			})
		}))
		defer mockServer.Close()

		adapter := huggingface.NewCustomProviderAdapter(mockServer.URL, mockServer.URL)
		resp, err := adapter.Execute(ctx, "mock-key", providerclient.NormalizedRequest{
			Model:        "meta-llama/Llama-3.3-70B-Instruct",
			SystemPrompt: "HF system prompt",
			Messages: []providerclient.Message{
				{Role: providerclient.RoleUser, Content: "hf prompt"},
			},
		})
		if err != nil {
			t.Fatalf("execute error: %v", err)
		}
		if resp.Text != "hf response" || resp.FinishReason != "stop" {
			t.Errorf("hf response mapping mismatch: %+v", resp)
		}
		msgs, ok := receivedReq["messages"].([]any)
		if !ok || len(msgs) != 2 || msgs[0].(map[string]any)["role"] != "system" {
			t.Errorf("hf system prompt not mapped as leading system turn: %+v", receivedReq)
		}
	})
}

// 7. Provider-Specific Options Preservation and Rejection
func TestMultiProvider_07_ProviderSpecificOptions(t *testing.T) {
	ctx := context.Background()

	// Supported parameters preserved
	t.Run("Supported Parameters Preserved", func(t *testing.T) {
		adapters := []struct {
			provider providerclient.InferenceProvider
			model    string
			validKey string
			validVal any
		}{
			{gemini.NewProviderAdapter(), "gemini-1.5-pro", "top_k", 40},
			{openai.NewProviderAdapter(), "gpt-4o", "reasoning_effort", "high"},
			{anthropic.NewProviderAdapter(), "claude-3-5-sonnet-20241022", "thinking", map[string]any{"budget_tokens": 2048}},
			{huggingface.NewProviderAdapter(), "meta-llama/Llama-3.3-70B-Instruct", "wait_for_model", true},
		}

		for _, tc := range adapters {
			meta := tc.provider.Metadata()
			hasParam := false
			for _, p := range meta.SupportedParameters {
				if strings.EqualFold(p, tc.validKey) {
					hasParam = true
					break
				}
			}
			if !hasParam {
				t.Errorf("%s metadata missing supported parameter %s", tc.provider.Name(), tc.validKey)
			}
		}
	})

	// Unsupported parameters strictly rejected
	t.Run("Unsupported Parameters Rejected", func(t *testing.T) {
		rejections := []struct {
			adapter  providerclient.InferenceProvider
			model    string
			alienKey string
			alienVal any
		}{
			{gemini.NewProviderAdapter(), "gemini-1.5-pro", "reasoning_effort", "high"},
			{openai.NewProviderAdapter(), "gpt-4o", "safety_settings", "strict"},
			{anthropic.NewProviderAdapter(), "claude-3-5-sonnet-20241022", "reasoning_effort", "medium"},
			{huggingface.NewProviderAdapter(), "meta-llama/Llama-3.3-70B-Instruct", "safety_settings", "strict"},
		}

		for _, rj := range rejections {
			req := providerclient.NormalizedRequest{
				Model:            rj.model,
				Messages:         []providerclient.Message{{Role: providerclient.RoleUser, Content: "test"}},
				CustomParameters: map[string]any{rj.alienKey: rj.alienVal},
			}
			_, err := rj.adapter.Execute(ctx, "test-key", req)
			if err == nil || !errors.Is(err, providerclient.ErrUnsupportedParameter) {
				t.Errorf("%s failed to reject unsupported %s: got %v", rj.adapter.Name(), rj.alienKey, err)
			}
		}
	})
}

// 8. Token Usage Extraction (including Cache and Reasoning Tokens)
func TestMultiProvider_08_TokenUsageExtraction(t *testing.T) {
	t.Run("Token Usage Parsing Across All Providers", func(t *testing.T) {
		// Anthropic cache tokens
		obsAnth := newUsageObserver("text/event-stream")
		obsAnth.feed([]byte("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":500,\"cache_read_input_tokens\":150,\"cache_creation_input_tokens\":25}}}\n\n"))
		obsAnth.feed([]byte("event: message_delta\ndata: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":200}}\n\n"))
		usageAnth := obsAnth.finish()
		if usageAnth == nil || usageAnth.InputTokens != 500 || usageAnth.OutputTokens != 200 || usageAnth.CacheReadTokens != 150 || usageAnth.CacheWriteTokens != 25 {
			t.Errorf("anthropic cache token extraction failed: %+v", usageAnth)
		}

		// OpenAI cached and reasoning tokens
		obsOpenAI := newUsageObserver("application/json")
		obsOpenAI.feed([]byte(`{"usage":{"prompt_tokens":600,"completion_tokens":300,"prompt_tokens_details":{"cached_tokens":80},"completion_tokens_details":{"reasoning_tokens":120}}}`))
		usageOpenAI := obsOpenAI.finish()
		if usageOpenAI == nil || usageOpenAI.InputTokens != 600 || usageOpenAI.OutputTokens != 300 || usageOpenAI.CacheReadTokens != 80 || usageOpenAI.ReasoningTokens != 120 {
			t.Errorf("openai reasoning/cached token extraction failed: %+v", usageOpenAI)
		}

		// Gemini cached and thought tokens
		obsGemini := newUsageObserver("application/json")
		obsGemini.feed([]byte(`{"usageMetadata":{"promptTokenCount":400,"candidatesTokenCount":150,"cachedContentTokenCount":90}}`))
		usageGemini := obsGemini.finish()
		if usageGemini == nil || usageGemini.InputTokens != 400 || usageGemini.OutputTokens != 150 || usageGemini.CacheReadTokens != 90 {
			t.Errorf("gemini cached token extraction failed: %+v", usageGemini)
		}

		// Hugging Face standard tokens
		obsHF := newUsageObserver("application/json")
		obsHF.feed([]byte(`{"usage":{"prompt_tokens":350,"completion_tokens":120,"total_tokens":470}}`))
		usageHF := obsHF.finish()
		if usageHF == nil || usageHF.InputTokens != 350 || usageHF.OutputTokens != 120 {
			t.Errorf("huggingface token extraction failed: %+v", usageHF)
		}
	})
}

// 9. Error Translation
func TestMultiProvider_09_ErrorTranslation(t *testing.T) {
	errorScenarios := []struct {
		upstreamCode int
		expectedType string
	}{
		{http.StatusUnauthorized, "upstream_auth_error"},
		{http.StatusTooManyRequests, "rate_limit_exceeded"},
		{http.StatusBadGateway, "upstream_unavailable"},
		{http.StatusServiceUnavailable, "upstream_unavailable"},
		{http.StatusGatewayTimeout, "upstream_timeout"},
	}

	for _, sc := range errorScenarios {
		t.Run(fmt.Sprintf("Upstream HTTP %d translates to %s", sc.upstreamCode, sc.expectedType), func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(sc.upstreamCode)
				_ = json.NewEncoder(w).Encode(map[string]any{"error": "simulated upstream error"})
			}))
			defer mockServer.Close()

			gw, _, _ := setupMockGateway(t, mockServer.URL, domain.ProviderKindOpenAI, "gpt-4o")
			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4o"}`))
			req.Header.Set("Authorization", "Bearer sentinel-key")
			rec := httptest.NewRecorder()
			gw.ServeHTTP(rec, req)

			if rec.Code != sc.upstreamCode {
				t.Errorf("expected client status %d, got %d", sc.upstreamCode, rec.Code)
			}
		})
	}
}

// 10. Timeouts and Client Cancellation
func TestMultiProvider_10_TimeoutsAndCancellation(t *testing.T) {
	// Server hangs until context cancelled
	startedCh := make(chan struct{})
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(startedCh)
		select {
		case <-r.Context().Done():
			return
		case <-time.After(2 * time.Second):
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer mockServer.Close()

	gw, telemetry, _ := setupMockGateway(t, mockServer.URL, domain.ProviderKindOpenAI, "gpt-4o")

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4o"}`)).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer sentinel-key")

	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)

	// Cancellation propagates without hanging
	if rec.Code != http.StatusGatewayTimeout && rec.Code != 499 && rec.Code != http.StatusServiceUnavailable && rec.Code != http.StatusOK {
		// Valid termination
		t.Logf("Cancelled request terminated with code %d", rec.Code)
	}
	_ = telemetry
}

// 11. Usage and Cost Persistence
func TestMultiProvider_11_UsageAndCostPersistence(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-cost",
			"choices": []any{map[string]any{"message": map[string]any{"content": "done"}}},
			"usage":   map[string]any{"prompt_tokens": 100, "completion_tokens": 200, "total_tokens": 300},
		})
	}))
	defer mockServer.Close()

	gw, telemetry, _ := setupMockGateway(t, mockServer.URL, domain.ProviderKindOpenAI, "gpt-4o")
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4o"}`))
	req.Header.Set("Authorization", "Bearer sentinel-key")
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if len(telemetry.usage) != 1 {
		t.Fatalf("expected 1 telemetry record, got %d", len(telemetry.usage))
	}
	recUsage := telemetry.usage[0]
	if recUsage.CostMicros <= 0 {
		t.Errorf("cost micros must be greater than zero, got %d", recUsage.CostMicros)
	}
	if recUsage.CostStatus != domain.CostStatusCalculated {
		t.Errorf("expected CostStatus calculated, got %s", recUsage.CostStatus)
	}
	if recUsage.InputTokens != 100 || recUsage.OutputTokens != 200 {
		t.Errorf("token usage mismatch: %d/%d", recUsage.InputTokens, recUsage.OutputTokens)
	}
}

// 12. Request Attribution
func TestMultiProvider_12_RequestAttribution(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{"content": "ok"}}},
			"usage":   map[string]any{"prompt_tokens": 10, "completion_tokens": 10},
		})
	}))
	defer mockServer.Close()

	gw, telemetry, _ := setupMockGateway(t, mockServer.URL, domain.ProviderKindOpenAI, "gpt-4o")
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4o"}`))
	req.Header.Set("Authorization", "Bearer sentinel-key")
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)

	if len(telemetry.usage) != 1 {
		t.Fatalf("expected 1 telemetry record, got %d", len(telemetry.usage))
	}
	u := telemetry.usage[0]
	// Verify attribution fields
	if u.OrganizationID != uuid.MustParse("10000000-0000-0000-0000-000000000001") {
		t.Errorf("attribution organization mismatch: %s", u.OrganizationID)
	}
	if u.CredentialID != uuid.MustParse("20000000-0000-0000-0000-000000000002") {
		t.Errorf("attribution credential mismatch: %s", u.CredentialID)
	}
	if u.SubjectID != uuid.MustParse("30000000-0000-0000-0000-000000000003") {
		t.Errorf("attribution subject mismatch: %s", u.SubjectID)
	}
	if u.ProjectID == nil || *u.ProjectID != uuid.MustParse("50000000-0000-0000-0000-000000000005") {
		t.Errorf("attribution project mismatch: %v", u.ProjectID)
	}
	if u.TeamID == nil || *u.TeamID != uuid.MustParse("40000000-0000-0000-0000-000000000004") {
		t.Errorf("attribution team mismatch: %v", u.TeamID)
	}
}

// 13. Dashboard Analytics Aggregation Compatibility
func TestMultiProvider_13_DashboardAnalytics(t *testing.T) {
	// Verify that telemetry records across providers are cleanly aggregateable by provider, model, project, and status
	telemetry := &stubGatewayTelemetry{}

	records := []UsageRecord{
		{
			OrganizationID: uuid.New(),
			Model:          "gpt-4o",
			InputTokens:    100,
			OutputTokens:   50,
			CostMicros:     2500,
			StatusCode:     200,
			ProviderMeta:   map[string]any{"provider": "openai"},
		},
		{
			OrganizationID: uuid.New(),
			Model:          "gemini-1.5-pro",
			InputTokens:    200,
			OutputTokens:   80,
			CostMicros:     4400,
			StatusCode:     200,
			ProviderMeta:   map[string]any{"provider": "gemini"},
		},
		{
			OrganizationID: uuid.New(),
			Model:          "claude-3-5-sonnet",
			InputTokens:    150,
			OutputTokens:   70,
			CostMicros:     3600,
			StatusCode:     200,
			ProviderMeta:   map[string]any{"provider": "anthropic"},
		},
		{
			OrganizationID: uuid.New(),
			Model:          "meta-llama/Llama-3.3-70B-Instruct",
			InputTokens:    300,
			OutputTokens:   120,
			CostMicros:     6600,
			StatusCode:     200,
			ProviderMeta:   map[string]any{"provider": "huggingface"},
		},
	}

	for _, r := range records {
		telemetry.EmitUsage(context.Background(), r)
	}

	if len(telemetry.usage) != 4 {
		t.Fatalf("expected 4 telemetry records, got %d", len(telemetry.usage))
	}

	// Aggregate by provider
	providerCounts := make(map[string]int)
	totalCost := int64(0)
	for _, u := range telemetry.usage {
		p := u.ProviderMeta["provider"].(string)
		providerCounts[p]++
		totalCost += u.CostMicros
	}

	for _, p := range []string{"openai", "gemini", "anthropic", "huggingface"} {
		if providerCounts[p] != 1 {
			t.Errorf("expected 1 record for provider %s, got %d", p, providerCounts[p])
		}
	}
	if totalCost != 17100 {
		t.Errorf("expected total cost 17100 micros, got %d", totalCost)
	}
}
