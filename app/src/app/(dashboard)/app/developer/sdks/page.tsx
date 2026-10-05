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
  title: "SDKs · Sentinel",
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
        description="No client library is published to a public registry yet. The gateway is plain HTTP, and these two TypeScript packages exist inside this monorepo only."
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
  -d '{"model":"claude-sonnet-4-5","max_tokens":1024,"messages":[]}'`}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Available packages</CardTitle>
            <CardDescription>
              Both are private workspace packages versioned at 0.0.0. The API client
              talks to the control plane, not the gateway.
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