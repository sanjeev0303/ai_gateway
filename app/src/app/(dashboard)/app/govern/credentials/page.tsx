import type { Metadata } from "next";

import { DonutChart } from "@/components/charts";
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
import { formatDate, formatNumber, formatRelative } from "@/lib/format";
import { toneFor } from "@/lib/org-tones";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@/types";
import { IssueCredentialDialog, CredentialRowActionsWrapper } from "./client";

export const metadata: Metadata = {
  title: "Credentials · Sentinel",
  description: "Control-plane API keys, their scopes, and last use.",
};

const CREDENTIAL_TONE: Record<string, StatusTone> = {
  active: "success",
  revoked: "neutral",
  expired: "danger",
};

interface CredentialRow {
  id: string;
  name: string;
  prefix: string;
  scope: string;
  status: string;
  statusTone: StatusTone;
  serviceAccount: string;
  lastUsed: string;
  created: string;
}

const credentialColumns: readonly Column<CredentialRow>[] = [
  { key: "name", header: "Credential", primary: "name", secondary: "prefix", sort: "name" },
  { key: "scope", header: "Scope", primary: "scope", sort: "scope" },
  { key: "status", header: "Status", primary: "status", tone: "statusTone", sort: "status" },
  {
    key: "serviceAccount",
    header: "Service account",
    primary: "serviceAccount",
    sort: "serviceAccount",
  },
  { key: "lastUsed", header: "Last used", primary: "lastUsed", sort: "lastUsed" },
  { key: "created", header: "Created", primary: "created", sort: "created" },
];

async function loadCredentials() {
  const { credentials } = await getApiClient();
  const list = await credentials.list();
  const now = new Date();

  const rows: CredentialRow[] = [...list.items]
    .sort((a, b) => a.name.localeCompare(b.name))
    .map((credential) => ({
      id: credential.id,
      name: credential.name,
      prefix: credential.key_prefix,
      scope: credential.scope || "unspecified",
      status: credential.status,
      statusTone: toneFor(CREDENTIAL_TONE, credential.status),
      serviceAccount: credential.service_account_id ?? "—",
      lastUsed: credential.last_used_at ? formatRelative(credential.last_used_at, now) : "Never",
      created: formatDate(credential.created_at),
    }));

  const scopeCounts = new Map<string, number>();
  for (const row of rows) {
    scopeCounts.set(row.scope, (scopeCounts.get(row.scope) ?? 0) + 1);
  }

  return {
    rows,
    total: rows.length,
    active: rows.filter((row) => row.status === "active").length,
    neverUsed: list.items.filter((item) => !item.last_used_at).length,
    scopeMix: [...scopeCounts.entries()]
      .map(([label, value]) => ({ label, value }))
      .sort((a, b) => b.value - a.value),
  };
}

import {
  Key,
  ShieldCheck,
  Lock,
  Terminal,
  Zap,
  Clock,
  ArrowRight,
  Sparkles,
  Layers,
  FileCode,
} from "lucide-react";
import { Button } from "@/components/ui/button";

