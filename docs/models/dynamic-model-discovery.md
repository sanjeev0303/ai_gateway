# Dynamic Model Discovery and Registry Architecture

## 1. Product Objective & Core Rules

Sentinel implements **provider-aware dynamic model discovery** governed by strict tenant isolation and Sentinel policy access controls.

### Core Guarantees
1. **No Static Fallback Catalog**: The gateway and control API **never** populate the selectable model list with a hardcoded catalog of models that the organization cannot access. If an organization has no active provider accounts connected, `/v1/models` returns an empty collection `[]`.
2. **Provider Connection Binding**: Models are only visible and routable if an active organization provider connection with a valid credential exists for that provider.
3. **Multi-Tenant Isolation**: Two organizations with different provider connections (e.g. Org A with Anthropic, Org B with OpenAI) will each strictly see only their connected models, and requests to unconnected providers are rejected with `503 upstream_unavailable`.
4. **Stale Model Deprecation**: When a provider's model catalog is refreshed, any previously registered models that the upstream API no longer reports are preserved in historical records but marked `is_active = false` and `status = 'inactive'`, preventing routing while preserving usage analytics auditability.

---

## 2. Model Discovery Workflow

When an administrator connects or synchronizes a provider account:
1. **Credential Validation**: Stored credentials in the `SecretStore` are resolved without exposing secrets to frontend or logs.
2. **Upstream Discovery**:
   - **OpenAI**: Queries `GET /v1/models`, mapping model identifiers, reasoning flags (`o1`, `o3`), and streaming/tool capabilities.
   - **Google Gemini**: Queries `GET /v1beta/models`, extracting `supportedGenerationMethods` (`generateContent`, `countTokens`), and `inputTokenLimit` context boundaries.
   - **Hugging Face**: Queries `GET https://huggingface.co/api/models?pipeline_tag=text-generation` or dedicated endpoints (`/models`).
   - **Anthropic Claude**: Queries `GET /v1/models` where supported, falling back to explicit provider-specific verified models (`claude-3-7-sonnet`, `claude-3-5-sonnet`, `claude-3-5-haiku`, `claude-3-opus`) when the upstream discovery endpoint returns 404/501.
3. **Capability Extraction**: Upstream metadata is parsed into structured capabilities (`streaming`, `tool_calling`, `reasoning`, `multimodal`) and exact token context limits.
4. **Catalog Persistence**:
   - Discovered models are upserted into PostgreSQL using `UpsertModelWithMetadata`.
   - Discontinued models for that provider kind are marked inactive using `MarkModelsInactiveExcept`.

---

## 3. Metadata Schema

Model records in `models` table and `domain.Model` store:

| Field | Type | Description |
| :--- | :--- | :--- |
| `id` | `UUID` | Unique model record identifier |
| `provider_kind` | `VARCHAR(32)` | Upstream provider kind (`anthropic`, `openai`, `gemini`, `huggingface`) |
| `model_slug` | `VARCHAR(128)` | Exact wire model identifier required by upstream API |
| `display_name` | `VARCHAR(255)` | Human-readable model title |
| `capabilities` | `TEXT[]` | Discovered inference capabilities (`text_generation`, `streaming`, `tool_calling`, `reasoning`) |
| `context_limit` | `INTEGER` | Context window size in tokens |
| `supports_streaming` | `BOOLEAN` | Whether model supports token streaming |
| `supports_tools` | `BOOLEAN` | Whether model supports function/tool calling |
| `supports_reasoning` | `BOOLEAN` | Whether model supports extended thinking / reasoning |
| `status` | `VARCHAR(32)` | Lifecycle status (`active`, `inactive`, `deprecated`) |
| `pricing_version` | `VARCHAR(32)` | Governance pricing version reference |
| `is_active` | `BOOLEAN` | Active routability flag |
| `created_at` / `updated_at` | `TIMESTAMPTZ` | Timestamp audit records |

---

## 4. Gateway Authorization & Pipeline Execution

An inference request is only permitted when **all 6** applicable conditions hold:
1. **Tenant Identity Authenticated**: Valid Sentinel gateway API key with `gateway` scope.
2. **Sentinel Policy Compliance**: Org and user policies admit the requested model (`policy.Authorize`).
3. **Model Active Status**: Model exists in the catalog and is marked `is_active = true` and `status != 'inactive'`.
4. **Active Provider Connection**: Tenant has an active provider account for the requested provider (`provider_accounts.status = 'active'`).
5. **Valid Provider Credential**: Active credential exists for that connection (`provider_credentials.status = 'active'`).
6. **Budget & Rate Limits**: Window counters have not exceeded rate limits or budget caps.

### Execution Stages in `executePipeline`
- **Stage 1**: Authenticate client API key; reject spoofed tenant or project headers.
- **Stage 2**: Inspect request body for target model and extract hints.
- **Stage 3**: Load immutable policy snapshot for the organization.
- **Stage 4**: Evaluate Sentinel policies; return `403 policy_denied` on rule violation.
- **Stage 5**: Check model availability and active status in model catalog; return `400 unsupported_model` if inactive.
- **Stage 6**: Resolve provider upstream connection; return `503 upstream_unavailable` if provider not connected or credential revoked.
- **Stage 7**: Rate limit checks across org, credential, and user scopes.
- **Stage 8**: Reserve estimated token budget in Redis.
- **Stage 9**: Route and forward inference request via `StreamProxy`.

---

## 5. Automated Verification Suites

The discovery and authorization matrix is covered by automated regression tests:

- **`backend/internal/application/provider/discovery_test.go`**:
  - `TestDynamicModelDiscovery/discovers_capabilities_and_marks_stale_models_inactive`: Verifies capability extraction and `MarkInactiveExcept` execution.
  - `TestDynamicModelDiscovery/dummy_key_populates_rich_metadata_across_providers`: Verifies metadata defaults for mock and development keys.
  - `TestDynamicModelDiscovery/SyncAccountModels_loads_stored_credential_and_syncs`: Verifies end-to-end sync using stored account credentials.
- **`backend/internal/interfaces/gatewayv2/discovery_test.go`**:
  - `TestMultiOrgModelDiscoveryAndIsolation`:
    - **Org A (Anthropic)**: `/v1/models` returns only Anthropic models. Requests to `gpt-4o` return `503 upstream_unavailable`.
    - **Org B (OpenAI)**: `/v1/models` returns only OpenAI models. Requests to `claude-3-5-sonnet` return `503 upstream_unavailable`.
    - **Org C (Unconnected)**: `/v1/models` returns empty list `[]`, verifying zero static fallbacks.
    - **Inactive Model**: Requests to deprecated `claude-2.1` return `400 unsupported_model`.
    - **Revocation**: Disabling or revoking provider credentials immediately purges models from catalog and rejects routing.
