import type { Metadata } from "next";
import { ExternalLink } from "lucide-react";

import { CodeBlock } from "@/components/developer/code-block";
import {
  DataTable,
  PageShell,
  StatCard,
  StatusBadge,
  type Column,
} from "@/components/dashboard";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { sdkIntegrations } from "@/lib/mock-data";

export const metadata: Metadata = {
  title: "SDKs · Sentinel",
  description: "Official client libraries and their install commands.",
};

const INSTALL: Record<string, string> = {
  TypeScript: "npm install @sentinel/sdk",
  Python: "pip install sentinel",
  Go: "go get github.com/sentinel-dev/sentinel-go",
  Ruby: "gem install sentinel",
};

interface SdkRow {
  id: string;
  name: string;
  language: string;
  installed: boolean;
  installedLabel: string;
  installedTone: "success" | "neutral";
  latestVersion: string;
  docsUrl: string;
  docsLabel: string;
}

const COLUMNS: readonly Column<SdkRow>[] = [
  { key: "name", header: "Package", primary: "name", secondary: "language", sort: "name" },
  { key: "installed", header: "Installed", primary: "installedLabel", tone: "installedTone" },
  { key: "latestVersion", header: "Latest", primary: "latestVersion", align: "right", sort: "latestVersion" },
  { key: "docsUrl", header: "Docs", primary: "docsLabel", href: "docsUrl", align: "right" },
];

export default function SdksPage() {
  const rows: SdkRow[] = sdkIntegrations.map((sdk) => ({
    id: sdk.id,
    name: sdk.name,
    language: sdk.language,
    installed: sdk.installed,
    installedLabel: sdk.installed ? "installed" : "not installed",
    installedTone: sdk.installed ? "success" : "neutral",
    latestVersion: sdk.latestVersion,
    docsUrl: sdk.docsUrl,
    docsLabel: "Open reference",
  }));

  const installed = sdkIntegrations.filter((sdk) => sdk.installed).length;

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="SDKs"
        description="Official clients for the gateway. Each wraps the same REST surface with local types."
        actions={
          <Button size="sm" variant="outline">
            <ExternalLink aria-hidden /> All SDKs
          </Button>
        }
      >
        <section className="grid gap-4 sm:grid-cols-3">
          <StatCard
            label="Libraries"
            value={String(sdkIntegrations.length)}
            hint="Maintained by Sentinel"
          />
          <StatCard
            label="Installed"
            value={`${installed} of ${sdkIntegrations.length}`}
            tone="success"
            hint="Detected from your workspaces"
          />
          <StatCard
            label="Surface"
            value="100%"
            tone="info"
            hint="Covered by the gateway contract"
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Install</CardTitle>
            <CardDescription>
              Libraries are versioned together with the gateway, so a minor release never
              moves the wire format underneath you.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {sdkIntegrations.map((sdk) => (
              <CodeBlock
                key={sdk.id}
                label={`${sdk.name} · ${sdk.latestVersion}`}
                language="bash"
                code={INSTALL[sdk.language] ?? `npm install ${sdk.name}`}
              />
            ))}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Available libraries</CardTitle>
            <CardDescription>
              Go and Ruby are community-maintained and currently behind the release train.
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
                    <StatusBadge key={sdk.id} tone={sdk.installed ? "success" : "neutral"}>
                      {sdk.language}
                    </StatusBadge>
                  ))}
                </div>
              }
              emptyMessage="No SDKs available."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
