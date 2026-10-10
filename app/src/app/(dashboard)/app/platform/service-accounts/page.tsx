import type { Metadata } from "next";

import { DataTable, PageShell, StatCard, type Column } from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatDate, formatNumber, microsToUsd, formatUsd } from "@/lib/format";
import { toneFor } from "@/lib/org-tones";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@/types";
import { CreateServiceAccountButton, ServiceAccountActions } from "./client";

export const metadata: Metadata = {
  title: "Service accounts · Sentinel",
  description: "Machine identities used by workloads instead of human sessions.",
};

const SERVICE_ACCOUNT_TONE: Record<string, StatusTone> = {
  active: "success",
  revoked: "neutral",
};

interface ServiceAccountRow {
  id: string;
  name: string;
  description: string;
  status: string;
  tone: StatusTone;
  requests: number;
  spend: number;
  created: string;
  detailUrl: string;
}

const COLUMNS: readonly Column<ServiceAccountRow>[] = [
  { key: "name", header: "Service account", primary: "name", secondary: "description", sort: "name", href: "detailUrl" },
  { key: "status", header: "Status", primary: "status", tone: "tone", sort: "status" },
  { key: "requests", header: "Requests", primary: "requests", format: "compact", align: "right", sort: "requests" },
  { key: "spend", header: "Spend (30d)", primary: "spend", format: "usd", align: "right", sort: "spend" },
  { key: "created", header: "Created", primary: "created", sort: "created" },
];

async function loadServiceAccounts() {
  const { serviceAccounts, usage } = await getApiClient();
  const [list, accountUsage] = await Promise.all([serviceAccounts.list(), usage.byServiceAccount()]);

  const spendByAccount = new Map(accountUsage.items.map((row) => [row.ServiceAccountID, row]));

  const rows = [...list.items]
    .sort((a, b) => a.name.localeCompare(b.name))
    .map((account) => {
      const attributed = spendByAccount.get(account.id);
      return {
        id: account.id,
        name: account.name,
        description: account.description || "No description",
        status: account.status,
        tone: toneFor(SERVICE_ACCOUNT_TONE, account.status),
        requests: attributed?.Requests ?? 0,
        spend: microsToUsd(attributed?.CostMicros ?? 0),
        created: formatDate(account.created_at),
        detailUrl: `/app/platform/service-accounts/${account.id}`,
      };
    });

  const totalRequests = rows.reduce((sum, row) => sum + row.requests, 0);

  return {
    rows,
    total: rows.length,
    active: rows.filter((row) => row.status === "active").length,
    described: list.items.filter((item) => Boolean(item.description)).length,
    totalRequests,
  };
}

export default async function ServiceAccountsPage() {
  const { rows, total, active, described, totalRequests } = await loadServiceAccounts();

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Service accounts"
        description="Non-human identities that call the gateway on behalf of workloads and pipelines."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Service accounts"
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
            label="Requests"
            value={formatNumber(totalRequests)}
            hint="Trailing 30 days"
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Accounts</CardTitle>
            <CardDescription>
              Service accounts hold no secret of their own. Keys are issued
              separately and can be scoped to a team or project.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No service accounts yet."
              toolbar={
                <div className="flex justify-end mb-4">
                  <CreateServiceAccountButton />
                </div>
              }
              actions={(row) => <ServiceAccountActions account={row} />}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Usage and rotation</CardTitle>
            <CardDescription>Accounts accrue telemetry</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2 text-sm text-muted-foreground">
            <p>
              Service account telemetry tracks the total API usage of 
              identities over a 30-day trailing period.
            </p>
            <p>
              Expiry and owning team or project are recorded on the gateway
              credential itself, listed under Platform.
            </p>
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
