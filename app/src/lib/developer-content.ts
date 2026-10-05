import type { CodeSnippet, SdkIntegration } from "@/types";

const GATEWAY_URL = "http://localhost:8084";

// Routes, auth header, and payload shape come from the real implementation:
// gatewayv2/server.go (routes), gateway.go (x-api-key), request_inspector.go
// (model/messages/max_tokens). The gateway is Anthropic-compatible and has no
// OpenAI-compatible surface, and no SDK is published, so these call /v1/messages.
export const codeSnippets: CodeSnippet[] = [
  {
    id: "snp_1",
    label: "Send a message",
    language: "bash",
    code: `curl ${GATEWAY_URL}/v1/messages \\
  -H "x-api-key: $AIGW_API_KEY" \\
  -H "content-type: application/json" \\
  -d '{
    "model": "claude-sonnet-4-5",
    "max_tokens": 1024,
    "messages": [{ "role": "user", "content": "Summarize this incident." }]
  }'`,
  },
  {
    id: "snp_2",
    label: "TypeScript",
    language: "typescript",
    code: `const response = await fetch("${GATEWAY_URL}/v1/messages", {
  method: "POST",
  headers: {
    "x-api-key": process.env.AIGW_API_KEY!,
    "content-type": "application/json",
  },
  body: JSON.stringify({
    model: "claude-sonnet-4-5",
    max_tokens: 1024,
    messages: [{ role: "user", content: "Summarize this incident." }],
  }),
});

const message = await response.json();
console.log(message);`,
  },
  {
    id: "snp_3",
    label: "Python",
    language: "python",
    code: `import os
import httpx

response = httpx.post(
    "${GATEWAY_URL}/v1/messages",
    headers={"x-api-key": os.environ["AIGW_API_KEY"]},
    json={
        "model": "claude-sonnet-4-5",
        "max_tokens": 1024,
        "messages": [{"role": "user", "content": "Summarize this incident."}],
    },
)

print(response.json())`,
  },
];

// Private workspace packages, versioned in lockstep at 0.0.0. docsUrl is a repo
// path, not a link.
export const sdkIntegrations: SdkIntegration[] = [
  {
    id: "sdk_1",
    name: "@ai-governance/api-client",
    language: "TypeScript",
    latestVersion: "0.0.0",
    docsUrl: "packages/api-client",
  },
  {
    id: "sdk_2",
    name: "@ai-governance/contracts",
    language: "TypeScript",
    latestVersion: "0.0.0",
    docsUrl: "packages/contracts",
  },
];