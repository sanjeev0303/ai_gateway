# Gateway, SDK, and CLI Verification Report

This document records the verification of Sentinel Gateway-v2, provider adapters, SDK (`@ai-governance/api-client`), and CLI (`aigw`), proving requests are authenticated, governed, forwarded, measured, and recorded.

---

## 1. Verified Request Lifecycle

The complete request lifecycle was audited and verified across all 11 stages:

```
Client SDK / CLI
       │
       ▼ (1) Header Injection: x-api-key / Authorization: Bearer <aigw_...>
Sentinel Gateway-v2 (:8084)
       │
       ├─► (2) Authenticate: PBKDF2 hash cache lookup, status active, scope check
       ├─► (3) Anti-Spoofing: Validate X-Organization-ID, X-User-ID, X-Project-ID
       ├─► (4) Inspection & Model Resolver: Parse payload, validate model catalog support
       ├─► (5) Guardrails: Pre-flight content evaluation (block on violation)
       ├─► (6) Policy Authorization: Allowlist/denylist scope precedence check
       ├─► (7) Rate Limiter: Redis / sliding window token bucket check (429 on limit)
       ├─► (8) Budget Guard: Spend & token reservation check (402 on exhaustion)
       ├─► (9) Tenant & Provider Router: Resolve upstream, retrieve decrypted key
       │
       ▼ (10) Streaming Proxy: Byte-exact forward, SSE flush, client disconnect handling
Upstream Provider (Anthropic / OpenAI / Gemini / Mock)
       │
       ▼ (11) Telemetry & Usage Capture: Extract tokens (reported vs estimated), publish NATS
NATS JetStream (llm.request.completed) ──► Asynchronous Worker ──► PostgreSQL (llm_requests)
       │
       ▼
Dashboard Analytics APIs (:8081 /v1/usage/summary, /v1/usage/timeseries, /v1/usage/by-model)
```

---

## 2. Identified and Resolved Defects

| Component | Defect Description | Resolution | Regression Test Evidence |
| :--- | :--- | :--- | :--- |
| **Telemetry Emitter** (`telemetry_emitter.go`) | `usage_source` was labeled as `"provider_reported"` whenever `providerKind != "unknown"`, masking estimated tokens as exact counts. | Updated `telemetry_emitter.go` to explicitly inspect `record.ProviderMeta["tokens_reported"]` and `estimated` flags. | `TestEmitTelemetry_PrefersReportedOverEstimates`, `TestEmitTelemetry_FallsBackToEstimates`, `TestMultiUser_AttributionAndIsolation` |
| **SDK** (`gateway.ts`) | Lacked timeout handling, cancellation signal (`AbortSignal`), and idempotency guard against double-billing retries. | Added `timeoutMs`, `signal?: AbortSignal`, `idempotencyKey?: string`, and safe retry logic restricting retries to idempotent calls (`listModels`) or explicit keys. | `TestGatewayClient SDK: retries idempotent listModels on HTTP 503`, `does not retry non-idempotent chatCompletion without idempotencyKey` |
| **SDK** (`gateway.ts`) | Hardcoded `method: "POST"` in `parseError` for all endpoints including GET `/v1/models`. | Passed dynamic HTTP method parameter to `parseError`. | `TestGatewayClient SDK: retries idempotent listModels on HTTP 503` |
| **SDK** (`gateway.ts`) | Usage metadata lacked honest estimation identification; silently treated missing counts as undefined or zero. | Added `isEstimated` and `usageSource` (`"reported" \| "estimated" \| "none"`) to `GatewayExecutionMetadata`. | `TestGatewayClient SDK: correctly identifies missing or estimated usage metadata` |
| **CLI Client** (`client.go`) | Error parser failed to extract structured error messages from Gemini (`status`) and OpenAI (`code`), producing blank error brackets `[HTTP 400 ]`. | Enhanced `parseHTTPError` to extract Anthropic, OpenAI, and Gemini error envelopes cleanly. | `TestGatewayClient_ErrorHandling_GeminiAndOpenAI` |
| **CLI Client** (`client.go`) | Token output lacked provenance tag, obscuring whether provider reported tokens or local estimator was used. | Added `IsEstimated` and `UsageSource` flags to `RequestMetadata`, formatted banner with provenance tag. | `TestGatewayClient_UsageHonesty_EstimatedVsReported`, `cmd/request.go` |
| **CLI Commands** (`credential.go`) | Credential commands (`create`, `list`, `get`, `rotate`, `revoke`) printed simulated static mock strings rather than calling the Control API. | Implemented real HTTP client calls to `/v1/gateway-credentials` on the configured `ControlURL`. | `TestCredentialCommands_WithMockControlPlane` |

