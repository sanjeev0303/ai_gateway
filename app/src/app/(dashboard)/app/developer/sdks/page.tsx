import type { Metadata } from "next";

import { CodeBlock } from "@/components/developer/code-block";
import {
  DataTable,
  PageShell,
  StatCard,
  StatusBadge,
  type Column,
} from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { sdkIntegrations } from "@/lib/developer-content";

export const metadata: Metadata = {
  title: "SDKs · Tollseal",
  description: "Client libraries available in this repository.",
};

interface SdkRow {
  id: string;
  name: string;
  language: string;
  latestVersion: string;
  docsUrl: string;
}

const COLUMNS: readonly Column<SdkRow>[] = [
  { key: "name", header: "Package", primary: "name", secondary: "language", sort: "name" },
  { key: "latestVersion", header: "Version", primary: "latestVersion", align: "right", sort: "latestVersion" },
  { key: "docsUrl", header: "Source", primary: "docsUrl" },
];

export default function SdksPage() {
  const rows: SdkRow[] = sdkIntegrations.map((sdk) => ({
    id: sdk.id,
    name: sdk.name,
    language: sdk.language,
    latestVersion: sdk.latestVersion,
    docsUrl: sdk.docsUrl,
  }));

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="SDKs"
        description="No client library is published to a public registry yet. The gateway is plain HTTP, and the TypeScript client in this monorepo wraps it: models, chat completions and messages, with streaming."
      >
        <section className="grid gap-4 sm:grid-cols-2">
          <StatCard label="Published libraries" value="0" hint="Nothing on npm, PyPI, or Go proxies" />
          <StatCard
            label="Workspace packages"
            value={String(sdkIntegrations.length)}
            hint="Consumed by the dashboard, not installed by users"
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Call the gateway directly</CardTitle>
            <CardDescription>
              Requests go to the gateway&apos;s Anthropic-shaped messages endpoint, so you do not need a
              wrapper library. See Quick start for a complete request.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <CodeBlock
              label="POST /v1/messages"
              language="bash"
              code={`curl http://localhost:8084/v1/messages \\
  -H "x-api-key: $AIGW_API_KEY" \\
  -H "content-type: application/json" \\
  -d '{"model":"claude-sonnet-4-5","max_tokens":1024,"messages":[{"role":"user","content":"Hello"}]}'`}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>TypeScript gateway client</CardTitle>
            <CardDescription>
              createGatewayClient in @ai-governance/api-client calls the gateway with your gateway key:
              listModels, chatCompletion and sendMessage, each with a streaming variant, plus retries
              and an optional project scope.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <CodeBlock
              label="packages/api-client/src/gateway.ts"
              language="typescript"
              code={`import { createGatewayClient } from "@ai-governance/api-client";

const gateway = createGatewayClient({
  gatewayUrl: process.env.AIGW_GATEWAY_URL ?? "http://localhost:8084",
  apiKey: process.env.AIGW_API_KEY!,
  projectId: process.env.AIGW_PROJECT_ID, // optional: scope requests to one project
});

const models = await gateway.listModels();
const { text, meta } = await gateway.chatCompletion({
  model: "gpt-4o-mini",
  messages: [{ role: "user", content: "Summarise this ticket." }],
});
console.log(text, meta.totalTokens, meta.durationMs);

// Stream the reply as it arrives
await gateway.streamMessage(
  { model: "claude-sonnet-4-5", max_tokens: 1024, messages: [{ role: "user", content: "Hello" }] },
  (chunk) => process.stdout.write(chunk),
);`}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Available packages</CardTitle>
            <CardDescription>
              Both are private workspace packages versioned at 0.0.0. The API client holds the gateway
              client above and the control-plane client the console uses.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={rows}
              rowKey="id"
              toolbar={
                <div className="flex flex-wrap gap-2">
                  {sdkIntegrations.map((sdk) => (
                    <StatusBadge key={sdk.id} tone="neutral">
                      {sdk.language}
                    </StatusBadge>
                  ))}
                </div>
              }
              emptyMessage="No packages available."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}