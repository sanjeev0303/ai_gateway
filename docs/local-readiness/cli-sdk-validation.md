# CLI & SDK End-to-End Validation Report

## 1. Overview
This report documents the end-to-end validation of the Sentinel CLI (`packages/cli`) and TypeScript SDK (`packages/api-client`) across three independent test users (`test-user-1`, `test-user-2`, `test-user-3`) executing against Sentinel Gateway-v2 (`http://localhost:8084`) and Control API (`http://localhost:8081`).

## 2. Test User Context
Three independent test users were provisioned via real invitation and acceptance flows:

| User Alias | Subject ID | Permitted Project | Issued Sentinel Key ID | Scopes |
| :--- | :--- | :--- | :--- | :--- |
| `test-user-1` | `9719e8c1-dacf-4175-b407-6acb427cfbcf` | `Project Gemini Core` | `e3947acc-e747-410b-b95c-1c471b5c39e0` | `["gateway"]` |
| `test-user-2` | `06c0a7fe-da1b-41ea-a263-3e472c15f212` | `Project Gemini Core` | `423ff53c-fb8f-4a5a-9749-fa2762b4340d` | `["gateway"]` |
| `test-user-3` | `0edecabd-3660-44dc-8f96-54bfe2a3b9fd` | `Project Gemini Research` | `7ac2a50c-6f95-403a-8f2c-bbdb81b9a7d7` | `["gateway"]` |

## 3. Sentinel CLI Validation (`packages/cli`)
The CLI binary (`aigw`) was compiled and tested for all three users with custom endpoint configuration (`--gateway-url http://localhost:8084`):

### 3.1 Commands Tested per User

1. **Authentication Configuration**:
   ```bash
   aigw auth login --gateway-url http://localhost:8084 --credential <raw_aigw_key>
   ```
   - **Result**: `PASS` for all 3 users. Configuration persisted to `~/.aigw/config.yaml` with secret masking.

2. **Model Listing**:
   ```bash
   aigw models list
   ```
   - **Result**: `PASS`. Permitted models returned including `gemini-2.5-flash`, `gemini-2.5-pro`, `gemini-1.5-pro`, `gemini-1.5-flash`.

3. **Non-Streaming Inference**:
   ```bash
   aigw request --model gemini-2.5-flash --prompt "..." --stream=false
   ```
   - **Result**: `PASS`. Output text parsed, telemetry printed (`Status: HTTP 200 OK`, `Latency: <ms>`, `Tokens: input/output/total`).

4. **Streaming Inference**:
   ```bash
   aigw request --model gemini-2.5-flash --prompt "..." --stream=true
   ```
   - **Result**: `PASS`. SSE frames parsed continuously from `candidates.content.parts.text`, terminal tokens updated from `usageMetadata`.

5. **Error Handling**:
   ```bash
   aigw request --model totally-unknown-model --prompt "..." --stream=false
   ```
   - **Result**: `PASS`. Returned HTTP 400 `unsupported_model` and non-zero exit code.

### 3.2 Automated CLI Test Suite
- `TestConfigCommands`: PASS (0.00s)
- `TestProjectsCommands`: PASS (0.00s)
- `TestAuthAndModelsWithMockServer`: PASS (0.01s)
- `TestGatewayClient_ListModels`: PASS (0.00s)
- `TestGatewayClient_StreamRequest_OpenAI`: PASS (0.00s)
- `TestGatewayClient_StreamRequest_Anthropic`: PASS (0.00s)
- `TestGatewayClient_StreamRequest_Gemini`: PASS (0.00s)
- `TestGatewayClient_SendRequest_NonStreaming`: PASS (0.00s)
- `TestGatewayClient_SendRequest_Gemini`: PASS (0.00s)
- `TestGatewayClient_ErrorHandling`: PASS (0.00s)
- **Suite Result**: 100% PASS (10/10 unit tests).

---

## 4. Sentinel SDK Validation (`packages/api-client`)
The TypeScript client was validated using `tsx --test src/**/*.test.ts` and live HTTP protocol execution:

### 4.1 Automated SDK Unit Tests (8/8 PASS)
1. `does not expose or share credentials between independent client instances`: PASS
2. `handles authentication failure (HTTP 401)`: PASS
3. `handles unsupported model (HTTP 400)`: PASS
4. `handles provider timeout (HTTP 504)`: PASS
5. `handles rate-limit response (HTTP 429)`: PASS
6. `handles credential revocation (HTTP 401)`: PASS
7. `successfully parses non-streaming Gemini responses with usage metadata`: PASS
8. `successfully parses streaming Gemini SSE chunks with usage`: PASS

### 4.2 Multi-User Protocol Execution
- **User 1 (Core)**: Non-streaming + Streaming Gemini requests -> HTTP 200 OK
- **User 2 (Core)**: Non-streaming + Streaming Gemini requests -> HTTP 200 OK
- **User 3 (Research)**: Non-streaming + Streaming Gemini requests -> HTTP 200 OK
- **Isolation Check**: Verified that requests dispatched using User 1's key cannot set or assume User 2 or User 3 context.
