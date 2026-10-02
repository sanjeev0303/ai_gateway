import type { Metadata } from "next";

import { CodeBlock } from "@/components/developer/code-block";
import {
  DataTable,
  PageShell,
  StatCard,
  type Column,
} from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { developerApiKeys } from "@/lib/mock-data";

export const metadata: Metadata = {
  title: "CLI · Sentinel",
  description: "Command reference for the sentinel CLI.",
};

interface CommandRow {
  id: string;
  command: string;
  group: string;
  summary: string;
  example: string;
}

const COLUMNS: readonly Column<CommandRow>[] = [
  { key: "command", header: "Command", primary: "command", secondary: "example", sort: "command" },
  { key: "group", header: "Group", primary: "group", sort: "group" },
  { key: "summary", header: "What it does", primary: "summary", sort: "summary" },
];

const COMMANDS: readonly CommandRow[] = [
  {
    id: "c1",
    command: "sentinel login",
    group: "auth",
    summary: "Authenticate the CLI and store a token in the local keyring",
    example: "sentinel login --org acme",
  },
  {
    id: "c2",
    command: "sentinel keys create",
    group: "keys",
    summary: "Issue a scoped API key for a given environment",
    example: 'sentinel keys create --name ci --environment staging',
  },
  {
    id: "c3",
    command: "sentinel keys list",
    group: "keys",
    summary: "List keys with their scopes, last use, and 24h request counts",
    example: "sentinel keys list --environment production",
  },
  {
    id: "c4",
    command: "sentinel keys revoke",
    group: "keys",
    summary: "Revoke a key immediately; in-flight requests are cut off",
    example: "sentinel keys revoke key_6",
  },
  {
    id: "c5",
    command: "sentinel models list",
    group: "registry",
    summary: "List registered models with entitlements and current status",
    example: "sentinel models list --available",
  },
  {
    id: "c6",
    command: "sentinel policies test",
    group: "policies",
    summary: "Dry-run a policy against a payload without sending it upstream",
    example: "sentinel policies test pol_pii --file request.json",
  },
  {
    id: "c7",
    command: "sentinel usage export",
    group: "usage",
    summary: "Export usage or cost data as CSV or JSON for a date range",
    example: "sentinel usage export --metric cost --since 2026-08-01 --format csv",
  },
  {
    id: "c8",
    command: "sentinel traces tail",
    group: "debug",
    summary: "Stream live gateway traffic with policy decisions attached",
    example: "sentinel traces tail --project customer-portal",
  },
  {
    id: "c9",
    command: "sentinel doctor",
    group: "debug",
    summary: "Check connectivity, credentials, and clock skew against the gateway",
    example: "sentinel doctor --verbose",
  },
];

const INSTALL_SNIPPET = `# macOS
brew install sentinel/tap/sentinel

# Linux
curl -fsSL https://sentinel.dev/install.sh | sh

# npm
npm install -g @sentinel/cli`;

export default function DeveloperCliPage() {
  const active = developerApiKeys.filter((key) => key.status === "active").length;

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="CLI"
        description="The same operations the dashboard performs, scriptable and safe to run in CI."
        eyebrow={
          <p className="text-xs text-muted-foreground">
            Latest <code className="font-mono">1.9.0</code> ·{" "}
            <code className="font-mono">sentinel --version</code>
          </p>
        }
      >
        <section className="grid gap-4 sm:grid-cols-3">
          <StatCard label="Commands" value={String(COMMANDS.length)} hint="Across 6 groups" />
          <StatCard
            label="Groups"
            value={String(new Set(COMMANDS.map((command) => command.group)).size)}
            hint="auth, keys, registry, policies, usage, debug"
          />
          <StatCard
            label="Active keys"
            value={String(active)}
            tone="success"
            hint="Visible to the authenticated identity"
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Install</CardTitle>
            <CardDescription>
              The CLI reads the same scoped credentials the SDKs use, so anything you can
              do in the dashboard you can do here.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <CodeBlock label="Install" language="bash" code={INSTALL_SNIPPET} />
            <CodeBlock
              label="Authenticate"
              language="bash"
              code={`sentinel login --org acme
sentinel doctor`}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Command reference</CardTitle>
            <CardDescription>
              Every command accepts <code className="font-mono">--output json</code> for
              scripting and <code className="font-mono">--help</code> for full flags.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={COMMANDS}
              rowKey="id"
              emptyMessage="No commands available."
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Using it in CI</CardTitle>
            <CardDescription>
              Issue a key once, store it as a secret, and scope it as narrowly as the job
              allows.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <CodeBlock
              label="GitHub Actions"
              language="bash"
              code={`- name: Enforce budgets
  env:
    SENTINEL_API_KEY: \${{ secrets.SENTINEL_API_KEY }}
  run: |
    sentinel policies test pol_budget --file request.json \\
      --fail-on violation`}
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
