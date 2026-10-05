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

export const metadata: Metadata = {
  title: "CLI · Sentinel",
  description: "Command reference for the aigw CLI.",
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

// Subcommands and flags below are taken from packages/cli/cmd. There is no
// --output flag, no --version, and no models/policies/usage/traces command.
const COMMANDS: readonly CommandRow[] = [
  {
    id: "c1",
    command: "aigw auth login",
    group: "auth",
    summary: "Authenticate against a gateway and store the credential locally",
    example: "aigw auth login --gateway-url http://localhost:8084",
  },
  {
    id: "c2",
    command: "aigw auth status",
    group: "auth",
    summary: "Show the current authentication status",
    example: "aigw auth status",
  },
  {
    id: "c3",
    command: "aigw auth logout",
    group: "auth",
    summary: "Remove the stored credential",
    example: "aigw auth logout",
  },
  {
    id: "c4",
    command: "aigw credential create",
    group: "credential",
    summary: "Create a credential; raw key is shown once",
    example: 'aigw credential create --name ci --subject-type service_account --subject-id svc_1 --scopes gateway --expires-in 30d',
  },
  {
    id: "c5",
    command: "aigw credential list",
    group: "credential",
    summary: "List credentials",
    example: "aigw credential list",
  },
  {
    id: "c6",
    command: "aigw credential get <credential-id>",
    group: "credential",
    summary: "Show one credential",
    example: "aigw credential get cred_1",
  },
  {
    id: "c7",
    command: "aigw credential rotate <credential-id>",
    group: "credential",
    summary: "Issue a replacement key for an existing credential",
    example: "aigw credential rotate cred_1 --scopes gateway,control",
  },
  {
    id: "c8",
    command: "aigw credential revoke <credential-id>",
    group: "credential",
    summary: "Revoke a credential; pass --force to skip the prompt",
    example: "aigw credential revoke cred_1 --force",
  },
  {
    id: "c9",
    command: "aigw credential expire <credential-id>",
    group: "credential",
    summary: "Expire a credential immediately",
    example: "aigw credential expire cred_1",
  },
  {
    id: "c10",
    command: "aigw credential restore <credential-id>",
    group: "credential",
    summary: "Clear the expiry on a credential",
    example: "aigw credential restore cred_1",
  },
  {
    id: "c11",
    command: "aigw config install",
    group: "config",
    summary: "Install a local gateway profile",
    example: 'aigw config install --gateway-url http://localhost:8084 --credential "$AIGW_API_KEY" --name local',
  },
  {
    id: "c12",
    command: "aigw config show",
    group: "config",
    summary: "Show the current configuration",
    example: "aigw config show",
  },
  {
    id: "c13",
    command: "aigw config uninstall",
    group: "config",
    summary: "Remove the local configuration",
    example: "aigw config uninstall",
  },
];

const INSTALL_SNIPPET = `cd packages/cli
go build -o aigw .
./aigw auth login --gateway-url http://localhost:8084`;

export default function DeveloperCliPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="CLI"
        description="The aigw CLI manages credentials and local gateway configuration. It covers the control plane only — it has no commands for model traffic."
        eyebrow={
          <p className="text-xs text-muted-foreground">
            Built from source · <code className="font-mono">aigw</code>
          </p>
        }
      >
        <section className="grid gap-4 sm:grid-cols-2">
          <StatCard label="Commands" value={String(COMMANDS.length)} hint="Across 3 groups" />
          <StatCard
            label="Scope"
            value="Credentials"
            tone="info"
            hint="auth, credential, config"
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Install</CardTitle>
            <CardDescription>
              The CLI is a private Go module in this monorepo. It is not published to a
              package registry, so build it from the checkout.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <CodeBlock label="Build and log in" language="bash" code={INSTALL_SNIPPET} />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Command reference</CardTitle>
            <CardDescription>
              Every command supports <code className="font-mono">--help</code> for its
              full flag list.
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
              Create a short-lived, narrow credential and keep it as a secret. The raw key
              is printed once, so capture it at creation.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <CodeBlock
              label="GitHub Actions"
              language="bash"
              code={`- name: Mint a short-lived credential
  run: |
    go run . credential create \\
      --name ci \\
      --subject-type service_account \\
      --subject-id "\${{ secrets.SERVICE_ACCOUNT_ID }}" \\
      --scopes gateway \\
      --expires-in 1h`}
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}