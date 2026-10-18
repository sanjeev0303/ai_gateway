# Sentinel CLI Provider-Aware Slash Commands Architecture

This document specifies the provider-aware interactive slash command architecture in the Sentinel CLI (`sentinel`), detailing the command registry, provider applicability, conversation state semantics, and capability resolution.

---

## 1. Overview & Architecture

The Sentinel CLI features a centralized, provider-aware command registry (`CommandRegistry`) that dynamically resolves the active credential's authorization scope and model capabilities to tailor the interactive REPL experience.

```
+-------------------------------------------------------------+
| Interactive Input: User types "/" or "/<command>"           |
+-------------------------------------------------------------+
                              |
                              v
+-------------------------------------------------------------+
| CommandRegistry: Resolves Active Provider & Model Capabilit.|
+-------------------------------------------------------------+
         |                                           |
         v                                           v
[Common Sentinel Commands]              [Provider-Specific Commands]
- /help, /provider, /model              - Anthropic: /max-tokens
- /status, /usage, /cost                - OpenAI: /reasoning (o1/o3)
- /permissions, /clear, /compact        - Gemini: /safety
- /stream, /system, /temp, /exit        - Hugging Face: /endpoint, /params
                              |
                              v
+-------------------------------------------------------------+
| Execution Engine: Local Validation & Gateway API Forwarding |
+-------------------------------------------------------------+
```

Each command defines:
- **Identifier & Aliases**: Canonical slash command name and shortcuts.
- **Provider Applicability**: Common across all providers or restricted to specific providers (`anthropic`, `openai`, `gemini`, `huggingface`).
- **Capability Requirements**: Conditional features such as `reasoning` (e.g. o1/o3-mini models).
- **Execution Target**: Explicit distinction between local client state mutations and authenticated Gateway API calls.
- **Parameter Validation**: Strict client-side argument verification before submission.

---

## 2. Command Set by Provider

### A. Common Sentinel Commands (All Providers)

| Command | Args | Execution | Description |
| :--- | :--- | :--- | :--- |
| `/help`, `/?` | `[cmd]` | Local | Display available commands for the active provider, or show syntax for a specific command. |
| `/` | — | Local | Display the formatted interactive slash-command menu. |
| `/provider` | `[name]` | Local | Show active provider or switch to another authorized provider connection. |
| `/model` | `[model_id]` | Gateway | List permitted models from the gateway, or switch active model within permitted scope. |
| `/status` | — | Local | Inspect Gateway URL, masked credential, active provider/model, streaming status, and session spend. |
| `/usage`, `/tokens` | — | Local | View cumulative session tokens (prompt, completion, total) and message turn count. |
| `/cost` | — | Local | Display cumulative estimated session cost in USD based on exact token rates. |
| `/permissions`, `/auth` | — | Local | Inspect effective authorization boundaries, bound provider connections, and accessible models. |
| `/clear` | — | Local | Clear multi-turn conversation messages from context and reset session usage counters. |
| `/compact` | — | Gateway | Compress prior conversation turns into a concise summary using the active model to free tokens. |
| `/stream` | `[on\|off]` | Local | Toggle or configure streaming token responses. |
| `/system` | `[prompt]` | Local | Inspect or update system prompt instructions for subsequent requests. |
| `/temp` | `[number]` | Local | Configure sampling temperature (0.0–1.0 for Claude; 0.0–2.0 for OpenAI/Gemini/HF). |
| `/exit`, `/quit` | — | Local | Exit the interactive Sentinel CLI session. |

---

### B. Anthropic-Specific Commands

When the active provider is `anthropic` (or the credential is Anthropic-bound):

| Command | Args | Execution | Description |
| :--- | :--- | :--- | :--- |
| `/max-tokens` | `<int>` | Local | Set maximum token completion limit for Claude models (e.g. `/max-tokens 4096`). |
| `/temp` | `[0.0-1.0]` | Local | Set sampling temperature bounded to Anthropic's valid 0.0–1.0 range. |

---

### C. OpenAI-Specific Commands

When the active provider is `openai`:

| Command | Args | Execution | Description |
| :--- | :--- | :--- | :--- |
| `/reasoning` | `[low\|medium\|high\|off]` | Local | Set reasoning effort for OpenAI reasoning models (`o1`, `o1-preview`, `o3-mini`). Filtered out for non-reasoning models. |
| `/temp` | `[0.0-2.0]` | Local | Set sampling temperature bounded to OpenAI's valid 0.0–2.0 range. |

