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

---

## 3. Server-Side SSRF Protection on Hugging Face Endpoints
Custom inference endpoints passed via `/endpoint` or request options are strictly validated server-side by `ValidateHFEndpoint` (`provider_router.go`):
- Scheme must be HTTPS (`https://`)
- IP addresses (loopback, private, public, link-local) are strictly prohibited
- Cloud metadata hostnames (`metadata.google.internal`, `instance-data`) are rejected
- Hostnames must belong to verified domains: `*.endpoints.huggingface.cloud` or `*.huggingface.co`

---

## 4. Test Verification Evidence
- `go test -v ./internal/infrastructure/provider/...`: All unit, contract, and adapter integration tests pass.
- `TestPhaseJ_GatewayInferenceRoutingAndGovernance`: E2E verification of all 4 providers in Gateway V2 data plane.
- `TestPhaseG_CLI_EndToEndScenario`: E2E verification of CLI interaction with multi-provider credentials.
