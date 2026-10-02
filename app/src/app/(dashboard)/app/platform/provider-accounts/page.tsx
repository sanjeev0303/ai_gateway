import type { Metadata } from "next";
import { Plus } from "lucide-react";

import { BarChart, DonutChart } from "@/components/charts";
import { DataTable, PageShell, StatCard, StatusBadge, type Column } from "@/components/dashboard";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatMs, formatPercent, formatUsd } from "@/lib/format";
import { models, providers } from "@/lib/mock-data";
import { PROVIDER_STATUS_TONE } from "@/types";

export const metadata: Metadata = {
  title: "Provider accounts · Sentinel",
  description: "Connected upstream providers, their models, and per-account fees.",
};

interface AccountRow {
  id: string;
  displayName: string;
  name: string;
  status: string;
  tone: (typeof PROVIDER_STATUS_TONE)[keyof typeof PROVIDER_STATUS_TONE];
  uptime: string;
  p95: string;
  models: string;
  fee: string;
}

const COLUMNS: readonly Column<AccountRow>[] = [
  { key: "displayName", header: "Provider", primary: "displayName", secondary: "name" },
  { key: "status", header: "Status", primary: "status", tone: "tone" },
  { key: "uptime", header: "Uptime 30d", primary: "uptime", align: "right", sort: "uptime" },
  { key: "p95", header: "Latency p95", primary: "p95", align: "right" },
  { key: "models", header: "Models", primary: "models", align: "right" },
  { key: "fee", header: "Platform fee", primary: "fee", align: "right" },
];

export default function ProviderAccountsPage() {
  const rows: AccountRow[] = providers.map((provider) => ({
    id: provider.id,
    displayName: provider.displayName,
    name: provider.name,
    status: provider.status,
    tone: PROVIDER_STATUS_TONE[provider.status],
    uptime: formatPercent(provider.uptime30d, 2),
    p95: formatMs(provider.latencyP95Ms),
    models: String(provider.models.length),
    fee: provider.monthlyFee === 0 ? "Included" : formatUsd(provider.monthlyFee),
  }));

  const degraded = providers.filter((p) => p.status !== "healthy");
  const totalFee = providers.reduce((sum, p) => sum + p.monthlyFee, 0);
  const modelSplit = providers
    .map((provider) => ({
      label: provider.displayName,
      value: models.filter((m) => m.providerId === provider.id).length,
    }))
    .sort((a, b) => b.value - a.value);

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Provider accounts"
        description="Every upstream account behind one gateway key, with live status and catalogue size."
        actions={
          <Button size="sm">
            <Plus aria-hidden /> Connect provider
          </Button>
        }
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard label="Connected accounts" value={String(providers.length)} hint="Across 3 regions" />
          <StatCard
            label="Healthy"
            value={String(providers.length - degraded.length)}
            tone="success"
            hint={`${degraded.length} needing attention`}
          />
          <StatCard label="Total models" value={String(models.length)} hint="Enabled in the registry" />
          <StatCard
            label="Platform fees"
            value={totalFee === 0 ? "Included" : formatUsd(totalFee)}
            hint="Recurring, per account"
          />
        </section>

        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Catalogue size by provider</CardTitle>
              <CardDescription>How many models each account exposes.</CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart data={modelSplit} centerLabel="models" size={190} />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Slowest accounts</CardTitle>
              <CardDescription>Ranked by 95th-percentile first-token latency.</CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart
                data={[...providers]
                  .sort((a, b) => b.latencyP95Ms - a.latencyP95Ms)
                  .slice(0, 5)
                  .map((p) => ({ label: p.displayName, value: p.latencyP95Ms }))}
                formatValue={(v) => formatMs(v)}
              />
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Accounts</CardTitle>
            <CardDescription>Status is derived from live traffic, not manual reporting.</CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={rows}
              rowKey="id"
              toolbar={
                <div className="flex items-center gap-2">
                  {degraded.length > 0 ? (
                    <StatusBadge tone="warning">
                      {degraded.length} degraded
                    </StatusBadge>
                  ) : (
                    <StatusBadge tone="success">All healthy</StatusBadge>
                  )}
                  <Badge variant="outline">Auto-failover on</Badge>
                </div>
              }
              emptyMessage="No provider accounts connected."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
