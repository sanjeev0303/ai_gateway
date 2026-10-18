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
import { ModelRegistryClient, SyncModelsDialog } from "./client";

export const metadata: Metadata = {
  title: "Model registry · Sentinel",
  description: "Every model the gateway can route to, per provider, with sync capability.",
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
  display_name: string;
  model_slug: string;
  provider_kind: string;
  capabilities?: string[];
  context_limit?: number;
  supports_streaming?: boolean;
  supports_tools?: boolean;
  supports_reasoning?: boolean;
  status?: string;
  is_active: boolean;
  updated_at: string;
}

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
      return {
        id: model.id,
        display_name: model.display_name,
        model_slug: model.model_slug,
        provider_kind: model.provider_kind,
        capabilities: model.capabilities,
        context_limit: model.context_limit,
        supports_streaming: model.supports_streaming,
        supports_tools: model.supports_tools,
        supports_reasoning: model.supports_reasoning,
        status: model.status,
        is_active: model.is_active,
        updated_at: model.updated_at,
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
    active: rows.filter((row) => row.is_active).length,
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
        actions={<SyncModelsDialog />}
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

        <ModelRegistryClient initialRows={rows} />

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
