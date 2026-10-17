# Provider Request Lifecycle & Verification Matrix

## 1. Overview & Verification Scope

This document details the complete end-to-end request and response lifecycle for the three supported LLM providers: **OpenAI**, **Anthropic**, and **Gemini**.

No speculative or unsupported providers are added. Offline deterministic mock-server tests are strictly decoupled from live external provider calls. When live provider keys are not provisioned in the local environment, live tests explicitly report `BLOCKED`.

---

## 2. Supported Provider Protocol & Lifecycle Matrix

| Provider Lifecycle Stage | OpenAI | Anthropic | Gemini |
|---|---|---|---|
| **Upstream Base URL** | `https://api.openai.com/` | `https://api.anthropic.com/` | `https://generativelanguage.googleapis.com/v1beta/openai/` |
| **Authentication Header** | `Authorization: Bearer <key>` | `x-api-key: <key>`, `anthropic-version` | `Authorization: Bearer <key>`, `x-goog-api-key: <key>` |
| **Model Name Validation** | `gpt-*`, `o1*`, `o3*`, `text-embedding*`, `dall-e*` | `claude-*` | `gemini-*` |
| **Invalid Model Response** | HTTP 400 `unsupported_model` | HTTP 400 `unsupported_model` | HTTP 400 `unsupported_model` |
| **Non-Streaming Payload** | `POST /v1/chat/completions` (JSON `chat.completion`) | `POST /v1/messages` (JSON `message`) | `POST /v1/chat/completions` (JSON `GenerateContentResponse`) |
| **Streaming SSE Format** | `data: {"choices":[{"delta":...}]}` ... `data: [DONE]` | `event: message_start` ... `event: content_block_delta` ... `event: message_delta` | `data: {"candidates":[{"content":...}]}` |
| **Usage Metadata Location** | `usage.prompt_tokens`, `completion_tokens`, `prompt_tokens_details.cached_tokens`, `completion_tokens_details.reasoning_tokens` | `message.usage.input_tokens`, `output_tokens`, `cache_read_input_tokens`, `cache_creation_input_tokens`, `message_delta.usage.output_tokens` | `usageMetadata.promptTokenCount`, `candidatesTokenCount`, `cachedContentTokenCount` |
| **Missing Usage Fallback** | Falls back to request-inspected token estimates (`tokens_reported: false`, `estimated: true`, `CostStatus: estimated`) | Same | Same |
| **Provider Error (429/500/503)** | Upstream status code forwarded to client; telemetry records error status; token count set to 0; `CostMicros: 0`; `CostStatus: unavailable`; budget released | Same | Same |
| **Upstream Timeout** | Gateway returns HTTP 504 `upstream_timeout`; records status 504 in telemetry | Same | Same |
| **Client Disconnection** | Aborts upstream connection via context cancellation; records HTTP 499 in telemetry; unspent budget released | Same | Same |
| **Attribution & Correlation** | Preserves `X-Request-Id`, `X-Correlation-Id`, `OrganizationID`, `CredentialID`, `SubjectType`, `SubjectID`, `ProjectID`, `TeamID` | Same | Same |
| **Policy & Guardrails** | Evaluated at Stages 4.5 & 5 prior to upstream connection; violations return HTTP 403 `policy_denied` or `guardrail_blocked` | Same | Same |

---

## 3. Critical Fixes Applied in Phase E

### 1. HTTP Error Status Code Preservation in Stream Proxy
- **Defect Identified**: Previously, `usageObserver.finish()` returned `nil` when token counts were absent (`Reported == false`). Because provider error bodies (HTTP 429, 500, 503) do not contain token usage, `obs.finish()` returned `nil`. Consequently, `streamProxy.Forward()` returned `observed = nil`, and `emitTelemetry()` reverted to recording HTTP 200 with fallback estimated spend.
- **Resolution**: Updated [`backend/internal/interfaces/gatewayv2/usage_observer.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/usage_observer.go) to return `&StreamUsage` whenever `StatusCode != 0`, even when tokens are not reported.

### 2. Zero-Token Enforcement on Failed Requests
- **Defect Identified**: Requests resulting in HTTP 4xx/5xx from upstream were erroneously falling back to pre-flight token estimates, polluting user spend records.
- **Resolution**: Updated [`backend/internal/interfaces/gatewayv2/gateway.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/gateway.go) so that when `actualStatusCode >= 400`, `InputTokens = 0`, `OutputTokens = 0`, `CostMicros = 0`, and `CostStatus = domain.CostStatusUnavailable`.

---

## 4. Test Suite Execution & Status

### Deterministic Mock-Provider Suite
* File: [`backend/internal/interfaces/gatewayv2/provider_lifecycle_e2e_test.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/provider_lifecycle_e2e_test.go)
* Results:
  * `TestProviderLifecycle_OpenAI_NonStreaming`: **PASS**
  * `TestProviderLifecycle_OpenAI_Streaming`: **PASS**
  * `TestProviderLifecycle_Anthropic_NonStreaming`: **PASS**
  * `TestProviderLifecycle_Anthropic_Streaming`: **PASS**
  * `TestProviderLifecycle_Gemini_NonStreaming`: **PASS**
  * `TestProviderLifecycle_Gemini_Streaming`: **PASS**
  * `TestProviderLifecycle_ProviderErrorsAndRateLimits`: **PASS** (429, 500, 503)
  * `TestProviderLifecycle_ClientDisconnectMidStream`: **PASS** (HTTP 499)
  * `TestProviderLifecycle_MissingUsageMetadataFallback`: **PASS**
  * `TestProviderLifecycle_ModelValidationAndPolicyDenial`: **PASS** (400 unsupported, 403 denied)

### Live Provider Status
* `Live_OpenAI`: **BLOCKED** (`OPENAI_API_KEY` not provisioned in local offline environment)
* `Live_Anthropic`: **BLOCKED** (`ANTHROPIC_API_KEY` not provisioned in local offline environment)
* `Live_Gemini`: **BLOCKED** (`GEMINI_API_KEY` not provisioned in local offline environment)
