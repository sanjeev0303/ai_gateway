# Provider-Specific Features & Compatibility Matrix

## Overview

The Sentinel AI Gateway implements a **capability-aware translation layer** that preserves provider-specific capabilities, parameters, and metadata without breaking cross-provider compatibility or forcing all models into an artificial lowest-common-denominator schema.

```
                      +--------------------------------------+
                      |      Normalized Gateway Request      |
                      |  (Model, Messages, Tools, Custom)    |
                      +-------------------+------------------+
                                          |
                      +-------------------+------------------+
                      | ValidateSupportedParameters(adapter) |
                      +-------------------+------------------+
                         /         |              \
           [Anthropic]  /          | [OpenAI]      \  [Gemini]
                       v           v                v
          +---------------+ +---------------+ +---------------+
          | ThinkingConfig| |ReasoningEffort| |SafetySettings |
          | TopK, Metadata| |Penalty, Seed  | |TopK, CandCount|
          | ToolResult Blk| |Tool Role      | |FuncResponse   |
          +---------------+ +---------------+ +---------------+
```

---

## 1. Provider-Specific Parameters Matrix

| Provider | Supported Parameters | Parameter Schema | Native Upstream Parameter Mapping |
| :--- | :--- | :--- | :--- |
| **Anthropic Claude** | `thinking`<br>`top_k`<br>`stop_sequences`<br>`metadata`<br>`tool_choice` | `thinking: {type: string, budget_tokens: int}`<br>`top_k: int`<br>`stop_sequences: []string`<br>`metadata: {user_id: string}`<br>`tool_choice: string \| object` | `thinking`<br>`top_k`<br>`stop_sequences`<br>`metadata`<br>`tool_choice` |
| **OpenAI** | `reasoning_effort`<br>`frequency_penalty`<br>`presence_penalty`<br>`seed`<br>`parallel_tool_calls`<br>`logit_bias`<br>`user`<br>`metadata`<br>`store`<br>`service_tier` | `reasoning_effort: "low" \| "medium" \| "high"`<br>`frequency_penalty: float`<br>`presence_penalty: float`<br>`seed: int`<br>`parallel_tool_calls: bool`<br>`logit_bias: map[string]int`<br>`user: string`<br>`metadata: map[string]string`<br>`store: bool`<br>`service_tier: string` | `reasoning_effort`<br>`frequency_penalty`<br>`presence_penalty`<br>`seed`<br>`parallel_tool_calls`<br>`logit_bias`<br>`user`<br>`metadata`<br>`store`<br>`service_tier` |
| **Google Gemini** | `safety_settings`<br>`top_k`<br>`candidate_count`<br>`stop_sequences` | `safety_settings: []{category: string, threshold: string}`<br>`top_k: int`<br>`candidate_count: int`<br>`stop_sequences: []string` | `safetySettings`<br>`generationConfig.topK`<br>`generationConfig.candidateCount`<br>`generationConfig.stopSequences` |
| **Hugging Face** | `wait_for_model`<br>`use_cache` | `wait_for_model: bool`<br>`use_cache: bool` | `parameters.wait_for_model`<br>`parameters.use_cache` |

---

## 2. Strict Parameter Rejection Mechanics

Rather than silently dropping unrecognized parameters (which leads to debugging nightmares and non-deterministic agent loops), Sentinel validates custom parameters at two distinct architectural levels:

1. **Provider-Level Parameter Validation (`ValidateSupportedParameters`)**:
   - Checked immediately by the provider adapter before request construction.
   - If a request contains a parameter that the targeted provider adapter does not recognize (for example, sending `reasoning_effort` to Anthropic, or `thinking` to OpenAI), Sentinel fails fast with `ErrUnsupportedParameter`.
   - Error format: `provider: unsupported parameter: parameter "<key>" is not supported by <provider>`.

2. **Model-Level Capability Validation (`ModelCapabilities.ValidateRequest`)**:
   - If a parameter is recognized by the provider but the specific model does not support it (for example, setting `reasoning_effort` on `gpt-3.5-turbo` instead of `o1`/`o3-mini`), Sentinel rejects the request with `ErrUnsupportedCapability`.
   - Error format: `provider: unsupported capability: model "<model>" does not support reasoning_effort`.

