import type { Metadata } from "next";

import { AreaChart, BarChart, DonutChart, chartColor } from "@/components/charts";
import {
  DataTable,
  PageShell,
  StatCard,
  StatusBadge,
  type Column,
} from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { formatCompact, formatPercent, formatUsd } from "@/lib/format";
import {
  billingSummary,
  costSeries,
  costSummary,
  organization,
  usageSeries,
} from "@/lib/mock-data";

export const metadata: Metadata = {
  title: "Usage · Sentinel",
  description: "Metered consumption against the allowance included in the current plan.",
};

interface MeterRow {
  id: string;
  meter: string;
  detail: string;
  used: number;
  included: number;
  pct: number;
  overage: string;
  tone: "success" | "warning" | "danger";
}

const COLUMNS: readonly Column<MeterRow>[] = [
  { key: "meter", header: "Meter", primary: "meter", secondary: "detail", sort: "meter" },
  { key: "used", header: "Used", primary: "used", format: "usd", align: "right" },
  { key: "included", header: "Included", primary: "included", format: "usd", align: "right" },
  { key: "pct", header: "Consumed", primary: "pct", format: "percentRaw", align: "right", meter: "pct", meterMax: 100, sort: "pct" },
  { key: "overage", header: "Overage rate", primary: "overage", align: "right" },
];

export default function BillingUsagePage() {
  const { allowance } = billingSummary;
  const consumedPct = (allowance.used / allowance.included) * 100;
  const hardCapPct = allowance.hardCap
    ? (allowance.used / allowance.hardCap) * 100
    : null;

  const tone = (pct: number) => (pct >= 100 ? "danger" : pct >= 80 ? "warning" : "success");

  const rows: MeterRow[] = [
    {
      id: "meter_platform",
      meter: "Model usage",
      detail: "All metered completions across every provider",
      used: allowance.used,
      included: allowance.included,
      pct: Math.round(consumedPct),
      overage: `1.1× at ${formatUsd(allowance.overageRate)} per unit`,
      tone: tone(consumedPct),
    },
    {
      id: "meter_seats",
      meter: "Seats",
      detail: `Licensed members in ${organization.name}`,
      used: organization.seatsUsed,
      included: organization.seatsTotal,
      pct: Math.round((organization.seatsUsed / organization.seatsTotal) * 100),
      overage: "Billed per seat above the plan",
      tone: tone((organization.seatsUsed / organization.seatsTotal) * 100),
    },
    {
      id: "meter_requests",
      meter: "Gateway requests",
      detail: "Total requests in the trailing 24 hours",
      used: usageSeries.totals.totalCalls,
      included: 1_000_000,
      pct: Math.round((usageSeries.totals.totalCalls / 1_000_000) * 100),
      overage: "Included in the platform fee",
      tone: tone((usageSeries.totals.totalCalls / 1_000_000) * 100),
    },
  ];

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Usage"
        description="What this billing period has consumed so far, and where the allowance stands."
        eyebrow={
          <p className="text-xs text-muted-foreground">
            {billingSummary.planName} plan · billed monthly
          </p>
        }
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Allowance used"
            value={formatUsd(allowance.used)}
            tone={tone(consumedPct)}
            hint={`of ${formatUsd(allowance.included)} included`}
            delta={{
              fraction: costSummary.changePct,
              direction: costSummary.changePct > 0 ? "up" : "down",
              higherIsWorse: true,
              label: "vs last month",
            }}
          />
          <StatCard
            label="Consumed"
            value={formatPercent(consumedPct, 0)}
            tone={tone(consumedPct)}
            hint="Share of the included allowance"
          />
          <StatCard
            label="Blended cost"
            value={`${formatUsd(costSummary.blendedCostPerMTok)} / Mtok`}
            hint="Across all traffic"
          />
          <StatCard
            label="Tokens"
            value={formatCompact(usageSeries.totals.totalTokens)}
            hint="Input and output combined"
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Allowance consumption</CardTitle>
            <CardDescription>
              {formatUsd(allowance.used)} of {formatUsd(allowance.included)} included
              {allowance.hardCap
                ? ` · hard cap at ${formatUsd(allowance.hardCap)}`
                : " · no hard cap"}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <Progress
              value={Math.min(consumedPct, 100)}
              className={
                tone(consumedPct) === "danger"
                  ? "[&>div]:bg-rose-500"
                  : tone(consumedPct) === "warning"
                    ? "[&>div]:bg-amber-500"
                    : undefined
              }
            />
            <p className="text-sm text-muted-foreground">
              Overage is charged at {formatUsd(allowance.overageRate)} per unit once the
              included allowance is exhausted
              {hardCapPct !== null && `, currently at ${formatPercent(hardCapPct, 0)} of the hard cap`}.
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Meters</CardTitle>
            <CardDescription>Each independently metered dimension of the current plan.</CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={rows}
              rowKey="id"
              toolbar={
                <div className="flex flex-wrap gap-2">
                  {rows.map((row) => (
                    <StatusBadge key={row.id} tone={row.tone}>
                      {row.meter}: {formatPercent(row.pct, 0)}
                    </StatusBadge>
                  ))}
                </div>
              }
              emptyMessage="No meters configured."
            />
          </CardContent>
        </Card>

        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Daily spend and usage</CardTitle>
              <CardDescription>Trailing 60 days, with input and output tokens split.</CardDescription>
            </CardHeader>
            <CardContent>
              <AreaChart
                series={[
                  { label: "Input tokens", color: chartColor(0), data: usageSeries.inputTokens },
                  { label: "Output tokens", color: chartColor(1), data: usageSeries.outputTokens },
                ]}
                stacked
                formatValue={(value) => formatCompact(value)}
              />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Spend by environment</CardTitle>
              <CardDescription>Where the metered usage is being generated.</CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart
                data={costSeries.byEnvironment}
                centerLabel="spend"
                size={190}
                formatValue={(value) => formatUsd(value)}
              />
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Spend by project</CardTitle>
            <CardDescription>The largest contributors to this period&apos;s metered charges.</CardDescription>
          </CardHeader>
          <CardContent>
            <BarChart
              data={costSeries.byProject.slice(0, 8)}
              formatValue={(value) => formatUsd(value)}
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
