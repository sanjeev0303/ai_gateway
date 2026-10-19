# Sentinel AI Gateway — Multi-Provider Verification Report

## 1. Scope & Provider Coverage
The Sentinel AI Gateway supports four primary LLM upstream providers with streaming and non-streaming inference, credential validation, dynamic model discovery, and security controls:
1. **Google Gemini** (`backend/internal/infrastructure/provider/gemini`)
2. **OpenAI** (`backend/internal/infrastructure/provider/openai`)
3. **Anthropic Claude** (`backend/internal/infrastructure/provider/anthropic`)
4. **Hugging Face** (`backend/internal/infrastructure/provider/huggingface`)

---

## 2. Capability & Feature Support Matrix

| Feature | Anthropic | OpenAI | Google Gemini | Hugging Face |
| :--- | :--- | :--- | :--- | :--- |
| **Non-Streaming Inference** | `/v1/messages` | `/v1/chat/completions` | `/v1beta/models/{model}:generateContent` | `/v1/chat/completions` (router & dedicated) |
| **Streaming Inference (SSE)** | `text/event-stream` (delta events) | `text/event-stream` (`choices[].delta`) | `alt=sse` (`streamGenerateContent`) | `text/event-stream` (`data: {"choices": ...}`) |
| **Dynamic Model Discovery** | Live GET `/v1/models` | Live GET `/v1/models` | Live GET `/v1beta/models` | Live GET `/api/models?pipeline_tag=text-generation` |
| **Credential Validation** | Live header check; maps 401/403 | Live auth check against `/v1/models` | Live query with API key | Live token validation against Hub API |
| **Zero-Fallback Guarantee** | Fails closed on upstream error | Fails closed on upstream error | Fails closed on upstream error | Fails closed on upstream error |
| **Provider-Specific Options** | Preserves `max_tokens` (mandatory) | Preserves `reasoning_effort` (o1/o3) | Preserves `safety_settings` bounds | Preserves custom dedicated HTTPS endpoints |
| **Token Extraction** | Exact `input_tokens`, `output_tokens` | Exact prompt, completion, reasoning | Exact `promptTokenCount`, `candidatesTokenCount` | Exact usage or marked estimate |
| **Error Translation** | Maps 401/403/429/500/503/504 | Maps 401/403/429/500/503/504 | Maps 400/401/403/429/500/503 | Maps 401/403/429/500/503 |
| **Live Smoke Test** | `TestAnthropic_LiveSmokeTest` | `TestOpenAI_LiveSmokeTest` | `TestGemini_LiveSmokeTest` | `TestHuggingFace_LiveSmokeTest` |

---

## 3. Server-Side SSRF Protection on Hugging Face Endpoints
Custom inference endpoints passed via `/endpoint` or request options are strictly validated server-side by `ValidateHFEndpoint` (`provider_router.go`):
- Scheme must be HTTPS (`https://`)
- IP addresses (loopback, private, public, link-local) are strictly prohibited
- Cloud metadata hostnames (`metadata.google.internal`, `instance-data`) are rejected
- Hostnames must belong to verified domains: `*.endpoints.huggingface.cloud` or `*.huggingface.co`

---

## 4. Operational & Resilience Verification

1. **Timeouts and Cancellation**:
   - `AIGW_PROVIDER_REQUEST_TIMEOUT` bounds long-running queries; client context cancellation immediately terminates upstream connections and releases reserved budget allocations.
2. **Correlation & Telemetry**:
   - Every request is tagged with an invariant `X-Request-Id` and `organization_id` foreign key.
   - Usage events are emitted to the persistent event bus and ingested into `llm_requests` with exact token metrics.
3. **Safe Smoke Testing**:
   - Unit tests use deterministic mock HTTP servers (`httptest.Server`).
   - Live upstream smoke tests are gated by `os.Getenv("*_API_KEY")` and cleanly `t.Skip` without errors when credentials are absent.
   - Zero hardcoded secrets exist in source code, logs, test fixtures, or committed templates.

---

## 5. Test Verification Evidence
- `go test -v ./internal/infrastructure/provider/...`: All unit, contract, live-gated smoke tests, and adapter integration tests pass.
- `TestPhaseJ_GatewayInferenceRoutingAndGovernance`: E2E verification of all 4 providers in Gateway V2 data plane.
- `TestPhaseG_CLI_EndToEndScenario`: E2E verification of CLI interaction with multi-provider credentials.
