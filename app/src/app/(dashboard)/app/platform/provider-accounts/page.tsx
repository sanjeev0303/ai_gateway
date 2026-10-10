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
import type { StatusTone } from "@/types";
import { CreateProviderAccountButton } from "./client";

export const metadata: Metadata = {
  title: "Provider accounts · Sentinel",
  description: "Connected upstream providers and the base URL each one is reached at.",
};

const PROVIDER_TONE: Record<string, StatusTone> = {
  active: "success",
  disabled: "neutral",
};

const display = (kind: string) =>
  kind
    .split("_")
    .map((part) => (part ? part[0].toUpperCase() + part.slice(1) : part))
    .join(" ");

interface AccountRow {
  id: string;
  name: string;
  kind: string;
  baseUrl: string;
  status: string;
  tone: StatusTone;
  created: string;
  detailUrl: string;
}

const COLUMNS: readonly Column<AccountRow>[] = [
  { key: "name", header: "Account", primary: "name", secondary: "kind", sort: "name", href: "detailUrl" },
  { key: "baseUrl", header: "Base URL", primary: "baseUrl", sort: "baseUrl" },
  { key: "status", header: "Status", primary: "status", tone: "tone", sort: "status" },
  {
    key: "created",
    header: "Created",
    primary: "created",
    sort: "created",
    align: "right",
  },
];

async function loadProviderAccounts() {
  const { providers } = await getApiClient();
  const list = await providers.listAccounts();

  const rows: AccountRow[] = [...list.items]
    .sort((a, b) => a.name.localeCompare(b.name))
    .map((account) => ({
      id: account.id,
      name: account.name,
      kind: display(account.provider_kind),
      baseUrl: account.base_url ?? "Provider default",
      status: account.status,
      tone: toneFor(PROVIDER_TONE, account.status),
      created: formatDate(account.created_at),
      detailUrl: `/app/platform/provider-accounts/${account.id}`,
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

export default async function ProviderAccountsPage() {
  const { rows, total, active, kinds, kindMix } = await loadProviderAccounts();

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Provider accounts"
        description="Every upstream account behind one gateway key."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Connected accounts"
            value={formatNumber(total)}
            hint={`${formatNumber(active)} active`}
          />
          <StatCard
            label="Disabled"
            value={formatNumber(total - active)}
            tone={total - active > 0 ? "warning" : "success"}
            hint="Not used for routing"
          />
          <StatCard
            label="Provider kinds"
            value={formatNumber(kinds)}
            hint="Distinct upstream providers"
          />
          <StatCard
            label="Custom endpoints"
            value={formatNumber(rows.filter((row) => row.baseUrl !== "Provider default").length)}
            hint="Overriding the default base URL"
          />
        </section>

        <section className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Accounts by provider</CardTitle>
              <CardDescription>Which upstream each account targets.</CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart
                data={kindMix}
                centerLabel={formatNumber(total)}
                formatValue={(value) => `${formatNumber(value)} accounts`}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Reliability and pricing</CardTitle>
              <CardDescription>Not attributes of a provider account.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-2 text-sm text-muted-foreground">
              <p>
                A provider account records its kind, base URL, and status. It
                carries no uptime figure, no latency percentile, and no platform
                fee, so this page cannot rank accounts by speed or cost.
              </p>
              <p>
                Live probe results are on the provider health page, and the model
                catalogue is on the model registry page.
              </p>
            </CardContent>
          </Card>
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Accounts</CardTitle>
            <CardDescription>
              Status reflects the stored account record. It is not inferred from
              traffic.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No provider accounts connected."
              toolbar={
                <div className="flex justify-end mb-4">
                  <CreateProviderAccountButton />
                </div>
              }
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
