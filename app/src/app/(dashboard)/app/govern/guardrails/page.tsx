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
import { formatCompact, formatNumber, formatPercent } from "@/lib/format";
import { guardrails } from "@/lib/mock-data";
import {
  GUARDRAIL_ACTION_TONE,
  type GuardrailLayer,
  type StatusTone,
} from "@/types";

export const metadata: Metadata = {
  title: "Guardrails · Sentinel",
  description: "Input and output detectors, their thresholds, and how often each one actually intervenes.",
};

const LAYER_TONE: Record<GuardrailLayer, StatusTone> = {
  input: "info",
  output: "warning",
};

interface GuardrailRow {
  id: string;
  name: string;
  description: string;
  layer: string;
  layerTone: StatusTone;
  action: string;
  actionTone: StatusTone;
  threshold: string;
  thresholdValue: number;
  triggers: string;
  triggerCount: number;
  interventions: string;
  interventionCount: number;
  precision: string;
  precisionFraction: number;
  meterTone: StatusTone;
  state: string;
  stateTone: StatusTone;
}

const guardrailColumns: readonly Column<GuardrailRow>[] = [
  { key: "name", header: "Guardrail", primary: "name", secondary: "description", sort: "name" },
  { key: "layer", header: "Layer", primary: "layer", tone: "layerTone", sort: "layer" },
  { key: "action", header: "Action", primary: "action", tone: "actionTone", sort: "action" },
  {
    key: "threshold",
    header: "Threshold",
    primary: "threshold",
    align: "right",
    sort: "thresholdValue",
  },
  {
    key: "triggers",
    header: "Triggers 24h",
    primary: "triggers",
    align: "right",
    sort: "triggerCount",
  },
  {
    key: "interventions",
    header: "Interventions 24h",
    primary: "interventions",
    align: "right",
    sort: "interventionCount",
  },
  {
    key: "precision",
    header: "Intervened",
    primary: "precision",
    tone: "meterTone",
    meter: "precisionFraction",
    meterMax: 1,
    align: "right",
    sort: "precisionFraction",
  },
  { key: "state", header: "State", primary: "state", tone: "stateTone", sort: "state" },
];

const rows: GuardrailRow[] = guardrails.map((guardrail) => {
  const fraction =
    guardrail.triggers24h > 0 ? guardrail.interventions24h / guardrail.triggers24h : 0;
  return {
    id: guardrail.id,
    name: guardrail.name,
    description: guardrail.description,
    layer: guardrail.layer,
    layerTone: LAYER_TONE[guardrail.layer],
    action: guardrail.action,
    actionTone: GUARDRAIL_ACTION_TONE[guardrail.action],
    threshold: formatPercent(guardrail.threshold, 2),
    thresholdValue: guardrail.threshold,
    triggers: formatNumber(guardrail.triggers24h),
    triggerCount: guardrail.triggers24h,
    interventions: formatNumber(guardrail.interventions24h),
    interventionCount: guardrail.interventions24h,
    precision: formatPercent(fraction),
    precisionFraction: fraction,
    meterTone:
      fraction >= 0.25 ? "danger" : fraction > 0 ? "warning" : "neutral",
    state: guardrail.enabled ? "Enabled" : "Disabled",
    stateTone: guardrail.enabled ? "success" : "neutral",
  };
});

const enabled = guardrails.filter((guardrail) => guardrail.enabled);
const totalTriggers = guardrails.reduce((acc, g) => acc + g.triggers24h, 0);
const totalInterventions = guardrails.reduce(
  (acc, g) => acc + g.interventions24h,
  0,
);
const blocking = guardrails.filter((guardrail) => guardrail.action === "block");
const silent = guardrails.filter(
  (guardrail) => guardrail.triggers24h > 0 && guardrail.interventions24h === 0,
);

const triggerVolume = [...guardrails]
  .sort((a, b) => b.triggers24h - a.triggers24h)
  .slice(0, 8)
  .map((guardrail) => ({ label: guardrail.name, value: guardrail.triggers24h }));

export default function GuardrailsPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Guardrails"
        description="Detectors run inline on the request or the response. A guardrail that triggers constantly but never intervenes is mistuned, not reassuring."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Guardrails"
            value={formatNumber(guardrails.length)}
            hint={`${formatNumber(enabled.length)} enabled`}
          />
          <StatCard
            label="Triggers 24h"
            value={formatCompact(totalTriggers)}
            hint="Detections fired"
          />
          <StatCard
            label="Interventions 24h"
            value={formatNumber(totalInterventions)}
            tone={totalInterventions > 0 ? "warning" : "neutral"}
            hint={totalTriggers > 0 ? `${formatPercent(totalInterventions / totalTriggers)} of triggers` : "No triggers"}
          />
          <StatCard
            label="Blocking"
            value={formatNumber(blocking.length)}
            tone={blocking.length > 0 ? "danger" : "neutral"}
            hint="Refuse the request outright"
          />
        </section>

        {silent.length > 0 ? (
          <Card>
            <CardHeader>
              <CardTitle>Triggering but not intervening</CardTitle>
              <CardDescription>
                These fired on real traffic and changed nothing. Either the
                threshold is too loose or the action is too weak.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-2">
              {silent.map((guardrail) => (
                <div
                  key={guardrail.id}
                  className="flex items-center justify-between gap-3 rounded-lg border border-border px-3 py-2"
                >
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium text-foreground">
                      {guardrail.name}
                    </p>
                    <p className="truncate text-xs text-muted-foreground">
                      {guardrail.description}
                    </p>
                  </div>
                  <p className="shrink-0 text-xs text-muted-foreground">
                    {formatNumber(guardrail.triggers24h)} triggers
                  </p>
                </div>
              ))}
            </CardContent>
          </Card>
        ) : null}

        <Card>
          <CardHeader>
            <CardTitle>Trigger volume</CardTitle>
            <CardDescription>
              Input-layer detectors see every request; output-layer detectors
              only see what survives them.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <BarChart
              data={triggerVolume}
              formatValue={(value) => formatCompact(value)}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>All guardrails</CardTitle>
            <CardDescription>
              Raising a threshold trades recall for precision, so the intervened
              column is the one to watch after a change.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={guardrailColumns}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No guardrails configured."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
