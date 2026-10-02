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
import { formatNumber, formatPercent, formatUsd } from "@/lib/format";
import { budgets } from "@/lib/mock-data";
import {
  BUDGET_ACTION_TONE,
  budgetUtilization,
  type BudgetScope,
  type StatusTone,
} from "@/types";

export const metadata: Metadata = {
  title: "Budgets · Sentinel",
  description: "Spend caps by organization, team, and project, with what happens when one is reached.",
};

const SCOPE_TONE: Record<BudgetScope, StatusTone> = {
  organization: "info",
  team: "neutral",
  project: "neutral",
};

function utilizationTone(budget: (typeof budgets)[number]): StatusTone {
  if (!budget.enabled) return "neutral";
  const used = budgetUtilization(budget);
  if (used >= 1) return "danger";
  if (used >= budget.alertThreshold / budget.limit) return "warning";
  return "success";
}

interface BudgetRow {
  id: string;
  name: string;
  target: string;
  scope: string;
  scopeTone: StatusTone;
  period: string;
  spent: string;
  spentValue: number;
  limit: string;
  limitValue: number;
  used: string;
  usedFraction: number;
  meterTone: StatusTone;
  action: string;
  actionTone: StatusTone;
  state: string;
  stateTone: StatusTone;
}

const budgetColumns: readonly Column<BudgetRow>[] = [
  { key: "name", header: "Budget", primary: "name", secondary: "target", sort: "name" },
  { key: "scope", header: "Scope", primary: "scope", tone: "scopeTone", sort: "scope" },
  { key: "period", header: "Period", primary: "period", sort: "period" },
  { key: "spent", header: "Spent", primary: "spent", format: "usd", align: "right", sort: "spentValue" },
  { key: "limit", header: "Limit", primary: "limit", align: "right", sort: "limitValue" },
  {
    key: "used",
    header: "Used",
    primary: "used",
    tone: "meterTone",
    meter: "usedFraction",
    meterMax: 1,
    align: "right",
    sort: "usedFraction",
  },
  { key: "action", header: "On breach", primary: "action", tone: "actionTone", sort: "action" },
  { key: "state", header: "State", primary: "state", tone: "stateTone", sort: "state" },
];

const rows: BudgetRow[] = budgets.map((budget) => {
  const used = budgetUtilization(budget);
  return {
    id: budget.id,
    name: budget.name,
    target: budget.targetName,
    scope: budget.scope,
    scopeTone: SCOPE_TONE[budget.scope],
    period: budget.period,
    spent: formatUsd(budget.spent),
    spentValue: budget.spent,
    limit: formatUsd(budget.limit),
    limitValue: budget.limit,
    used: formatPercent(used),
    usedFraction: used,
    meterTone: utilizationTone(budget),
    action: budget.action,
    actionTone: BUDGET_ACTION_TONE[budget.action],
    state: budget.enabled ? "Active" : "Paused",
    stateTone: budget.enabled ? "success" : "neutral",
  };
});

const totalLimit = budgets.reduce((acc, budget) => acc + budget.limit, 0);
const totalSpent = budgets.reduce((acc, budget) => acc + budget.spent, 0);
const enabled = budgets.filter((budget) => budget.enabled);
const exhausted = budgets.filter(
  (budget) => budget.enabled && budget.spent >= budget.limit,
);
const alerting = budgets.filter(
  (budget) =>
    budget.enabled &&
    budget.spent < budget.limit &&
    budget.spent >= budget.alertThreshold,
);

const spendVsLimit = [...budgets]
  .sort((a, b) => b.limit - a.limit)
  .map((budget) => ({ label: budget.name, value: budget.spent }));

const actionMix = (["hard_stop", "throttle", "alert"] as const).map((action) => ({
  label: action.replace("_", " "),
  value: budgets.filter((budget) => budget.action === action).length,
}));

export default function BudgetsPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Budgets"
        description="Caps are evaluated per period. The breach action decides whether a project gets a warning, a slowdown, or a hard refusal."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Committed spend"
            value={formatUsd(totalSpent)}
            hint={`${formatPercent(totalLimit > 0 ? totalSpent / totalLimit : 0)} of caps`}
          />
          <StatCard
            label="Total caps"
            value={formatUsd(totalLimit)}
            hint={`${formatNumber(budgets.length)} budgets`}
          />
          <StatCard
            label="At the cap"
            value={formatNumber(exhausted.length)}
            tone={exhausted.length > 0 ? "danger" : "success"}
            hint="Breach action has fired"
          />
          <StatCard
            label="Approaching"
            value={formatNumber(alerting.length)}
            tone={alerting.length > 0 ? "warning" : "neutral"}
            hint="Past the alert threshold"
          />
        </section>

        <section className="grid gap-4 lg:grid-cols-3">
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>Spend against caps</CardTitle>
              <CardDescription>
                A bar past its cap means the breach action is now the thing
                deciding whether traffic flows.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart data={spendVsLimit} formatValue={formatUsd} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Breach actions</CardTitle>
              <CardDescription>
                Hard stops are the only ones that refuse a request outright.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart
                data={actionMix}
                centerLabel={formatNumber(enabled.length)}
                formatValue={(value) => `${formatNumber(value)} budgets`}
              />
            </CardContent>
          </Card>
        </section>

        <Card>
          <CardHeader>
            <CardTitle>All budgets</CardTitle>
            <CardDescription>
              Scopes stack: a project budget is checked alongside its team
              budget, and the tighter one wins.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={budgetColumns}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No budgets configured."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
