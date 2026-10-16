import { Metadata } from "next";
import { DataTable, PageShell, StatCard, type Column } from "@/components/dashboard";
import { getApiClient } from "@/lib/session";
import { notFound } from "next/navigation";
import { formatCompact, formatUsd, formatNumber, formatMs, formatPercent, formatDate, formatRelative, microsToUsd } from "@/lib/format";
import type { StatusTone } from "@/types";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

interface ModelUsageRow {
  id: string;
  name: string;
  href: string;
  calls: number;
  tokens: number;
  cost: number;
}

const modelBreakdownColumns: readonly Column<ModelUsageRow>[] = [
  { key: "name", header: "Model", primary: "name", sort: "name", href: "href" },
  { key: "calls", header: "Requests", primary: "calls", format: "compact", align: "right", sort: "calls" },
  { key: "tokens", header: "Tokens", primary: "tokens", format: "compact", align: "right", sort: "tokens" },
  { key: "cost", header: "Spend", primary: "cost", format: "usd", align: "right", sort: "cost" },
];

interface AccountHealthRow {
  id: string;
  name: string;
  status: string;
  tone: StatusTone;
  lastProbe: string;
}

const accountColumns: readonly Column<AccountHealthRow>[] = [
  { key: "name", header: "Account", primary: "name", sort: "name" },
  { key: "status", header: "Status", primary: "status", tone: "tone", sort: "status" },
  { key: "lastProbe", header: "Last Probed", primary: "lastProbe", sort: "lastProbe" },
];

export async function generateMetadata({ params }: { params: Promise<{ id: string }> }): Promise<Metadata> {
  const { id } = await params;
  const decoded = decodeURIComponent(id);
  return { title: `Provider ${decoded} · Sentinel` };
}

export default async function ProviderDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const providerKind = decodeURIComponent(id);
  const { usage, providers, models } = await getApiClient();

  try {
    const [
      summary,
      latencyData,
      byModel,
      accountsList,
      catalogList,
    ] = await Promise.all([
      usage.summary({ provider: providerKind }),
      usage.latency({ provider: providerKind }),
      usage.byModel({ provider: providerKind }),
      providers.listAccounts(),
      models.list(),
    ]);

    const accounts = (accountsList?.items ?? []).filter(
      (a: { provider_kind: string }) => a.provider_kind.toLowerCase() === providerKind.toLowerCase()
    );
    const catalogForProvider = (catalogList?.items ?? []).filter(
      (m: { provider_kind: string }) => m.provider_kind.toLowerCase() === providerKind.toLowerCase()
    );

    const totalRequests = summary?.Requests ?? 0;
    const totalTokens = summary?.TotalTokens ?? 0;
    const totalCost = microsToUsd(summary?.TotalCostMicros ?? 0);
    const errorCount = summary?.Errors ?? 0;
    const errorRate = totalRequests > 0 ? errorCount / totalRequests : 0;

    const latencyOverall = latencyData?.items?.find((r: { Model?: string }) => !r.Model) ?? latencyData?.items?.[0];
    const p95Latency = latencyOverall?.P95TotalMs ?? latencyOverall?.AvgTotalMs;

    const modelNameMap = new Map(catalogForProvider.map((m: { model_slug: string; display_name: string }) => [m.model_slug, m.display_name]));
    const modelRows: ModelUsageRow[] = (byModel?.items ?? []).map((m: { Model: string; Requests: number; TotalTokens: number; CostMicros: number }) => ({
      id: m.Model,
      name: modelNameMap.get(m.Model) ?? m.Model,
      href: `/app/observe/models/${encodeURIComponent(m.Model)}`,
      calls: m.Requests,
      tokens: m.TotalTokens,
      cost: microsToUsd(m.CostMicros),
    }));

    const accountRows: AccountHealthRow[] = accounts.map((a: { id: string; name: string; status: string; updated_at?: string }) => ({
      id: a.id,
      name: a.name,
      status: a.status,
      tone: a.status === "active" ? "success" : "neutral",
      lastProbe: a.updated_at ? formatRelative(a.updated_at) : "Never",
    }));

    return (
      <div className="p-4 sm:p-6">
        <PageShell
          title={`Provider: ${providerKind}`}
          description={`Operational telemetry, connected accounts, and model metrics for ${providerKind}.`}
        >
          <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-5">
            <StatCard label="Total Requests" value={formatCompact(totalRequests)} />
            <StatCard label="Total Tokens" value={formatCompact(totalTokens)} />
            <StatCard label="Estimated Spend" value={formatUsd(totalCost)} />
            <StatCard label="P95 Latency" value={p95Latency ? formatMs(p95Latency) : "—"} />
            <StatCard
              label="Error Rate"
              value={formatPercent(errorRate, 2)}
              tone={errorRate >= 0.05 ? "danger" : errorRate >= 0.02 ? "warning" : "success"}
            />
          </section>

          <div className="grid gap-6 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Connected Provider Accounts</CardTitle>
              </CardHeader>
              <CardContent>
                <DataTable columns={accountColumns} rows={accountRows} rowKey="id" pageSize={5} emptyMessage="No accounts configured for this provider." />
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Model Breakdown</CardTitle>
              </CardHeader>
              <CardContent>
                <DataTable columns={modelBreakdownColumns} rows={modelRows} rowKey="id" pageSize={5} emptyMessage="No model usage recorded for this provider." />
              </CardContent>
            </Card>
          </div>
        </PageShell>
      </div>
    );
  } catch (err) {
    notFound();
  }
}