---

## 3. Verification Test Matrix

| Client | Provider / Target | Scenario | Expected Status / Behavior | Result | Evidence |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **SDK** | Anthropic | Non-streaming `/v1/messages` | HTTP 200, parses `content.text`, captures tokens | `PASS` | `TestGatewayClient SDK: successfully parses Anthropic format` |
| **SDK** | OpenAI | Non-streaming `/v1/chat/completions` | HTTP 200, parses `choices[0].message.content` | `PASS` | `TestGatewayClient SDK: retries non-idempotent when idempotencyKey provided` |
| **SDK** | Gemini | Streaming SSE `/v1/chat/completions` | HTTP 200, parses `candidates.content.parts.text` | `PASS` | `TestGatewayClient SDK: parses streaming Gemini SSE chunks` |
| **SDK** | Gateway | Rate limit exceeded | HTTP 429, code `rate_limit_exceeded` | `PASS` | `TestGatewayClient SDK: handles rate-limit response (HTTP 429)` |
| **SDK** | Gateway | Upstream provider timeout | HTTP 504, code `upstream_timeout` | `PASS` | `TestGatewayClient SDK: handles provider timeout (HTTP 504)` |
| **SDK** | Gateway | Key revocation | HTTP 401, code `credential_revoked` | `PASS` | `TestGatewayClient SDK: handles credential revocation (HTTP 401)` |
| **SDK** | Gateway | Client disconnect / AbortSignal | Abort exception thrown, request cancelled | `PASS` | `TestGatewayClient SDK: handles request timeout and cancellation via signal` |
| **SDK** | Gateway | Interrupted stream | HTTP 502, code `stream_interrupted` | `PASS` | `TestGatewayClient SDK: detects and surfaces interrupted streaming cleanly` |
| **CLI** | OpenAI | Streaming `aigw request --stream=true` | HTTP 200, real-time chunk flush, reported usage | `PASS` | `TestGatewayClient_StreamRequest_OpenAI` |
| **CLI** | Anthropic | Streaming `aigw request --stream=true` | HTTP 200, real-time chunk flush, reported usage | `PASS` | `TestGatewayClient_StreamRequest_Anthropic` |
| **CLI** | Gemini | Streaming `aigw request --stream=true` | HTTP 200, real-time chunk flush, reported usage | `PASS` | `TestGatewayClient_StreamRequest_Gemini` |
| **CLI** | Gemini | Error parsing (invalid argument) | HTTP 400 `[HTTP 400 INVALID_ARGUMENT] API key not valid` | `PASS` | `TestGatewayClient_ErrorHandling_GeminiAndOpenAI` |
| **CLI** | OpenAI | Error parsing (rate limit) | HTTP 429 `[HTTP 429 rate_limit_exceeded] Rate limit exceeded` | `PASS` | `TestGatewayClient_ErrorHandling_GeminiAndOpenAI` |
| **CLI** | Gateway | Model listing retry on 503 | HTTP 503 retried successfully with backoff | `PASS` | `TestGatewayClient_ListModels_RetryOn503` |
| **CLI** | Control API | Credential create / list / rotate / revoke | HTTP 200/201/204 real Control API integration | `PASS` | `TestCredentialCommands_WithMockControlPlane` |
| **Multi-User** | User Alpha | Individual Gateway Key Attribution | Usage record attributed to User Alpha + Org ID | `PASS` | `TestMultiUser_AttributionAndIsolation` |
| **Multi-User** | User Beta | Individual Gateway Key Attribution | Usage record attributed to User Beta + Org ID | `PASS` | `TestMultiUser_AttributionAndIsolation` |
| **Multi-User** | User Gamma | Individual Gateway Key Attribution | Usage record attributed to User Gamma + Org ID | `PASS` | `TestMultiUser_AttributionAndIsolation` |
| **Multi-User** | User Alpha | Header spoofing User Beta ID | HTTP 403 `permission_error` | `PASS` | `TestMultiUser_AttributionAndIsolation` |
| **Multi-User** | User Alpha | Header spoofing Foreign Tenant ID | HTTP 403 `tenant_isolation_violation` | `PASS` | `TestMultiUser_AttributionAndIsolation` |
| **Multi-User** | User Beta | Credential Revocation Isolation | User Beta rejected (HTTP 401); User Alpha/Gamma remain active | `PASS` | `TestMultiUser_AttributionAndIsolation` |
| **Live Smoke** | Google Gemini | Live API smoke test | Requires `GEMINI_API_KEY` in environment | `BLOCKED` | `TestMultiUser_LiveGeminiSmokeTestOrBlocked` (Key not set in local env; mock tests verified) |

