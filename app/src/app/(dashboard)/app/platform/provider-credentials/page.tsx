import type { Metadata } from "next";

import { DonutChart } from "@/components/charts";
import { DataTable, PageShell, StatCard, type Column } from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatDate, formatNumber } from "@/lib/format";
import { toneFor } from "@/lib/org-tones";
import { getApiClient } from "@/lib/session";
import { ProviderCredentialsClient, type CredentialRow } from "./client";

export const metadata: Metadata = {
  title: "Provider credentials · Sentinel",
  description: "API credentials for upstream providers, with fingerprint and status.",
};

const CREDENTIAL_TONE: Record<string, "success" | "warning" | "danger" | "neutral"> = {
  active: "success",
  revoked: "neutral",
  expired: "danger",
};

const display = (kind: string) =>
  kind
    .split("_")
    .map((part) => (part ? part[0].toUpperCase() + part.slice(1) : part))
    .join(" ");

const COLUMNS: readonly Column<CredentialRow>[] = [
  { key: "name", header: "Credential", primary: "name", secondary: "accountName", sort: "name" },
  { key: "kind", header: "Kind", primary: "kind", sort: "kind" },
  { key: "fingerprint", header: "Fingerprint", primary: "fingerprint", sort: "fingerprint" },
  { key: "tail", header: "Tail", primary: "tail", sort: "tail" },
  { key: "status", header: "Status", primary: "status", tone: "statusTone", sort: "status" },
  { key: "created", header: "Created", primary: "created", sort: "created", align: "right" },
];

async function loadProviderCredentials() {
  const { providers } = await getApiClient();
  const [accounts, credentials] = await Promise.all([
    providers.listAccounts(),
    providers.listCredentials(),
  ]);

  const accountMap = new Map(accounts.items.map((a) => [a.id, a.name]));

  const rows: CredentialRow[] = [...credentials.items]
    .sort((a, b) => a.name.localeCompare(b.name))
    .map((credential) => ({
      id: credential.id,
      name: credential.name,
      accountName: accountMap.get(credential.provider_account_id) ?? credential.provider_account_id,
      kind: display(credential.kind),
      fingerprint: credential.fingerprint,
      tail: credential.tail,
      status: credential.status,
      statusTone: toneFor(CREDENTIAL_TONE, credential.status) as "success" | "warning" | "danger" | "neutral",
      created: formatDate(credential.created_at),
    }));

  const kindCounts = new Map<string, number>();
  for (const row of rows) {
    kindCounts.set(row.kind, (kindCounts.get(row.kind) ?? 0) + 1);
  }

  return {
    rows,
    total: rows.length,
    active: rows.filter((row) => row.status === "active").length,
    kinds: kindCounts.size,
    kindMix: [...kindCounts.entries()]
      .map(([label, value]) => ({ label, value }))
      .sort((a, b) => b.value - a.value),
  };
}

export default async function ProviderCredentialsPage() {
  const { rows, total, active, kinds, kindMix } = await loadProviderCredentials();

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Provider credentials"
        description="Credentials for upstream providers. Only a fingerprint and tail are stored."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Credentials"
            value={formatNumber(total)}
            hint={`${formatNumber(active)} active`}
          />
          <StatCard
            label="Revoked"
            value={formatNumber(total - active)}
            tone={total - active > 0 ? "neutral" : "success"}
            hint="No longer usable"
          />
          <StatCard
            label="Provider kinds"
            value={formatNumber(kinds)}
            hint="Distinct upstream providers"
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Credentials by provider</CardTitle>
            <CardDescription>Which upstream each credential targets.</CardDescription>
          </CardHeader>
          <CardContent>
            <DonutChart
              data={kindMix}
              centerLabel={formatNumber(total)}
              formatValue={(value) => `${formatNumber(value)} credentials`}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>All credentials</CardTitle>
            <CardDescription>
              Only a short fingerprint and tail are stored. The full secret is never returned.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <ProviderCredentialsClient initialRows={rows} />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}