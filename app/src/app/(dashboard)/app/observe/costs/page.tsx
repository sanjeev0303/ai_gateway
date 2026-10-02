import type { Metadata } from "next";

import { BarChart, DonutChart, LineChart, chartColor } from "@/components/charts";
import {
  PageShell,
  StatCard,
} from "@/components/dashboard";
import { Progress } from "@/components/ui/progress";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatPercent, formatUsd, formatUsdExact } from "@/lib/format";
import { costSeries, costSummary } from "@/lib/mock-data";
import type { StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Costs · Sentinel",
  description: "Spend to date, forecast, and attribution by model, team, and project.",
};

export default function CostsPage() {
  const { forecast } = costSummary;
  const budgetTone: StatusTone =
    forecast.budget !== null && forecast.actual >= forecast.budget
      ? "danger"
      : forecast.projectedOverspend > 0
        ? "warning"
        : "success";
  const budgetProgress =
    forecast.budget !== null && forecast.budget > 0
      ? Math.min(1, forecast.actual / forecast.budget)
      : 0;

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Costs"
        description="Every dollar attributed to a model, team, project, and environment, with a month-end forecast."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Spend month to date"
            value={formatUsd(costSummary.mtd)}
            hint={`${formatUsd(costSummary.lastMonth)} same period last month`}
            delta={{
              fraction: costSummary.changePct,
              direction: costSummary.changePct > 0 ? "up" : "down",
              higherIsWorse: true,
              label: "vs last month",
            }}
            sparkline={costSeries.dailySpend}
          />
          <StatCard
            label="Forecast at month end"
            value={formatUsd(forecast.projected)}
            tone={budgetTone}
            hint={
              forecast.projectedOverspend > 0
                ? `${formatUsd(forecast.projectedOverspend)} over budget`
                : "Within budget"
            }
          />
          <StatCard
            label="Blended cost per Mtok"
            value={formatUsdExact(costSummary.blendedCostPerMTok)}
            hint="Across all traffic"
          />
          <StatCard
            label="Monthly budget"
            value={forecast.budget === null ? "No cap" : formatUsd(forecast.budget)}
            tone={budgetTone}
            hint={`${formatPercent(budgetProgress)} consumed`}
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Daily spend</CardTitle>
            <CardDescription>
              The last 60 days, so the current month can be compared against the
              full prior month.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <LineChart
              label="Daily spend"
              data={costSeries.dailySpend}
              color={chartColor(0)}
              height={260}
              formatValue={formatUsd}
            />
            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Budget consumed</span>
                <span className="tabular-nums">
                  {formatUsd(forecast.actual)}
                  {forecast.budget !== null && ` of ${formatUsd(forecast.budget)}`}
                </span>
              </div>
              <Progress value={budgetProgress * 100} />
            </div>
          </CardContent>
        </Card>

        <section className="grid gap-4 lg:grid-cols-3">
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>Spend by team</CardTitle>
              <CardDescription>
                Month-to-date spend attributed to each team.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart data={costSeries.byTeam} formatValue={formatUsd} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Spend by provider</CardTitle>
              <CardDescription>Where the invoice lands.</CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart
                data={costSeries.byProvider}
                centerLabel={formatUsd(costSummary.mtd)}
                formatValue={formatUsd}
              />
            </CardContent>
          </Card>
        </section>

        <section className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Spend by project</CardTitle>
              <CardDescription>Highest first.</CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart data={costSeries.byProject} formatValue={formatUsd} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Spend by environment</CardTitle>
              <CardDescription>
                Production versus staging and development.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart data={costSeries.byEnvironment} formatValue={formatUsd} />
            </CardContent>
          </Card>
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Spend by model</CardTitle>
            <CardDescription>
              Trailing 30 days, highest first.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <BarChart data={costSeries.byModel} formatValue={formatUsd} />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