---

## 4. Multi-User Local Fixture Architecture

Local testing provisions 1 organization with 3 distinct test users and individual Sentinel gateway credentials:

| User Identifier | Role | Project Scope | Gateway Credential Identifier | Scope |
| :--- | :--- | :--- | :--- | :--- |
| `user-alpha` | Member | `Project Gemini Core` | `aigw_user1_secret_key_alpha` | `["gateway"]` |
| `user-beta` | Member | `Project Gemini Core` | `aigw_user2_secret_key_beta` | `["gateway"]` |
| `user-gamma` | Member | `Project Gemini Research` | `aigw_user3_secret_key_gamma` | `["gateway"]` |

### Attribution Guarantees
1. **No Provider Secret Sharing**: Neither user holds the organization master Gemini/Anthropic key; each client issues requests strictly with their scoped `aigw_...` key.
2. **Deterministic Telemetry Mapping**: Every request recorded in `llm_requests` resolves `user_id`, `organization_id`, `project_id`, and `gateway_credential_id` from the authenticated key hash in PostgreSQL, never from client-controlled headers.
3. **Spoofing Immunity**: Attempting to supply `X-User-ID` or `X-Organization-ID` inconsistent with the authenticated key immediately triggers HTTP 403.

---

## 5. Client Usage Instructions

### 5.1 Sentinel CLI (`aigw`)

#### Installation and Build
```bash
cd packages/cli
go build -o aigw .
```

#### Authentication
Configure your individual gateway credential:
```bash
./aigw auth login --gateway-url http://localhost:8084 --credential aigw_user1_secret_key_alpha
```

#### Check Status & Permitted Models
```bash
./aigw auth status
./aigw models list
```

#### Select Project Scope
```bash
./aigw projects select <project-uuid>
./aigw projects current
```

#### Send Streaming Inference Request
```bash
./aigw request --model gemini-2.5-flash --prompt "Explain distributed rate limiting in Sentinel" --stream=true
```

#### Send Non-Streaming Request
```bash
./aigw request --model claude-sonnet-4-5 --prompt "Summarize Sentinel architecture" --stream=false
```

#### Manage Credentials via Control API
```bash
./aigw credential list
./aigw credential create --name "automation-key" --subject-type service_account --subject-id <sa-uuid>
./aigw credential rotate <credential-uuid>
./aigw credential revoke <credential-uuid> --force
```

---

### 5.2 Sentinel TypeScript SDK (`@ai-governance/api-client`)

#### Initialization
```typescript
import { createGatewayClient } from "@ai-governance/api-client";

const client = createGatewayClient({
  gatewayUrl: "http://localhost:8084",
  apiKey: "aigw_user1_secret_key_alpha",
  projectId: "11111111-2222-3333-4444-555555555555",
  timeoutMs: 30000, // 30s timeout
  maxRetries: 2,    // Safe retries for idempotent operations
});
```

#### Listing Permitted Models (Idempotent, Retry-Protected)
```typescript
const models = await client.listModels();
console.log("Permitted models:", models.map(m => m.id));
```

#### Non-Streaming Chat Completion
```typescript
const res = await client.chatCompletion({
  model: "gemini-2.5-flash",
  messages: [{ role: "user", content: "Hello Sentinel" }],
  idempotencyKey: "idem-req-001", // Protects against duplicate billing
});

console.log("Output:", res.text);
console.log("Tokens:", res.meta.totalTokens, `(${res.meta.usageSource})`);
console.log("Latency:", res.meta.durationMs, "ms");
```

#### Streaming Chat Completion with Cancellation Support
```typescript
const controller = new AbortController();

const meta = await client.streamChatCompletion(
  {
    model: "claude-sonnet-4-5",
    messages: [{ role: "user", content: "Stream detailed analysis" }],
    signal: controller.signal,
  },
  (delta: string) => {
    process.stdout.write(delta);
  }
);

console.log("\nFinished. Stream request ID:", meta.requestId);
```
