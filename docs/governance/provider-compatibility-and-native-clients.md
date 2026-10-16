# Provider Compatibility, Subscription Boundaries, and Native Client Configuration

## Executive Summary

Sentinel AI Gateway (`gateway-v2`) provides a unified, policy-governed data plane connecting internal employees, service accounts, and automated workloads to multi-provider AI backends (Anthropic, OpenAI, Gemini, Azure OpenAI, and Custom models).

This document establishes the technical architecture, official configuration patterns for native developer clients, and the compliance boundaries governing provider subscription models.

---

## 1. Provider Compatibility Architecture

The Sentinel Gateway operates as a reverse proxy and policy enforcement point:

| Provider | Supported Protocols & Endpoints | Native Header Contract | Upstream Credential Source |
|---|---|---|---|
| **Anthropic** | `POST /v1/messages`, `GET /v1/models` | `x-api-key: aigw_...`, `anthropic-version` | Organization Provider Account (encrypted Vault) |
| **OpenAI** | `POST /v1/chat/completions`, `POST /v1/embeddings`, `GET /v1/models` | `Authorization: Bearer aigw_...` | Organization Provider Account (encrypted Vault) |
| **Gemini** | `POST /v1beta/models/{model}:generateContent`, `POST /v1/chat/completions` (OpenAI format) | `x-goog-api-key`, `Authorization: Bearer aigw_...` | Organization Provider Account (encrypted Vault) |
| **Azure OpenAI** | `POST /openai/deployments/{deployment}/chat/completions`, OpenAI route | `api-key: aigw_...`, `Authorization: Bearer aigw_...` | Organization Provider Account (encrypted Vault) |
| **Custom / Self-Hosted** | `POST /v1/chat/completions`, `POST /v1/completions` | `Authorization: Bearer aigw_...` | Endpoint config / Custom Auth Token |

### Dynamic Route & Model Resolution
1. **Model Resolution**: Clients request canonical model identifiers (`gpt-4o`, `claude-sonnet-4-5`, `gemini-2.0-flash`).
2. **Provider Detection**: The gateway inspects payload structure and model prefixes to determine the target provider.
3. **Policy Authorization**: Scoped governance rules (allowlists, rate limits, budgets) are evaluated before connecting upstream.
4. **Credential Translation**: The gateway injects the organization's encrypted provider credentials into the upstream request, stripping developer credentials.

---

## 2. Consumer Web Subscriptions vs. Enterprise API Credentials

### Architectural Incompatibility of Browser Subscriptions
Consumer web subscriptions (such as ChatGPT Plus/Pro, Claude Pro/Team web interface, or Google Gemini Advanced) are architecturally distinct from API access:

1. **Ephemeral Browser Sessions**: Consumer interfaces use short-lived, browser-bound session cookies and OAuth PKCE tokens protected by anti-bot verification (Cloudflare Turnstile, browser fingerprinting, and device attestation).
2. **Strict Prohibition on Impersonation / Scraping**:
   - Anthropic and OpenAI Terms of Service explicitly prohibit automated scraping, reverse-engineering web session cookies, and proxying web interfaces.
   - Sharing web accounts or scraping session cookies risks permanent termination of organizational domains.
3. **No Multi-Tenant Governance**: Web sessions cannot attribute individual requests to internal projects, teams, or service accounts, making auditability, budget tracking, and real-time rate limiting impossible.
4. **Lack of Programmatic API Guarantees**: Web endpoints lack SLA guarantees, structured JSON schemas, predictable token tracking, or byte-exact SSE event streams.

### Policy Directive
> **MANDATORY POLICY**: Sentinel AI Gateway strictly forbids browser session scraping or OAuth cookie impersonation. All employee and service account traffic must route through legitimate enterprise API credentials configured in the Sentinel Platform dashboard.

---

## 3. Native Client Configuration

Developers can configure official vendor SDKs and developer CLI tools to route through Sentinel without modifying application logic.

### Environment Variable Standard

Configure the following environment variables in shell profiles (`~/.bashrc`, `~/.zshrc`) or local development `.env` files:

#### OpenAI CLI & Official SDKs
```bash
export OPENAI_BASE_URL="http://localhost:8084/v1"
export OPENAI_API_KEY="aigw_live_employee_key_here"

# Optional: bind to a specific project scope
export AIGW_PROJECT_ID="00000000-0000-0000-0000-000000000001"
```

#### Anthropic CLI & Official SDKs
```bash
export ANTHROPIC_BASE_URL="http://localhost:8084"
export ANTHROPIC_API_KEY="aigw_live_employee_key_here"
```

#### Python Native Example
```python
import os
from openai import OpenAI

client = OpenAI(
    base_url=os.getenv("OPENAI_BASE_URL", "http://localhost:8084/v1"),
    api_key=os.getenv("OPENAI_API_KEY"),
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Hello via Sentinel!"}],
)
print(response.choices[0].message.content)
```

---

## 4. Sentinel CLI (`aigw`)

For a seamless employee access experience, Sentinel provides a dedicated native CLI (`packages/cli`):

### Installation & Quickstart
```bash
# Build / Install binary
go install ai-governance-saas/packages/cli@latest

# Configure gateway connection & authenticate
aigw auth login --gateway-url http://localhost:8084 --credential aigw_...

# Inspect permitted models (filtered by corporate policy)
aigw models list

# Switch project context
aigw projects select 00000000-0000-0000-0000-000000000001

# Run streaming prompt with execution metadata
aigw chat --model gpt-4o --prompt "Analyze system metrics"
```

### Features
- **Real-Time Streaming**: Byte-exact SSE event parsing for OpenAI and Anthropic streams.
- **Strict Governance**: Rejects unpermitted models with descriptive policy denial reasons.
- **Execution Telemetry**: Displays latency, HTTP status, request ID, and input/output token counts in execution footers.
- **Safe Credential Management**: Stores keys in `~/.aigw/config.yaml` with mode `0600` permissions.
