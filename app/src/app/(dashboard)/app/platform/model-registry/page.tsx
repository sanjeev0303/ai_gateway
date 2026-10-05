import type { Metadata } from "next";

import { BarChart, DonutChart } from "@/components/charts";
import {
  DataTable,
  PageShell,
  StatCard,
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
  formatCompact,
  formatDate,
  formatNumber,
  formatUsd,
  microsToUsd,
} from "@/lib/format";
import { toneFor } from "@/lib/org-tones";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Model registry · Sentinel",
  description: "Every model the gateway can route to, per provider.",
};

const DAY = 24 * 60 * 60 * 1000;

const ACTIVE_TONE: Record<string, StatusTone> = {
  active: "success",
  inactive: "neutral",
};

const display = (kind: string) =>
  kind
    .split("_")
    .map((part) => (part ? part[0].toUpperCase() + part.slice(1) : part))
    .join(" ");

interface ModelRow {
  id: string;
  displayName: string;
  slug: string;
  provider: string;
  status: string;
  tone: StatusTone;
  spend: string;
  requests: string;
  updated: string;
}

const COLUMNS: readonly Column<ModelRow>[] = [
  {
    key: "model",
    header: "Model",
    primary: "displayName",
    secondary: "slug",
    sort: "displayName",
  },
  { key: "provider", header: "Provider", primary: "provider", sort: "provider" },
  { key: "status", header: "Status", primary: "status", tone: "tone", sort: "status" },
  { key: "spend", header: "Spend 30d", primary: "spend", align: "right" },
  { key: "requests", header: "Requests 30d", primary: "requests", align: "right" },
  {
    key: "updated",
    header: "Updated",
    primary: "updated",
    sort: "updated",
    align: "right",
  },
];

async function loadModelRegistry() {
  const client = await getApiClient();
  const from = new Date(Date.now() - 30 * DAY);

  const [registry, usageByModel] = await Promise.all([
    client.models.list(),
    client.usage.byModel({ from }),
  ]);

  // usage_hourly/daily/monthly group by the raw request `model`, which the
  // pricing join pins to pricing_models.model_slug — so slug is the join key.
  const usage = new Map(usageByModel.items.map((row) => [row.Model, row]));

  const rows: ModelRow[] = [...registry.items]
    .sort((a, b) => a.display_name.localeCompare(b.display_name))
    .map((model) => {
      const stats = usage.get(model.model_slug);
      const status = model.is_active ? "active" : "inactive";
      return {
        id: model.id,
        displayName: model.display_name,
        slug: model.model_slug,
        provider: display(model.provider_kind),
        status,
        tone: toneFor(ACTIVE_TONE, status),
        spend: stats ? formatUsd(microsToUsd(stats.CostMicros)) : "—",
        requests: stats ? formatCompact(stats.Requests) : "—",
        updated: formatDate(model.updated_at),
      };
    });

  const kindCounts = new Map<string, number>();
  for (const model of registry.items) {
    kindCounts.set(
      display(model.provider_kind),
      (kindCounts.get(display(model.provider_kind)) ?? 0) + 1,
    );
  }

  const nameBySlug = new Map(
    registry.items.map((model) => [model.model_slug, model.display_name]),
  );

  const topSpend = [...usageByModel.items]
    .sort((a, b) => b.CostMicros - a.CostMicros)
    .slice(0, 6)
    .map((row) => ({
      label: nameBySlug.get(row.Model) ?? row.Model,
      value: microsToUsd(row.CostMicros),
    }));

  return {
    rows,
    topSpend,
    registered: rows.length,
    active: rows.filter((row) => row.status === "active").length,
    providers: kindCounts.size,
    byProvider: [...kindCounts.entries()]
      .map(([label, value]) => ({ label, value }))
      .sort((a, b) => b.value - a.value),
  };
}

export default async function ModelRegistryPage() {
  const { rows, topSpend, registered, active, providers, byProvider } =
    await loadModelRegistry();

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Model registry"
        description="Every model the gateway can route to, grouped by the provider that serves it."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Models registered"
            value={formatNumber(registered)}
            hint="Across all providers"
          />
          <StatCard
            label="Active"
            value={formatNumber(active)}
            tone={active === registered ? "success" : "neutral"}
            hint="Routable"
          />
          <StatCard
            label="Inactive"
            value={formatNumber(registered - active)}
            tone={registered - active > 0 ? "warning" : "success"}
            hint="Registered but not routable"
          />
          <StatCard
            label="Providers"
            value={formatNumber(providers)}
            hint="Feeding the registry"
          />
        </section>

        <section className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Catalogue by provider</CardTitle>
              <CardDescription>Where the registered models come from.</CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart
                data={byProvider}
                centerLabel={formatNumber(registered)}
                formatValue={(value) => `${formatNumber(value)} models`}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Spend by model</CardTitle>
              <CardDescription>
                Top six models by trailing 30-day cost. A model can appear here
                before it is registered.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart data={topSpend} formatValue={(value) => formatUsd(value)} />
            </CardContent>
          </Card>
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Registered models</CardTitle>
            <CardDescription>
              Spend and request counts come from the usage rollups over the same
              trailing 30 days.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No models registered."
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Capabilities, pricing, and entitlements</CardTitle>
            <CardDescription>Not part of a registry record.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2 text-sm text-muted-foreground">
            <p>
              A registry record holds a provider, a model slug, a display name,
              and whether it is active. It carries no modality, no context window,
              no list price, and no health score, so this page cannot rank models
              by capability or cost per token.
            </p>
            <p>
              Derived cost is priced by the worker from the price sheet in force
              at request time, but there is no control API for price sheets, and
              no policy API governs which tier may call which model. Observed
              spend is therefore the only cost signal available here.
            </p>
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