---

### D. Google Gemini-Specific Commands

When the active provider is `gemini`:

| Command | Args | Execution | Description |
| :--- | :--- | :--- | :--- |
| `/safety` | `[standard\|strict\|permissive]` | Local | Configure Gemini safety threshold presets (`BLOCK_LOW_AND_ABOVE`, `BLOCK_MEDIUM_AND_ABOVE`, `BLOCK_ONLY_HIGH`). |
| `/temp` | `[0.0-2.0]` | Local | Set sampling temperature bounded to Gemini's 0.0–2.0 range. |

---

### E. Hugging Face-Specific Commands

When the active provider is `huggingface`:

| Command | Args | Execution | Description |
| :--- | :--- | :--- | :--- |
| `/endpoint` | `[url]` | Local | View or set custom dedicated Hugging Face Inference Endpoint URL. |
| `/params` | — | Local | Display active decoding parameters (temperature, max_tokens, streaming mode). |

---

## 3. Interactive Menu Experience

Typing `/` or `/?` generates a clean, tabular command menu grouped by provider-specific and common commands:

```text
sentinel [anthropic/claude-sonnet-4-5]> /
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Sentinel CLI Slash Commands:
  Active Provider: ANTHROPIC
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  [ANTHROPIC Specific Commands]
    /max-tokens <number>      [Local]    Set maximum token limit for Claude completion responses

  [Common Sentinel Commands]
    /clear                    [Local]    Reset conversation message history and session usage counters
    /compact                  [Gateway]  Compact conversation history via gateway summarization to save context tokens
    /cost                     [Local]    Show cumulative estimated session cost in USD
    /exit                     [Local]    Exit the interactive session
    /help [command]           [Local]    Display available slash commands for the active provider and model
    /model [model_id]         [Gateway]  Display active model, select model, or list permitted models from gateway
    /permissions              [Local]    Inspect effective authorization scope, bound provider account, and limits
    /provider [name]          [Local]    Display current provider or switch authorized provider connection
    /status                   [Local]    Show gateway URL, masked credential, active provider/model, and session telemetry
    /stream [on|off]          [Local]    Toggle or set streaming response mode
    /system [instructions]    [Local]    Set or inspect active system prompt instructions
    /temp [number]            [Local]    Set or inspect sampling temperature (0.0 to 1.0 for Claude, 0.0 to 2.0 for others)
    /usage                    [Local]    Show cumulative session token usage breakdown

Type /<command> to execute, or /help <command> for details.
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

---

## 4. Conversation State & Compaction Semantics

The Sentinel CLI tracks genuine multi-turn conversation history in `SessionState.Messages`:

### A. Reset Semantics (`/clear`)
- Discards all accumulated `ChatMessage` entries from the local context window.
- Resets cumulative input, output, and total token counters to 0.
- Resets session spend estimation to $0.00.
- Informs the user of the exact number of purged messages.

### B. Compaction Semantics (`/compact`)
- **Minimal State Guard**: If conversation contains $\le 2$ turns, reports that context is already minimal and skips redundant network calls.
- **Gateway LLM Summarization**: When $> 2$ turns exist:
  1. Compiles prior message history into a distillation prompt.
  2. Dispatches a summarization request through `GatewayClient.SendRequest` using the active model and provider credentials.
  3. Replaces earlier turns with a single structured assistant message: `[Compacted Conversation Context: <summary>]` while preserving the user's latest prompt turn.
  4. Records token usage from the summarization call into session metrics.
- **Graceful Fallback**: If the gateway is unreachable during compaction, the CLI performs deterministic local truncation, keeping the recent context without crashing.

---

## 5. Scope & Permission Enforcement

The CLI automatically invokes `SyncPermittedScope` upon initialization:
1. **Model & Provider Discovery**: Queries `/v1/models` using the authenticated gateway key.
2. **Provider Bounding**:
   - If the credential is bound to a single provider (e.g. Anthropic), `/provider openai` is rejected with:
     ```text
     provider "openai" is not authorized for this credential (authorized: anthropic)
     ```
3. **Model Restrictions**:
   - If `AllowedModels` restricts access to `gemini-1.5-flash`, attempting `/model gemini-1.5-pro` is rejected:
     ```text
     model "gemini-1.5-pro" is not permitted for this credential
     ```
4. **Dynamic Menu Shift**:
   - Switching provider via `/provider <name>` immediately updates the active command set and interactive prompt:
     ```text
     sentinel [openai/gpt-4o]>
     ```
