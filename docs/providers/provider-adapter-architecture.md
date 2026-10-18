# Sentinel Multi-Provider Adapter Architecture & Compatibility Matrix

## 1. Overview and Design Principles

Sentinel provides unified AI governance, policy enforcement, rate limiting, and cost accounting across heterogeneous AI providers while preserving provider-native capabilities.

The multi-provider architecture is designed around three strict principles:
1. **Zero Silent Parameter Discarding**: Unsupported parameters or multimodal/tool requests on models lacking capability return explicit, typed errors (`provider.ErrUnsupportedCapability`) rather than being silently ignored.
2. **Backward-Compatible Layering**: The core [`Provider`](file:///home/sanju/ai-project/ai_gateway/backend/internal/ports/provider/provider.go) interface (credential validation, health check, model discovery) remains preserved. The extended [`InferenceProvider`](file:///home/sanju/ai-project/ai_gateway/backend/internal/ports/provider/provider.go) interface layers unified request conversion, capability querying, and normalized streaming/non-streaming inference.
3. **Provider-Specific Protocol Fidelity**: Adapters model provider differences explicitly (OpenAI SSE chunks, Anthropic event streams, Gemini content structures, and Hugging Face router endpoints) rather than forcing arbitrary lowest-common-denominator conversions.

---

## 2. Core Interfaces and Contracts

The provider contracts are defined in [`backend/internal/ports/provider/provider.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/ports/provider/provider.go).

### 2.1 Provider Interface (Lifecycle & Health)
```go
type Provider interface {
    ValidateCredentials(ctx context.Context, apiKey string) error
    HealthCheck(ctx context.Context, apiKey string) error
    ListModels(ctx context.Context, apiKey string) ([]Model, error)
}
```

### 2.2 InferenceProvider Interface (Inference & Capabilities)
```go
type InferenceProvider interface {
    Provider
    Name() string
    Kind() string
    Metadata() ProviderMetadata
    GetModelCapabilities(ctx context.Context, modelID string) (ModelCapabilities, error)
    Execute(ctx context.Context, apiKey string, req NormalizedRequest) (*NormalizedResponse, error)
    Stream(ctx context.Context, apiKey string, req NormalizedRequest) (NormalizedStream, error)
    EstimateTokens(ctx context.Context, apiKey string, req NormalizedRequest) (*TokenUsage, error)
}
```

### 2.3 Normalized Stream Contract
```go
type NormalizedStream interface {
    Next() (*StreamChunk, error)
    Close() error
}
```

---

## 3. Normalized Internal Representation

The internal normalized representation provides a canonical structure for all inference requests, responses, and token usages across providers:

### 3.1 Normalized Request ([`NormalizedRequest`](file:///home/sanju/ai-project/ai_gateway/backend/internal/ports/provider/provider.go))
| Field | Type | Description |
| :--- | :--- | :--- |
| `Model` | `string` | Target model identifier (e.g., `gpt-4o`, `claude-sonnet-4-5`, `gemini-1.5-pro`) |
| `Messages` | `[]Message` | Conversation history with roles (`system`, `user`, `assistant`, `tool`) and multimodal parts |
| `SystemPrompt` | `string` | System instruction or prompt |
| `Temperature` | `*float64` | Sampling temperature |
| `TopP` | `*float64` | Nucleus sampling factor |
| `MaxTokens` | `int` | Maximum generation tokens |
| `Stream` | `bool` | Flag requesting streaming SSE response |
| `Tools` | `[]ToolDefinition` | Tool/function calling schemas |
| `ToolChoice` | `any` | Tool choice constraint (`auto`, `required`, or specific function) |
| `ResponseFormat`| `*ResponseFormat` | Structured output enforcement (e.g. `json_object` or schema) |
| `StopSequences` | `[]string` | Stop generation sequences |
| `CustomParameters`| `map[string]any` | Provider-specific pass-through parameters |

### 3.2 Normalized Response ([`NormalizedResponse`](file:///home/sanju/ai-project/ai_gateway/backend/internal/ports/provider/provider.go))
| Field | Type | Description |
| :--- | :--- | :--- |
| `ID` | `string` | Provider completion or message ID |
| `Model` | `string` | Resolved model identifier |
| `Provider` | `string` | Canonical provider kind (`openai`, `anthropic`, `gemini`, `huggingface`) |
| `Text` | `string` | Final generated assistant text |
| `ToolCalls` | `[]ToolCall` | Model-initiated function calls with normalized JSON arguments |
| `Usage` | `TokenUsage` | Granular token breakdown |
| `FinishReason` | `string` | Reason for completion (`stop`, `length`, `tool_calls`, etc.) |

### 3.3 Token Usage ([`TokenUsage`](file:///home/sanju/ai-project/ai_gateway/backend/internal/ports/provider/provider.go))
- `PromptTokens`: Input prompt tokens
- `CompletionTokens`: Output generated tokens
- `TotalTokens`: Sum of input + output
- `CachedTokens`: Upstream prompt caching tokens (Anthropic / OpenAI / Gemini)
- `ReasoningTokens`: Dedicated reasoning/thinking tokens (OpenAI o1/o3, Gemini thoughts)

---

## 4. Provider Capability Matrix

The gateway verifies request requirements against the model's actual declared capabilities:

| Capability | Anthropic Claude | OpenAI | Google Gemini | Hugging Face Inference |
| :--- | :---: | :---: | :---: | :---: |
| **Text Generation** | Yes | Yes | Yes | Yes |
| **Streaming (SSE)** | Yes | Yes | Yes | Yes |
| **System Instructions** | Native Parameter | `system` / `developer` message | `systemInstruction` turn | `system` message |
| **Chat History** | `user` / `assistant` turns | `user` / `assistant` / `tool` | `user` / `model` turns | `user` / `assistant` turns |
| **Tool / Function Calling** | Yes (`tools`) | Yes (`tools`) | Yes (`functionDeclarations`)| Model-dependent (fails closed on generic endpoints) |
| **Structured JSON** | Tool-forced / JSON mode | JSON Object / Strict Schema | `responseMimeType: application/json` | Prompt instruction |
| **Multimodal (Vision)** | Yes (Base64 JPEG/PNG/WebP/GIF)| Yes (Image URL & Base64) | Yes (`inlineData` Blob) | Pipeline-dependent |
| **Embeddings** | No (External / Voyage) | Yes (`text-embedding-3-*`) | Yes (`text-embedding-004`)| Yes (`sentence-transformers/*`) |
| **Context Limit** | 200,000 tokens | 128,000 - 200,000 tokens | 1,048,576 - 2,097,152 tokens | 32,768 - 131,072 tokens |

---

## 5. Provider Adapters Implementation

### 5.1 Anthropic Claude Adapter
- **Location**: [`backend/internal/infrastructure/provider/anthropic/`](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/provider/anthropic/)
- **Authentication**: `x-api-key: <token>` and `anthropic-version: 2023-06-01`
- **Request Endpoint**: `POST /v1/messages`
- **Token Estimation**: Native `POST /v1/messages/count_tokens`
- **Streaming Parser**: Server-Sent Events with event type routing (`content_block_delta`, `message_delta`, `message_stop`).

### 5.2 OpenAI Adapter
- **Location**: [`backend/internal/infrastructure/provider/openai/`](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/provider/openai/)
- **Authentication**: `Authorization: Bearer <token>`
- **Request Endpoint**: `POST /v1/chat/completions`
- **Token Estimation**: Fast local heuristic estimator with fallback
- **Streaming Parser**: JSON stream decoder handling `chat.completion.chunk` and `[DONE]` events.

### 5.3 Google Gemini Adapter
- **Location**: [`backend/internal/infrastructure/provider/gemini/`](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/provider/gemini/)
- **Authentication**: `x-goog-api-key: <token>`
- **Request Endpoint**: `POST /v1beta/models/{model}:generateContent`
- **Token Estimation**: Native `POST /v1beta/models/{model}:countTokens`
- **Streaming Parser**: SSE decoder handling `alt=sse` streams and candidate content fragments.

### 5.4 Hugging Face Inference Providers Adapter
- **Location**: [`backend/internal/infrastructure/provider/huggingface/`](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/provider/huggingface/)
- **Authentication**: `Authorization: Bearer <token>`
- **Request Endpoint**: `POST /v1/chat/completions` (OpenAI-compatible inference routing)
- **Token Estimation**: Heuristic tokenizer character-to-token ratio calculation
- **Streaming Parser**: SSE scanner processing data lines and `[DONE]` delimiters.

---

## 6. Standardized Error Mapping

All provider adapters translate provider-specific errors into standardized sentinel errors defined in [`backend/internal/ports/provider/provider.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/ports/provider/provider.go):

| Upstream Status / Condition | Standard Sentinel Error | HTTP Gateway Status |
| :--- | :--- | :---: |
| 401 / 403 / Invalid Token | `provider.ErrInvalidCredentials` | 401 Unauthorized |
| 429 / Quota Exhaustion | `provider.ErrRateLimitExceeded` | 429 Too Many Requests |
| 502 / 503 / 504 / Overloaded | `provider.ErrProviderUnavailable` | 503 Service Unavailable |
| 404 / Model Not Found | `provider.ErrModelNotFound` | 404 Not Found |
| Feature unsupported by Model | `provider.ErrUnsupportedCapability` | 400 Bad Request |
| Prompt exceeds Model limits | `provider.ErrContextLimitExceeded` | 400 Bad Request |

---

## 7. Adding a New Provider Adapter

To add an additional provider adapter:
1. Implement [`Provider`](file:///home/sanju/ai-project/ai_gateway/backend/internal/ports/provider/provider.go) and [`InferenceProvider`](file:///home/sanju/ai-project/ai_gateway/backend/internal/ports/provider/provider.go) in a new package under `backend/internal/infrastructure/provider/<provider_name>/`.
2. Implement `Metadata()` defining connection URLs, authentication header format, and supported features.
3. Implement `GetModelCapabilities()` reflecting accurate context limits and supported flags.
4. Implement `Execute()`, `Stream()`, and `EstimateTokens()` converting to and from `NormalizedRequest` and `NormalizedResponse`.
5. Map provider-specific HTTP errors to standardized `provider.Err*` errors.
6. Register the adapter into [`application.Registry`](file:///home/sanju/ai-project/ai_gateway/backend/internal/application/provider/registry.go) in [`backend/internal/bootstrap/run.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/bootstrap/run.go).
7. Add unit tests and register the adapter in [`compatibility_test.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/provider/compatibility_test.go).