export default async function CredentialsPage() {
  const { rows, total, active, neverUsed, scopeMix } = await loadCredentials();

  return (
    <div className="p-4 sm:p-6 space-y-6">
      <PageShell
        title="Credentials"
        description="Control-plane API keys for administrative automation, policy enforcement, and infrastructure CI/CD. Hashes are securely stored, and secrets are shown only once at creation."
        actions={<IssueCredentialDialog />}
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Credentials"
            value={formatNumber(total)}
            hint={`${formatNumber(active)} active`}
            icon={Key}
          />
          <StatCard
            label="Never used"
            value={formatNumber(neverUsed)}
            tone={neverUsed > 0 ? "warning" : "neutral"}
            hint={total > 0 ? "No last-used timestamp yet" : "Zero unused keys"}
            icon={Clock}
          />
          <StatCard
            label="Scopes"
            value={formatNumber(scopeMix.length)}
            hint={scopeMix.length > 0 ? "Distinct scopes" : "Standard defaults ready"}
            icon={Layers}
          />
          <StatCard
            label="Revoked"
            value={formatNumber(total - active)}
            tone={total - active > 0 ? "neutral" : "success"}
            hint={total - active > 0 ? "No longer usable" : "Zero revoked keys"}
            icon={ShieldCheck}
          />
        </section>

        {total === 0 ? (
          <section className="space-y-6">
            {/* Visual Hero Card for First-Time Setup */}
            <Card className="border-border/70 shadow-sm overflow-hidden bg-linear-to-br from-card via-card to-muted/20">
              <CardContent className="p-6 sm:p-8">
                <div className="flex flex-col lg:flex-row items-start lg:items-center justify-between gap-6">
                  <div className="space-y-3 max-w-2xl">
                    <div className="inline-flex items-center gap-2 px-2.5 py-1 rounded-full text-xs font-semibold bg-primary/10 text-primary border border-primary/20">
                      <Sparkles className="size-3.5" />
                      <span>Control-Plane Administration</span>
                    </div>
                    <h3 className="text-xl sm:text-2xl font-bold tracking-tight text-foreground">
                      No control-plane credentials created yet
                    </h3>
                    <p className="text-sm text-muted-foreground leading-relaxed">
                      Control-plane API keys let your DevOps pipelines, Terraform scripts, or internal dashboards manage organizations, configure provider routing, issue policies, and query system audits programmatically without web UI sessions.
                    </p>
                    <div className="flex flex-wrap items-center gap-4 pt-2">
                      <IssueCredentialDialog
                        triggerLabel="Create First Control Key"
                        triggerClassName="font-medium shadow-sm"
                      />
                      <a
                        href="/app/platform/gateway-credentials"
                        className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground font-medium transition-colors"
                      >
                        Looking for LLM runtime keys? View Gateway Credentials
                        <ArrowRight className="size-3.5" />
                      </a>
                    </div>
                  </div>

                  <div className="w-full lg:w-80 rounded-xl border bg-muted/40 p-4 space-y-3 text-xs font-mono">
                    <div className="flex items-center justify-between text-muted-foreground border-b pb-2">
                      <span className="flex items-center gap-1.5 font-sans font-medium text-foreground">
                        <Terminal className="size-3.5 text-primary" />
                        Quick cURL Example
                      </span>
                      <span className="text-[10px] uppercase font-bold text-muted-foreground">GET /v1/models</span>
                    </div>
                    <pre className="text-muted-foreground leading-relaxed whitespace-pre-wrap break-all text-[11px] select-all bg-background/50 p-2.5 rounded-lg border">
                      curl https://api.sentinel.io/v1/models \<br />
                      &nbsp;&nbsp;-H &quot;Authorization: Bearer aigw_key_...&quot;
                    </pre>
                    <p className="text-[11px] font-sans text-muted-foreground">
                      Direct REST endpoint authentication for autonomous workloads.
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Scope Capabilities & Architecture Guidance */}
            <div className="grid gap-4 md:grid-cols-3">
              <Card className="border-border/60 hover:border-border transition-colors">
                <CardHeader className="pb-3">
                  <div className="size-9 rounded-lg bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 flex items-center justify-center mb-1">
                    <ShieldCheck className="size-5" />
                  </div>
                  <CardTitle className="text-base">Control Plane Admin</CardTitle>
                  <CardDescription className="text-xs">
                    Full administrative privileges
                  </CardDescription>
                </CardHeader>
                <CardContent className="text-xs text-muted-foreground space-y-2">
                  <p>
                    Manage provider accounts, rotate upstream keys, update spending budgets, and configure tenant routing policies.
                  </p>
                  <div className="pt-2 font-mono text-[11px] text-foreground">
                    scope: <code>control_plane</code>
                  </div>
                </CardContent>
              </Card>

              <Card className="border-border/60 hover:border-border transition-colors">
                <CardHeader className="pb-3">
                  <div className="size-9 rounded-lg bg-blue-500/10 text-blue-600 dark:text-blue-400 flex items-center justify-center mb-1">
                    <FileCode className="size-5" />
                  </div>
                  <CardTitle className="text-base">Policy Manager</CardTitle>
                  <CardDescription className="text-xs">
                    Governance and guardrails
                  </CardDescription>
                </CardHeader>
                <CardContent className="text-xs text-muted-foreground space-y-2">
                  <p>
                    Update guardrail rules, DLP content filters, model blocklists, and team routing targets without modifying organization credentials.
                  </p>
                  <div className="pt-2 font-mono text-[11px] text-foreground">
                    scope: <code>policy_manager</code>
                  </div>
                </CardContent>
              </Card>

              <Card className="border-border/60 hover:border-border transition-colors">
                <CardHeader className="pb-3">
                  <div className="size-9 rounded-lg bg-amber-500/10 text-amber-600 dark:text-amber-400 flex items-center justify-center mb-1">
                    <Lock className="size-5" />
                  </div>
                  <CardTitle className="text-base">Read Only Auditor</CardTitle>
                  <CardDescription className="text-xs">
                    Telemetry, usage, & audit logs
                  </CardDescription>
                </CardHeader>
                <CardContent className="text-xs text-muted-foreground space-y-2">
                  <p>
                    Read organization audit events, inspect model consumption, and pull itemized billing telemetry for external SIEM and data pipelines.
                  </p>
                  <div className="pt-2 font-mono text-[11px] text-foreground">
                    scope: <code>read_only</code>
                  </div>
                </CardContent>
              </Card>
            </div>
          </section>
        ) : (
          <section className="grid gap-4 lg:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Credentials by scope</CardTitle>
                <CardDescription>How the keys on file are scoped.</CardDescription>
              </CardHeader>
              <CardContent>
                <DonutChart
                  data={scopeMix}
                  centerLabel={formatNumber(total)}
                  formatValue={(value) => `${formatNumber(value)} keys`}
                />
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Expiry and blast radius</CardTitle>
                <CardDescription>Control-plane administration keys</CardDescription>
              </CardHeader>
              <CardContent className="space-y-2 text-sm text-muted-foreground">
                <p>
                  Control-plane keys authenticate direct administrative automation and CI/CD operations.
                  Keys are issued with single-display secret generation and immediate revocation.
                </p>
                <p>
                  For employee and application runtime routing, see{" "}
                  <a href="/app/platform/gateway-credentials" className="text-primary hover:underline font-medium">
                    Gateway Credentials
                  </a>.
                </p>
              </CardContent>
            </Card>
          </section>
        )}

        <Card>
          <CardHeader className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div>
              <CardTitle>All credentials</CardTitle>
              <CardDescription>
                Only a short prefix is stored and returned, so a leak of this table is not a leak of your keys.
              </CardDescription>
            </div>
            {total > 0 && <IssueCredentialDialog triggerVariant="outline" triggerLabel="Issue Key" />}
          </CardHeader>
          <CardContent>
            <DataTable
              columns={credentialColumns}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No control-plane credentials issued yet."
              ActionsComponent={CredentialRowActionsWrapper}
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
