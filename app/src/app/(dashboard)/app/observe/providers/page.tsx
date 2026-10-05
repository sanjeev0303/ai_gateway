import type { Metadata } from "next";

import { BarChart, LineChart, chartColor } from "@/components/charts";
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
import {
  formatNumber,
  formatPercent,
  formatUsd,
  microsToUsd,
} from "@/lib/format";
import { toneFor } from "@/lib/org-tones";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Providers · Sentinel",
  description: "Upstream connections, probe health, and cost for every provider.",
};

const ACCOUNT_STATUS_TONE: Record<string, StatusTone> = {
  active: "success",
  disabled: "neutral",
};

// provider_health.status is a free-text column, so this map is a best effort.
const HEALTH_TONE: Record<string, StatusTone> = {
  healthy: "success",
  ok: "success",
  degraded: "warning",
  down: "danger",
  error: "danger",
};

interface ProviderRow {
  id: string;
  name: string;
  kind: string;
  status: string;
  statusTone: StatusTone;
  health: string;
  healthTone: StatusTone;
  reason: string;
  requests: number;
  tokens: number;
  share: string;
  shareFraction: number;
  spend: number;
}

const providerColumns: readonly Column<ProviderRow>[] = [
  { key: "name", header: "Connection", primary: "name", secondary: "kind", sort: "name" },
  { key: "status", header: "Status", primary: "status", tone: "statusTone", sort: "status" },
  {
    key: "health",
    header: "Probe",
    primary: "health",
    tone: "healthTone",
    sort: "health",
  },
  { key: "reason", header: "Probe detail", primary: "reason", sort: "reason" },
  {
    key: "requests",
    header: "Requests",
    primary: "requests",
    format: "compact",
    align: "right",
    sort: "requests",
  },
  {
    key: "tokens",
    header: "Tokens",
    primary: "tokens",
    format: "compact",
    align: "right",
    sort: "tokens",
  },
  {
    key: "share",
    header: "Share",
    primary: "share",
    meter: "shareFraction",
    meterMax: 1,
    sort: "shareFraction",
    align: "right",
  },
  {
    key: "spend",
    header: "Spend 30d",
    primary: "spend",
    format: "usd",
    align: "right",
    sort: "spend",
  },
];

const DAY = 24 * 60 * 60 * 1000;

function trailingWindow() {
  return { from: new Date(Date.now() - 30 * DAY) };
}

export default async function ProvidersPage() {
  const { providers, usage } = await getApiClient();
  const window = trailingWindow();
  const [accounts, health, providerUsage, series] = await Promise.all([
    providers.listAccounts(),
    providers.health(),
    usage.byProvider(window),
    usage.timeseries(window),
  ]);

  // by-provider groups on provider_kind, which is a real key on both sides.
  const healthByKind = new Map(health.items.map((row) => [row.provider_kind, row]));
  const usageByKind = new Map(providerUsage.items.map((row) => [row.Provider, row]));

  const totalSpend = providerUsage.items.reduce(
    (sum, row) => sum + microsToUsd(row.CostMicros),
    0,
  );
  const totalRequests = providerUsage.items.reduce((sum, row) => sum + row.Requests, 0);
  const active = accounts.items.filter((account) => account.status === "active").length;
  const degraded = health.items.filter(
    (row) => (row.status !== "healthy" && row.status !== "ok") || Boolean(row.last_error),
  ).length;

  const rows: ProviderRow[] = accounts.items.map((account) => {
    const probe = healthByKind.get(account.provider_kind);
    const attributed = usageByKind.get(account.provider_kind);
    const spend = microsToUsd(attributed?.CostMicros ?? 0);
    const shareFraction = totalSpend > 0 ? spend / totalSpend : 0;
    return {
      id: account.id,
      name: account.name,
      kind: account.provider_kind,
      status: account.status,
      statusTone: toneFor(ACCOUNT_STATUS_TONE, account.status),
      health: probe?.status ?? "unknown",
      healthTone: toneFor(HEALTH_TONE, probe?.status ?? ""),
      reason: probe?.status_reason || probe?.last_error || "—",
      requests: attributed?.Requests ?? 0,
      tokens: attributed?.TotalTokens ?? 0,
      share: formatPercent(shareFraction),
      shareFraction,
      spend,
    };
  });

  const requestRate = series.items.map((point) => ({
    date: point.BucketStart.slice(5, 16).replace("T", " "),
    value: point.Requests,
  }));

  const spendByProvider = providerUsage.items
    .map((row) => ({ label: row.Provider, value: microsToUsd(row.CostMicros) }))
    .sort((a, b) => b.value - a.value);

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Providers"
        description="Upstream connections, what the last probe said, and what each one costs."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Connections"
            value={formatNumber(accounts.items.length)}
            hint={`${formatNumber(active)} active`}
          />
          <StatCard
            label="Failing probes"
            value={formatNumber(degraded)}
            tone={degraded > 0 ? "danger" : "success"}
            hint="Unhealthy or errored"
          />
          <StatCard
            label="Requests"
            value={formatNumber(totalRequests)}
            hint="Trailing 30 days"
          />
          <StatCard
            label="Spend 30d"
            value={formatUsd(totalSpend)}
            hint="Across every provider"
          />
        </section>

        <section className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Spend by provider</CardTitle>
              <CardDescription>Trailing 30 days, highest first.</CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart data={spendByProvider} formatValue={formatUsd} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Gateway request rate</CardTitle>
              <CardDescription>
                Trailing 30 days, aggregated across every provider, so a single
                outage shows as a dip.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <LineChart
                label="Requests"
                data={requestRate}
                color={chartColor(0)}
                formatValue={formatNumber}
              />
            </CardContent>
          </Card>
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Connections</CardTitle>
            <CardDescription>
              Every upstream the gateway holds credentials for.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={providerColumns}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No providers connected."
            />
          </CardContent>
        </Card>

        <section>
          <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
            Status board
          </h2>
          <ul className="mt-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {rows.map((row) => {
              const account = accounts.items.find((item) => item.id === row.id);
              const probe = healthByKind.get(row.kind);
              return (
                <li key={row.id} className="rounded-lg border bg-card p-4">
                  <div className="flex items-start justify-between gap-2">
                    <div>
                      <p className="font-medium">{row.name}</p>
                      <p className="text-xs text-muted-foreground">{row.kind}</p>
                    </div>
                    <StatusBadge tone={row.statusTone}>{row.status}</StatusBadge>
                  </div>
                  <dl className="mt-3 grid grid-cols-2 gap-2 text-xs">
                    <div>
                      <dt className="text-muted-foreground">Probe</dt>
                      <dd className="tabular-nums">
                        {probe?.checked_at ? probe.checked_at.slice(0, 16).replace("T", " ") : "—"}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">Spend 30d</dt>
                      <dd className="tabular-nums">{formatUsd(row.spend)}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">Requests</dt>
                      <dd className="tabular-nums">{formatNumber(row.requests)}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">Connected</dt>
                      <dd className="tabular-nums">{account?.created_at.slice(0, 10) ?? "—"}</dd>
                    </div>
                  </dl>
                  {row.reason !== "—" ? (
                    <p className="mt-3 text-xs text-muted-foreground">{row.reason}</p>
                  ) : null}
                </li>
              );
            })}
          </ul>
        </section>
      </PageShell>
    </div>
  );
}
