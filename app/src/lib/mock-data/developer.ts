import { NOW } from "./clock";
import type { CodeSnippet, DeveloperApiKey, SdkIntegration } from "@/types";

const DAY = 86_400_000;
const HOUR = 3_600_000;

const daysAgo = (n: number) => new Date(NOW.getTime() - n * DAY).toISOString();
const hoursAgo = (n: number) => new Date(NOW.getTime() - n * HOUR).toISOString();
const minutesAgo = (n: number) => new Date(NOW.getTime() - n * 60_000).toISOString();
const daysAhead = (n: number) => new Date(NOW.getTime() + n * DAY).toISOString();

export const developerApiKeys: DeveloperApiKey[] = [
  {
    id: "key_1",
    name: "Production gateway",
    maskedKey: "sk_live_...7f3a",
    environment: "production",
    scopes: ["gateway:read", "gateway:write", "usage:read", "budgets:read"],
    createdAt: daysAgo(210),
    lastUsedAt: minutesAgo(1),
    status: "active",
    requestsLast24h: 412_880,
  },
  {
    id: "key_2",
    name: "Staging integration tests",
    maskedKey: "sk_test_...c41d",
    environment: "staging",
    scopes: ["gateway:read", "gateway:write"],
    createdAt: daysAgo(140),
    lastUsedAt: hoursAgo(5),
    status: "active",
    requestsLast24h: 8_240,
  },
  {
    id: "key_3",
    name: "Analytics export job",
    maskedKey: "sk_live_...9be2",
    environment: "production",
    scopes: ["usage:read", "audit:read"],
    createdAt: daysAgo(96),
    lastUsedAt: hoursAgo(19),
    status: "active",
    requestsLast24h: 1_190,
  },
  {
    id: "key_4",
    name: "Local notebook",
    maskedKey: "sk_dev_...04c8",
    environment: "development",
    scopes: ["gateway:read"],
    createdAt: daysAgo(320),
    lastUsedAt: daysAgo(12),
    expiresAt: daysAhead(38),
    status: "active",
    requestsLast24h: 0,
  },
  {
    id: "key_5",
    name: "Legacy billing sync",
    maskedKey: "sk_live_...22fa",
    environment: "production",
    scopes: ["billing:read"],
    createdAt: daysAgo(400),
    lastUsedAt: daysAgo(34),
    status: "expired",
    requestsLast24h: 0,
  },
  {
    id: "key_6",
    name: "Contractor sandbox",
    maskedKey: "sk_test_...a310",
    environment: "staging",
    scopes: ["gateway:read", "gateway:write"],
    createdAt: daysAgo(58),
    lastUsedAt: daysAgo(16),
    status: "revoked",
    requestsLast24h: 0,
  },
];

export const codeSnippets: CodeSnippet[] = [
  {
    id: "snp_1",
    label: "Send a chat completion",
    language: "bash",
    code: `curl https://api.sentinel.dev/v1/chat/completions \\
  -H "Authorization: Bearer $SENTINEL_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "claude-sonnet-4-5",
    "messages": [{ "role": "user", "content": "Summarize this incident." }]
  }'`,
  },
  {
    id: "snp_2",
    label: "TypeScript SDK",
    language: "typescript",
    code: `import Sentinel from "@sentinel/sdk";

const sentinel = new Sentinel({ apiKey: process.env.SENTINEL_API_KEY });

const response = await sentinel.chat.completions.create({
  model: "claude-sonnet-4-5",
  messages: [{ role: "user", content: "Summarize this incident." }],
});

console.log(response.choices[0].message.content);`,
  },
  {
    id: "snp_3",
    label: "Python SDK",
    language: "python",
    code: `import os
from sentinel import Sentinel

client = Sentinel(api_key=os.environ["SENTINEL_API_KEY"])

response = client.chat.completions.create(
    model="claude-sonnet-4-5",
    messages=[{"role": "user", "content": "Summarize this incident."}],
)

print(response.choices[0].message.content)`,
  },
];

export const sdkIntegrations: SdkIntegration[] = [
  {
    id: "sdk_1",
    name: "@sentinel/sdk",
    language: "TypeScript",
    installed: true,
    latestVersion: "4.2.1",
    docsUrl: "https://sentinel.dev/docs/sdk/typescript",
  },
  {
    id: "sdk_2",
    name: "sentinel-python",
    language: "Python",
    installed: true,
    latestVersion: "3.8.0",
    docsUrl: "https://sentinel.dev/docs/sdk/python",
  },
  {
    id: "sdk_3",
    name: "sentinel-go",
    language: "Go",
    installed: false,
    latestVersion: "1.4.3",
    docsUrl: "https://sentinel.dev/docs/sdk/go",
  },
  {
    id: "sdk_4",
    name: "sentinel-rails",
    language: "Ruby",
    installed: false,
    latestVersion: "0.9.2",
    docsUrl: "https://sentinel.dev/docs/sdk/ruby",
  },
];
