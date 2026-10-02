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
import { formatNumber, formatRelative } from "@/lib/format";
import { NOW, alerts } from "@/lib/mock-data";
import {
  ALERT_SEVERITY_TONE,
  type AlertCategory,
  type AlertState,
  type StatusTone,
} from "@/types";

export const metadata: Metadata = {
  title: "Alerts · Sentinel",
  description: "Open, acknowledged, and resolved alerts across cost, policy, provider, and credential signals.",
};

const CATEGORY_TONE: Record<AlertCategory, StatusTone> = {
  cost_anomaly: "warning",
  policy_violation: "danger",
  provider_outage: "danger",
  error_rate: "warning",
  credential_expiry: "info",
  budget_exceeded: "warning",
  unusual_access: "danger",
};

const STATE_TONE: Record<AlertState, StatusTone> = {
  open: "danger",
  acknowledged: "warning",
  resolved: "success",
};

interface AlertRow {
  id: string;
  title: string;
  description: string;
  severity: string;
  severityTone: StatusTone;
  category: string;
  categoryTone: StatusTone;
  state: string;
  stateTone: StatusTone;
  notified: string;
  notifiedCount: number;
  when: string;
  ageMinutes: number;
}

const alertColumns: readonly Column<AlertRow>[] = [
  { key: "title", header: "Alert", primary: "title", secondary: "description", sort: "title" },
  { key: "severity", header: "Severity", primary: "severity", tone: "severityTone", sort: "severity" },
  { key: "category", header: "Category", primary: "category", tone: "categoryTone", sort: "category" },
  { key: "state", header: "State", primary: "state", tone: "stateTone", sort: "state" },
  { key: "notified", header: "Notified", primary: "notified", align: "right", sort: "notifiedCount" },
  { key: "when", header: "Raised", primary: "when", sort: "ageMinutes" },
];

const rows: AlertRow[] = alerts.map((alert) => ({
  id: alert.id,
  title: alert.title,
  description: alert.description,
  severity: alert.severity,
  severityTone: ALERT_SEVERITY_TONE[alert.severity],
  category: alert.category,
  categoryTone: CATEGORY_TONE[alert.category],
  state: alert.state,
  stateTone: STATE_TONE[alert.state],
  notified: formatNumber(alert.notifiedUserIds.length),
  notifiedCount: alert.notifiedUserIds.length,
  when: formatRelative(alert.occurredAt, NOW),
  ageMinutes: Math.round((NOW.getTime() - new Date(alert.occurredAt).getTime()) / 60_000),
}));

const open = alerts.filter((alert) => alert.state === "open");
const acknowledged = alerts.filter((alert) => alert.state === "acknowledged");
const resolved = alerts.filter((alert) => alert.state === "resolved");
const critical = open.filter(
  (alert) => alert.severity === "critical" || alert.severity === "high",
);

const categoryCounts = [...alerts.reduce((acc, alert) => {
  acc.set(alert.category, (acc.get(alert.category) ?? 0) + 1);
  return acc;
}, new Map<AlertCategory, number>())]
  .map(([category, count]) => ({ label: category.replace(/_/g, " "), value: count }))
  .sort((a, b) => b.value - a.value);

const stateMix = (["open", "acknowledged", "resolved"] as const).map((state) => ({
  label: state,
  value: alerts.filter((alert) => alert.state === state).length,
}));

export default function AlertsPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Alerts"
        description="An acknowledged alert is still yours until it is resolved. Acknowledging is how you stop the pager, not how you close the issue."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Open"
            value={formatNumber(open.length)}
            tone={open.length > 0 ? "danger" : "success"}
            hint="Unacknowledged"
          />
          <StatCard
            label="Critical or high"
            value={formatNumber(critical.length)}
            tone={critical.length > 0 ? "danger" : "success"}
            hint="Still open"
          />
          <StatCard
            label="Acknowledged"
            value={formatNumber(acknowledged.length)}
            tone={acknowledged.length > 0 ? "warning" : "neutral"}
            hint="Owned, not closed"
          />
          <StatCard
            label="Resolved"
            value={formatNumber(resolved.length)}
            tone="success"
            hint="In the window"
          />
        </section>

        {critical.length > 0 ? (
          <Card>
            <CardHeader>
              <CardTitle>Needs attention</CardTitle>
              <CardDescription>
                Open alerts at critical or high severity. These are the ones that
                justify waking someone up.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              {critical.map((alert) => (
                <div
                  key={alert.id}
                  className="flex flex-col gap-2 rounded-lg border border-border p-3 sm:flex-row sm:items-start sm:justify-between"
                >
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-foreground">{alert.title}</p>
                    <p className="text-xs text-muted-foreground">{alert.description}</p>
                  </div>
                  <p className="shrink-0 text-xs text-muted-foreground">
                    {formatRelative(alert.occurredAt, NOW)}
                  </p>
                </div>
              ))}
            </CardContent>
          </Card>
        ) : null}

        <section className="grid gap-4 lg:grid-cols-3">
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>Alerts by category</CardTitle>
              <CardDescription>
                Cost and policy signals dominate in a well-run workspace.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart data={categoryCounts} formatValue={(value) => formatNumber(value)} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Lifecycle</CardTitle>
              <CardDescription>
                A large acknowledged pile means triage is lagging behind the
                pager.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart
                data={stateMix}
                centerLabel={formatNumber(alerts.length)}
                formatValue={(value) => `${formatNumber(value)} alerts`}
              />
            </CardContent>
          </Card>
        </section>

        <Card>
          <CardHeader>
            <CardTitle>All alerts</CardTitle>
            <CardDescription>
              Each row links to the page where the underlying signal is visible.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={alertColumns}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No alerts in the retention window."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
