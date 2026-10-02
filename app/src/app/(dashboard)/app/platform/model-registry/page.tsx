import type { Metadata } from "next";

import { BarChart, DonutChart } from "@/components/charts";
import { DataTable, PageShell, StatCard, StatusBadge, type Column } from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatCompact, formatUsd, formatUsdExact } from "@/lib/format";
import { modelEntitlements, models, providers } from "@/lib/mock-data";
import { MODEL_STATUS_TONE, type StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Model registry · Sentinel",
  description: "Every model the gateway can route to, with entitlements, pricing, and health.",
};

interface ModelRow {
  id: string;
  displayName: string;
  name: string;
  providerName: string;
  modality: string;
  status: string;
  tone: StatusTone;
  context: string;
  price: string;
  spend: string;
  calls: string;
  health: string;
  entitlement: string;
}

const COLUMNS: readonly Column<ModelRow>[] = [
  { key: "displayName", header: "Model", primary: "displayName", secondary: "name" },
  { key: "providerName", header: "Provider", primary: "providerName" },
  { key: "modality", header: "Modality", primary: "modality" },
  { key: "status", header: "Status", primary: "status", tone: "tone" },
  { key: "context", header: "Context", primary: "context", align: "right" },
  { key: "price", header: "In / out per Mtok", primary: "price", align: "right" },
  { key: "spend", header: "Spend 30d", primary: "spend", align: "right", sort: "spend" },
  { key: "calls", header: "Calls 30d", primary: "calls", align: "right" },
  { key: "health", header: "Health", primary: "health", align: "right" },
  { key: "entitlement", header: "Access", primary: "entitlement" },
];

export default function ModelRegistryPage() {
  const providerName = (id: string) =>
    providers.find((p) => p.id === id)?.displayName ?? "Unknown";

  const rows: ModelRow[] = models.map((model) => {
    const tiers = modelEntitlements
      .filter((entitlement) => entitlement.modelNames.includes(model.name))
      .map((entitlement) => entitlement.tier);
    return {
      id: model.id,
      displayName: model.displayName,
      name: model.name,
      providerName: providerName(model.providerId),
      modality: model.modality,
      status: model.status,
      tone: MODEL_STATUS_TONE[model.status],
      context: formatCompact(model.contextWindowTokens),
      price: `${formatUsdExact(model.pricing.inputPerMTok)} / ${formatUsdExact(model.pricing.outputPerMTok)}`,
      spend: formatUsd(model.spend30d),
      calls: formatCompact(model.callCount30d),
      health: `${model.healthScore}`,
      entitlement: tiers.length > 0 ? tiers.join(", ") : "unassigned",
    };
  });

  const ga = models.filter((m) => m.status === "ga");
  const totalSpend = models.reduce((sum, m) => sum + m.spend30d, 0);
  const avgHealth = Math.round(
    models.reduce((sum, m) => sum + m.healthScore, 0) / models.length,
  );
  const byModality = Object.entries(
    models.reduce<Record<string, number>>((acc, m) => {
      acc[m.modality] = (acc[m.modality] ?? 0) + 1;
      return acc;
    }, {}),
  ).map(([label, value]) => ({ label, value }));

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Model registry"
        description="One catalogue for every model across providers, with the entitlement that governs who may call it."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard label="Models registered" value={String(models.length)} hint={`${ga.length} generally available`} />
          <StatCard label="Spend 30d" value={formatUsd(totalSpend)} hint="Across the whole catalogue" />
          <StatCard
            label="Average health"
            value={`${avgHealth}/100`}
            tone={avgHealth >= 95 ? "success" : "warning"}
            hint="Quality, latency, and cost"
          />
          <StatCard
            label="Providers"
            value={String(new Set(models.map((m) => m.providerId)).size)}
            hint="Feeding the registry"
          />
        </section>

        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Catalogue by modality</CardTitle>
              <CardDescription>How the registered models break down.</CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart data={byModality} centerLabel="models" size={190} />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Spend by model</CardTitle>
              <CardDescription>Top six models by trailing 30-day spend.</CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart
                data={[...models]
                  .sort((a, b) => b.spend30d - a.spend30d)
                  .slice(0, 6)
                  .map((m) => ({ label: m.displayName, value: m.spend30d }))}
                formatValue={(v) => formatUsd(v)}
              />
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Registered models</CardTitle>
            <CardDescription>
              Access reflects the model entitlement policy currently in force.
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
            <CardTitle>Entitlement coverage</CardTitle>
            <CardDescription>Which plan tiers may call each registered model.</CardDescription>
          </CardHeader>
          <CardContent>
            <ul className="flex flex-wrap gap-2">
              {rows.map((row) => (
                <li key={row.id}>
                  <StatusBadge tone={row.entitlement === "unassigned" ? "warning" : "success"}>
                    {row.displayName}: {row.entitlement}
                  </StatusBadge>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
