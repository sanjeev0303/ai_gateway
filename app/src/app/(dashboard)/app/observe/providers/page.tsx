import type { Metadata } from "next";

import { BarChart, chartColor, LineChart } from "@/components/charts";
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
import { formatMs, formatNumber, formatPercent, formatUsd } from "@/lib/format";
import { NOW, models, platformHealth, providers } from "@/lib/mock-data";
import { PROVIDER_STATUS_TONE } from "@/types";
import type { StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Providers · Sentinel",
  description: "Upstream health, latency, and cost for every connected provider.",
};

function latencyTone(ms: number): StatusTone {
  if (ms >= 4_000) return "danger";
  if (ms >= 2_500) return "warning";
  return "success";
}

interface ProviderRow {
  id: string;
  name: string;
  status: string;
  statusTone: StatusTone;
  uptime: number;
  modelCount: number;
  p50: number;
  p95: number;
  p95Tone: StatusTone;
  fee: number;
  lastIncident: string;
}

const providerColumns: readonly Column<ProviderRow>[] = [
  { key: "name", header: "Provider", primary: "name", sort: "name" },
  { key: "status", header: "Status", primary: "status", tone: "statusTone", sort: "status" },
  {
    key: "uptime",
    header: "Uptime 30d",
    primary: "uptime",
    format: "percentRaw",
    meter: "uptime",
    meterMax: 100,
    align: "right",
    sort: "uptime",
  },
  {
    key: "modelCount",
    header: "Models",
    primary: "modelCount",
    format: "number",
    align: "right",
    sort: "modelCount",
  },
  {
    key: "p50",
    header: "TTFT p50",
    primary: "p50",
    format: "ms",
    align: "right",
    sort: "p50",
  },
  {
    key: "p95",
    header: "TTFT p95",
    primary: "p95",
    format: "ms",
    tone: "p95Tone",
    align: "right",
    sort: "p95",
  },
  {
    key: "fee",
    header: "Platform fee",
    primary: "fee",
    format: "usd",
    align: "right",
    sort: "fee",
  },
  { key: "lastIncident", header: "Last incident", primary: "lastIncident", sort: "lastIncident" },
];

const rows: ProviderRow[] = providers.map((provider) => ({
  id: provider.id,
  name: provider.displayName,
  status: provider.status,
  statusTone: PROVIDER_STATUS_TONE[provider.status],
  uptime: provider.uptime30d,
  modelCount: provider.models.length,
  p50: provider.latencyP50Ms,
  p95: provider.latencyP95Ms,
  p95Tone: latencyTone(provider.latencyP95Ms),
  fee: provider.monthlyFee,
  lastIncident: provider.lastIncidentAt ?? "None recorded",
}));

const healthy = providers.filter((provider) => provider.status === "healthy").length;
const avgUptime =
  providers.reduce((acc, provider) => acc + provider.uptime30d, 0) / providers.length;
const worstP95 = Math.max(...providers.map((provider) => provider.latencyP95Ms));
const totalFees = providers.reduce((acc, provider) => acc + provider.monthlyFee, 0);

export default function ProvidersPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Providers"
        description="Upstream health, time to first token, and the fixed platform fee each connection costs."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Providers healthy"
            value={`${formatNumber(healthy)} of ${formatNumber(providers.length)}`}
            tone={healthy === providers.length ? "success" : "warning"}
            hint="Current status"
          />
          <StatCard
            label="Average uptime"
            value={formatPercent(avgUptime / 100)}
            hint="Trailing 30 days"
          />
          <StatCard
            label="Slowest p95"
            value={formatMs(worstP95)}
            tone={latencyTone(worstP95)}
            hint="Time to first token"
          />
          <StatCard
            label="Platform fees"
            value={formatUsd(totalFees)}
            hint="Per month, before usage"
          />
        </section>

        <section className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Uptime by provider</CardTitle>
              <CardDescription>Trailing 30-day availability.</CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart
                data={providers.map((provider) => ({
                  label: provider.displayName,
                  value: provider.uptime30d,
                }))}
                formatValue={(value) => formatPercent(value / 100)}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Time to first token</CardTitle>
              <CardDescription>
                p50 and p95 per provider. The gap is queueing, not generation.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart
                label="TTFT"
                data={providers.map((provider) => ({
                  label: provider.displayName,
                  value: provider.latencyP95Ms,
                }))}
                formatValue={formatMs}
              />
            </CardContent>
          </Card>
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Gateway request rate</CardTitle>
            <CardDescription>
              Aggregated across every provider, so a single outage shows as a
              dip.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <LineChart
              label="Gateway request rate"
              data={platformHealth.requestVolume}
              color={chartColor(0)}
              formatValue={formatNumber}
            />
          </CardContent>
        </Card>

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
            {providers.map((provider) => {
              const owned = models.filter((model) => model.providerId === provider.id);
              return (
                <li key={provider.id} className="rounded-lg border bg-card p-4">
                  <div className="flex items-start justify-between gap-2">
                    <div>
                      <p className="font-medium">{provider.displayName}</p>
                      <p className="text-xs text-muted-foreground">
                        {formatNumber(provider.models.length)} models ·{" "}
                        {formatUsd(provider.monthlyFee)}/mo
                      </p>
                    </div>
                    <StatusBadge tone={PROVIDER_STATUS_TONE[provider.status]}>
                      {provider.status}
                    </StatusBadge>
                  </div>
                  <dl className="mt-3 grid grid-cols-2 gap-2 text-xs">
                    <div>
                      <dt className="text-muted-foreground">Uptime</dt>
                      <dd className="tabular-nums">
                        {formatPercent(provider.uptime30d / 100)}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">p95 TTFT</dt>
                      <dd className="tabular-nums">{formatMs(provider.latencyP95Ms)}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">Spend 30d</dt>
                      <dd className="tabular-nums">
                        {formatUsd(owned.reduce((acc, model) => acc + model.spend30d, 0))}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">Connected</dt>
                      <dd className="tabular-nums">{provider.createdAt.slice(0, 10)}</dd>
                    </div>
                  </dl>
                </li>
              );
            })}
          </ul>
        </section>

        <p className="text-xs text-muted-foreground">
          Status reflects the probe taken as of {NOW.toISOString().slice(0, 16).replace("T", " ")}{" "}
          UTC.
        </p>
      </PageShell>
    </div>
  );
}
