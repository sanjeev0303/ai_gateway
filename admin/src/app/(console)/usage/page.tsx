import type { Metadata } from "next";

import type { AdminUsage, AdminUsageRow } from "@ai-governance/api-client";
import { AreaChart } from "@ai-governance/console-ui/charts/area-chart";
import { chartColor } from "@ai-governance/console-ui/charts/utils";
import { DataTable, type Column } from "@ai-governance/console-ui/dashboard/data-table";
import { PageShell } from "@ai-governance/console-ui/dashboard/page-shell";
import { RangeTabs } from "@ai-governance/console-ui/dashboard/range-tabs";
import { ErrorState } from "@ai-governance/console-ui/dashboard/state";
import { formatCompact, formatProviderName, formatUsd, microsToUsd } from "@ai-governance/console-ui/lib/format";
import { getApiClient } from "@/lib/session";
import { Panel, RegisterEmpty } from "../_components/ledger";
import { usd, WINDOW_LABEL, windowFrom, type SearchParams } from "../_lib";

export const metadata: Metadata = { title: "Usage · Platform console" };

interface BreakdownRow {
  id: string;
  label: string;
  href: string;
  requests: number;
  tokens: number;
  spend: string;
  spendMicros: number;
  latency: number;
}

function rows(items: AdminUsageRow[], label: (row: AdminUsageRow) => string, href?: (row: AdminUsageRow) => string): BreakdownRow[] {
  return items.map((row) => ({
    id: row.key,
    label: label(row),
    href: href ? href(row) : "",
    requests: row.requests,
    tokens: row.tokens,
    spend: usd(row.cost_micros),
    spendMicros: row.cost_micros,
    latency: row.avg_latency_ms,
  }));
}

function columns(header: string, linked: boolean): Column<BreakdownRow>[] {
  return [
    { key: "label", header, primary: "label", href: linked ? "href" : undefined, grow: true },
    { key: "requests", header: "Requests", primary: "requests", format: "compact", align: "right" },
    { key: "tokens", header: "Tokens", primary: "tokens", format: "compact", align: "right" },
    { key: "spend", header: "Spend", primary: "spend", sort: "spendMicros", align: "right" },
    { key: "latency", header: "Avg latency", primary: "latency", format: "ms", align: "right" },
  ];
}

export default async function AdminUsagePage({ searchParams }: { searchParams: SearchParams }) {
  const window = windowFrom(await searchParams);

  let usage: AdminUsage;
  try {
    const { admin } = await getApiClient();
    usage = await admin.usage(window);
  } catch {
    return (
      <PageShell title="Usage">
        <ErrorState description="Platform usage could not be loaded. Check that the control API is running, then reload." />
      </PageShell>
    );
  }

  const requests = usage.daily.map((d) => ({ date: d.day.slice(5), value: d.requests }));
  const spend = usage.daily.map((d) => ({ date: d.day.slice(5), value: microsToUsd(d.cost_micros) }));

  return (
    <PageShell
      title="Usage"
      description={`Gateway traffic across every organization over the ${WINDOW_LABEL[window]}, by UTC day. Spend is priced at request time.`}
      actions={<RangeTabs />}
    >
      <div className="flex flex-col gap-6 pb-12">
        <div className="grid gap-5 lg:grid-cols-2">
          <Panel title="Requests per day">
            <div className="p-4">
              {requests.length > 0 ? (
                <AreaChart label="Platform requests per day" series={[{ label: "Requests", color: chartColor(0), data: requests }]} formatValue={formatCompact} />
              ) : (
                <RegisterEmpty>No gateway requests in this range.</RegisterEmpty>
              )}
            </div>
          </Panel>
          <Panel title="Spend per day">
            <div className="p-4">
              {spend.length > 0 ? (
                <AreaChart label="Platform spend per day" series={[{ label: "Spend", color: chartColor(1), data: spend }]} formatValue={formatUsd} />
              ) : (
                <RegisterEmpty>No spend in this range.</RegisterEmpty>
              )}
            </div>
          </Panel>
        </div>

        <Panel title="By organization" description="The 20 organizations with the most spend.">
          <div className="p-4">
            <DataTable
              columns={columns("Organization", true)}
              rows={rows(usage.by_org, (r) => r.label, (r) => `/organizations/${r.key}`)}
              rowKey="id"
              emptyMessage="No organization sent traffic in this range."
            />
          </div>
        </Panel>

        <div className="grid gap-5 xl:grid-cols-2">
          <Panel title="By provider">
            <div className="p-4">
              <DataTable
                columns={columns("Provider", false)}
                rows={rows(usage.by_provider, (r) => formatProviderName(r.label))}
                rowKey="id"
                emptyMessage="No provider served traffic in this range."
              />
            </div>
          </Panel>
          <Panel title="By model" description="The 20 models with the most spend.">
            <div className="p-4">
              <DataTable columns={columns("Model", false)} rows={rows(usage.by_model, (r) => r.label)} rowKey="id" emptyMessage="No model served traffic in this range." />
            </div>
          </Panel>
        </div>
      </div>
    </PageShell>
  );
}
