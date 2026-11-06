import type { Metadata } from "next";

import {
  DataTable,
  ObserveFilters,
  PageShell,
  RangeTabs,
  StatCard,
  type Column,
  BreakdownTabs,
} from "@/components/dashboard";
import { isDateRange, type DateRange } from "@/lib/date-range";
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
  formatProviderName,
  microsToUsd,
} from "@/lib/format";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Models · Tollseal",
  description: "Every model on the gateway, and the spend each one attracts.",
};

interface ModelUsageRow {
  id: string;
  name: string;
  slug: string;
  provider: string;
  active: string;
  activeTone: StatusTone;
  requests: number;
  tokens: number;
  share: string;
  shareFraction: number;
  spend: number;
  detailUrl: string;
}

const modelColumns: readonly Column<ModelUsageRow>[] = [
  { key: "name", header: "Model", primary: "name", secondary: "slug", sort: "name", href: "detailUrl" },
  { key: "provider", header: "Provider", primary: "provider", sort: "provider" },
  { key: "active", header: "Status", primary: "active", tone: "activeTone", sort: "active" },
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

/** The API exposes activity as a boolean, not a status string. */
function activeState(isActive: boolean | undefined): { label: string; tone: StatusTone } {
  if (isActive === undefined) return { label: "unlisted", tone: "neutral" };
  return isActive
    ? { label: "active", tone: "success" }
    : { label: "inactive", tone: "neutral" };
}

const RANGE_MS: Record<DateRange, number> = {
  "24h": 24 * 60 * 60 * 1000,
  "7d": 7 * 24 * 60 * 60 * 1000,
  "30d": 30 * 24 * 60 * 60 * 1000,
  "90d": 90 * 24 * 60 * 60 * 1000,
};

const RANGE_HINT: Record<DateRange, string> = {
  "24h": "Last 24 hours",
  "7d": "Last 7 days",
  "30d": "Last 30 days",
  "90d": "Last 90 days",
};

export default async function ModelsPage({
  searchParams,
}: {
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}) {
  const params = await searchParams;
  const raw = typeof params.range === "string" ? params.range : null;
  const range: DateRange = isDateRange(raw) ? raw : "30d";

  const { models, usage } = await getApiClient();
  
  const filters = {
    project_id: typeof params.project_id === "string" ? params.project_id : undefined,
    user_id: typeof params.user_id === "string" ? params.user_id : undefined,
    team_id: typeof params.team_id === "string" ? params.team_id : undefined,
    service_account_id: typeof params.service_account_id === "string" ? params.service_account_id : undefined,
    provider: typeof params.provider === "string" ? params.provider : undefined,
    model: typeof params.model === "string" ? params.model : undefined,
    status: typeof params.status === "string" ? params.status : undefined,
  };

  const to = new Date();
  const query = { from: new Date(to.getTime() - RANGE_MS[range]), to, ...filters };

  const [catalog, modelUsage] = await Promise.all([models.list(), usage.byModel(query)]);

  // usage_events.model is the identifier recorded at request time, which is not
  // guaranteed to match a catalog slug. Match either, and fall back to the raw
  // string rather than dropping the row or inventing a name.
  const byIdentifier = new Map<string, (typeof catalog.items)[number]>();
  for (const model of catalog.items) {
    byIdentifier.set(model.model_slug, model);
    byIdentifier.set(model.display_name, model);
  }

  const totalSpend = modelUsage.items.reduce(
    (sum, row) => sum + microsToUsd(row.CostMicros),
    0,
  );

  const rows: ModelUsageRow[] = modelUsage.items.map((row) => {
    const matched = byIdentifier.get(row.Model);
    const { label: active, tone: activeTone } = activeState(matched?.is_active);
    const spend = microsToUsd(row.CostMicros);
    const shareFraction = totalSpend > 0 ? spend / totalSpend : 0;
    return {
      id: row.Model,
      name: matched?.display_name ?? row.Model,
      slug: matched?.model_slug ?? row.Model,
      provider: formatProviderName(matched?.provider_kind, row.Model),
      active,
      activeTone,
      requests: row.Requests,
      tokens: row.TotalTokens,
      share: formatPercent(shareFraction),
      shareFraction,
      spend,
      detailUrl: `/app/observe/models/${encodeURIComponent(row.Model)}`,
    };
  });

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Models"
        description="Everything the gateway has routed, and the spend each one attracts."
        actions={<RangeTabs />}
      >
        <BreakdownTabs />
        <ObserveFilters />
        <section className="grid gap-4 sm:grid-cols-2">
          <StatCard
            label="Models used"
            value={formatNumber(rows.length)}
            hint={RANGE_HINT[range]}
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Model usage</CardTitle>
            <CardDescription>
              Ranked by spend over the trailing 30 days.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={modelColumns}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No model usage recorded."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
