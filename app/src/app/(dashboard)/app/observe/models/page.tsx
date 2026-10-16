import type { Metadata } from "next";

import { BarChart } from "@/components/charts";
import {
  DataTable,
  ObserveFilters,
  PageShell,
  RangeTabs,
  StatCard,
  type Column,
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
  formatUsd,
  microsToUsd,
} from "@/lib/format";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Models · Sentinel",
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

interface CatalogRow {
  id: string;
  name: string;
  slug: string;
  provider: string;
  active: string;
  activeTone: StatusTone;
  detailUrl: string;
}

const catalogColumns: readonly Column<CatalogRow>[] = [
  { key: "name", header: "Model", primary: "name", secondary: "slug", sort: "name", href: "detailUrl" },
  { key: "provider", header: "Provider", primary: "provider", sort: "provider" },
  { key: "active", header: "Status", primary: "active", tone: "activeTone", sort: "active" },
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
  const totalRequests = modelUsage.items.reduce((sum, row) => sum + row.Requests, 0);
  const activeCatalog = catalog.items.filter((model) => model.is_active).length;

  const rows: ModelUsageRow[] = modelUsage.items.map((row) => {
    const matched = byIdentifier.get(row.Model);
    const { label: active, tone: activeTone } = activeState(matched?.is_active);
    const spend = microsToUsd(row.CostMicros);
    const shareFraction = totalSpend > 0 ? spend / totalSpend : 0;
    return {
      id: row.Model,
      name: matched?.display_name ?? row.Model,
      slug: matched?.model_slug ?? row.Model,
      provider: matched?.provider_kind ?? "Unlisted",
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

  const catalogRows: CatalogRow[] = catalog.items.map((model) => {
    const { label: active, tone: activeTone } = activeState(model.is_active);
    return {
      id: model.id,
      name: model.display_name,
      slug: model.model_slug,
      provider: model.provider_kind,
      active,
      activeTone,
      detailUrl: `/app/observe/models/${encodeURIComponent(model.model_slug)}`,
    };
  });

  const spendByModel = [...rows]
    .sort((a, b) => b.spend - a.spend)
    .map((model) => ({ label: model.name, value: model.spend }));

  const unlisted = rows.filter((model) => model.active === "unlisted").length;

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Models"
        description="Everything the gateway has routed, and the spend each one attracts."
        actions={<RangeTabs />}
      >
        <ObserveFilters />
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Models in catalog"
            value={formatNumber(catalog.items.length)}
            hint={`${formatNumber(activeCatalog)} active`}
          />
          <StatCard
            label="Models used"
            value={formatNumber(rows.length)}
            tone={unlisted > 0 ? "warning" : "neutral"}
            hint={unlisted > 0 ? `${formatNumber(unlisted)} not in catalog` : RANGE_HINT[range]}
          />
          <StatCard
            label="Requests"
            value={formatNumber(totalRequests)}
            hint={RANGE_HINT[range]}
          />
          <StatCard
            label="Spend across models"
            value={formatUsd(totalSpend)}
            hint={RANGE_HINT[range]}
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Spend by model</CardTitle>
            <CardDescription>
              Trailing 30 days, highest first. A long tail is normal; a single
              model dominating is a review signal.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <BarChart data={spendByModel} formatValue={formatUsd} />
          </CardContent>
        </Card>

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

        <Card>
          <CardHeader>
            <CardTitle>Model catalog</CardTitle>
            <CardDescription>
              Every model the gateway knows about, including ones with no recent
              traffic.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={catalogColumns}
              rows={catalogRows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No models registered."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