---

## 3. Tool Calling & Message Turn Semantics

Sentinel prevents semantic degradation when translating conversation history and tool execution results:

### Anthropic Claude
- **Assistant Tool Use**: When an assistant turn contains tool calls, Sentinel translates each tool call into a structured `ContentBlock` of type `"tool_use"` with unique `id`, tool `name`, and valid `input` JSON payload.
- **Tool Result Handling**: In Anthropic Messages API, tool responses cannot be passed as raw text turns or foreign roles. Sentinel wraps each `RoleTool` message as a content block of `type: "tool_result"` referencing the corresponding `tool_use_id`.
- **Turn Alternation Coalescing**: Consecutive tool results (from parallel tool calls) are automatically merged into a single user turn containing multiple `tool_result` blocks, preserving Anthropic's strict turn-alternation requirement.

### OpenAI
- **Tool Calls**: Serialized into OpenAI's native `tool_calls` structure on assistant messages with `id`, `type: "function"`, and stringified `function.arguments`.
- **Tool Results**: Mapped directly to OpenAI's native `role: "tool"` turn with matching `tool_call_id` and stringified `content`.

### Google Gemini
- **Function Calling**: Mapped into Gemini's `FunctionDeclaration` inside `tools` and parsed from `FunctionCall` inside candidate content parts.
- **Function Responses**: When `RoleTool` is received, Sentinel translates the turn into Gemini's `role: "function"` with a `Part` containing `FunctionResponse{Name, Response}`, converting text or JSON outputs into structured maps.

### Hugging Face
- Serverless Hugging Face Inference endpoints do not support function calling or multimodal image parts. Sentinel marks `SupportsTools: false` and `SupportsVision: false` in `ModelCapabilities`.
- If a client supplies `Tools` or multimodal parts to Hugging Face, Sentinel immediately rejects the request with `ErrUnsupportedCapability` instead of silently stripping the tools.

---

## 4. Streaming Event & Metadata Preservation

When proxying streaming responses (`NormalizedStream`), Sentinel preserves telemetry and operational metadata:

1. **Provider Request Identifiers**:
   - Preserved in every `StreamChunk` and final `NormalizedResponse`.
   - Anthropic: `msg_01...`
   - OpenAI: `chatcmpl-...`
   - Gemini: `gemini-<version>-<timestamp>`
   - Hugging Face: upstream completion ID.

2. **Granular Token Usage Breakdown**:
   - `PromptTokens`: Total input tokens consumed.
   - `CompletionTokens`: Output tokens generated.
   - `TotalTokens`: Sum of input and output tokens.
   - `CachedTokens`: Tokens read from prompt caching (Anthropic `cache_read_input_tokens`, OpenAI `prompt_tokens_details.cached_tokens`, Gemini `cachedContentTokenCount`).
   - `ReasoningTokens`: Internal reasoning and thinking tokens (OpenAI `completion_tokens_details.reasoning_tokens`, Gemini `thoughtsTokenCount`).

3. **Incremental Tool Call Deltas**:
   - Tool calling arguments streamed in parts (`input_json_delta` in Anthropic, `delta.tool_calls` in OpenAI) are preserved in `StreamChunk.ToolCalls` without truncation.

---

## 5. CLI Command & Provider Isolation

Sentinel's interactive CLI (`sentinel`) guarantees that provider-specific features remain strictly partitioned:

- `/reasoning` is registered strictly with `Provider: "openai"` and requires model capability `"reasoning"`. Executing `/reasoning` when bound to Anthropic or Gemini yields an error: `command "/reasoning" is not available for active provider anthropic`.
- `/safety` is registered strictly with `Provider: "gemini"` and cannot be invoked on Anthropic or OpenAI.
- `/endpoint` and `/params` are registered strictly with `Provider: "huggingface"`.
- Slash commands are never rerouted to an unrelated provider simply because of similar naming.
