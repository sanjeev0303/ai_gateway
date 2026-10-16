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

export default async function CredentialsPage() {
  const { rows, total, active, neverUsed, scopeMix } = await loadCredentials();

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Credentials"
        description="Control-plane API keys for this organization. Key hashes are never returned, so only the prefix shown here leaves the server."
        actions={<IssueCredentialDialog />}
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Credentials"
            value={formatNumber(total)}
            hint={`${formatNumber(active)} active`}
          />
          <StatCard
            label="Never used"
            value={formatNumber(neverUsed)}
            tone={neverUsed > 0 ? "warning" : "success"}
            hint="No last-used timestamp yet"
          />
          <StatCard label="Scopes" value={formatNumber(scopeMix.length)} hint="Distinct scopes" />
          <StatCard
            label="Revoked"
            value={formatNumber(total - active)}
            tone={total - active > 0 ? "neutral" : "success"}
            hint="No longer usable"
          />
        </section>

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

        <Card>
          <CardHeader>
            <CardTitle>All credentials</CardTitle>
            <CardDescription>
              Only a short prefix is stored and returned, so a leak of this table
              is not a leak of your keys.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={credentialColumns}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No control-plane credentials."
              ActionsComponent={CredentialRowActionsWrapper}
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
