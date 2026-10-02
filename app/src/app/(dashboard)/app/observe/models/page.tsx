import type { Metadata } from "next";

import { BarChart } from "@/components/charts";
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
import { formatNumber, formatUsd, formatUsdExact } from "@/lib/format";
import { models, providers } from "@/lib/mock-data";
import { MODEL_STATUS_TONE } from "@/types";
import type { ModelModality, StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Models · Sentinel",
  description: "Every model on the gateway with its price, health, and usage.",
};

const MODALITY_TONE: Record<ModelModality, StatusTone> = {
  text: "neutral",
  vision: "info",
  audio: "info",
  embedding: "neutral",
  multimodal: "success",
};

const providerName = new Map(providers.map((provider) => [provider.id, provider.displayName]));

function healthTone(score: number): StatusTone {
  if (score >= 80) return "success";
  if (score >= 60) return "warning";
  return "danger";
}

interface ModelRow {
  id: string;
  name: string;
  provider: string;
  modality: string;
  modalityTone: StatusTone;
  status: string;
  statusTone: StatusTone;
  context: number;
  maxOutput: number;
  input: number;
  output: number;
  calls: number;
  spend: number;
  health: number;
  healthTone: StatusTone;
}

const modelColumns: readonly Column<ModelRow>[] = [
  { key: "name", header: "Model", primary: "name", secondary: "provider", sort: "name" },
  {
    key: "modality",
    header: "Modality",
    primary: "modality",
    tone: "modalityTone",
    sort: "modality",
  },
  { key: "status", header: "Status", primary: "status", tone: "statusTone", sort: "status" },
  {
    key: "context",
    header: "Context",
    primary: "context",
    format: "compact",
    align: "right",
    sort: "context",
  },
  {
    key: "maxOutput",
    header: "Max out",
    primary: "maxOutput",
    format: "compact",
    align: "right",
    sort: "maxOutput",
  },
  {
    key: "input",
    header: "In / Mtok",
    primary: "input",
    format: "usdExact",
    align: "right",
    sort: "input",
  },
  {
    key: "output",
    header: "Out / Mtok",
    primary: "output",
    format: "usdExact",
    align: "right",
    sort: "output",
  },
  {
    key: "calls",
    header: "Requests",
    primary: "calls",
    format: "compact",
    align: "right",
    sort: "calls",
  },
  {
    key: "spend",
    header: "Spend 30d",
    primary: "spend",
    format: "usd",
    align: "right",
    sort: "spend",
  },
  {
    key: "health",
    header: "Health",
    primary: "health",
    tone: "healthTone",
    meter: "health",
    meterMax: 100,
    align: "right",
    sort: "health",
  },
];

const rows: ModelRow[] = models.map((model) => ({
  id: model.id,
  name: model.displayName,
  provider: providerName.get(model.providerId) ?? "Unknown",
  modality: model.modality,
  modalityTone: MODALITY_TONE[model.modality],
  status: model.status,
  statusTone: MODEL_STATUS_TONE[model.status],
  context: model.contextWindowTokens,
  maxOutput: model.maxOutputTokens,
  input: model.pricing.inputPerMTok,
  output: model.pricing.outputPerMTok,
  calls: model.callCount30d,
  spend: model.spend30d,
  health: model.healthScore,
  healthTone: healthTone(model.healthScore),
}));

const spendByModel = [...models]
  .sort((a, b) => b.spend30d - a.spend30d)
  .map((model) => ({ label: model.displayName, value: model.spend30d }));

const ga = models.filter((model) => model.status === "ga").length;
const preview = models.filter((model) => model.status === "preview").length;
const retiring = models.filter(
  (model) => model.status === "deprecated" || model.status === "retired",
).length;
const totalSpend = models.reduce((acc, model) => acc + model.spend30d, 0);

export default function ModelsPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Models"
        description="Everything the gateway can route to, with pricing, health, and the spend each one attracts."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Models available"
            value={formatNumber(models.length)}
            hint={`${formatNumber(ga)} generally available`}
          />
          <StatCard
            label="In preview"
            value={formatNumber(preview)}
            tone={preview > 0 ? "info" : "neutral"}
            hint="Not recommended for production yet"
          />
          <StatCard
            label="Retiring"
            value={formatNumber(retiring)}
            tone={retiring > 0 ? "warning" : "success"}
            hint="Deprecated or already retired"
          />
          <StatCard
            label="Spend across models"
            value={formatUsd(totalSpend)}
            hint="Trailing 30 days"
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
            <CardTitle>Model catalog</CardTitle>
            <CardDescription>
              Health is a composite of quality, latency, and cost.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={modelColumns}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No models registered."
            />
          </CardContent>
        </Card>

        <p className="text-xs text-muted-foreground">
          Prices are per one million tokens. Cheapest input route is{" "}
          {formatUsdExact(
            models.reduce(
              (best, model) =>
                model.pricing.inputPerMTok < best.pricing.inputPerMTok ? model : best,
            ).pricing.inputPerMTok,
          )}
          / Mtok.
        </p>
      </PageShell>
    </div>
  );
}
