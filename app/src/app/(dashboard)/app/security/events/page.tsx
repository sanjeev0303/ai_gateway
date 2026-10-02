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
import { NOW, platformEvents } from "@/lib/mock-data";
import type { PlatformEvent, StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Events · Sentinel",
  description: "Platform-originated events: pipeline failures, provider incidents, and control-plane changes.",
};

const SEVERITY_TONE: Record<PlatformEvent["severity"], StatusTone> = {
  info: "info",
  warning: "warning",
  error: "danger",
};

const SEVERITY_ORDER: Record<PlatformEvent["severity"], number> = {
  error: 0,
  warning: 1,
  info: 2,
};

interface EventRow {
  id: string;
  severity: string;
  severityTone: StatusTone;
  severityRank: number;
  source: string;
  message: string;
  link: string;
  when: string;
}

const eventColumns: readonly Column<EventRow>[] = [
  { key: "severity", header: "Severity", primary: "severity", tone: "severityTone", sort: "severityRank" },
  { key: "source", header: "Source", primary: "source", sort: "source" },
  { key: "message", header: "Event", primary: "message", sort: "message" },
  { key: "link", header: "Go to", primary: "link", sort: "link" },
  { key: "when", header: "When", primary: "when", sort: "when" },
];

const rows: EventRow[] = [...platformEvents]
  .sort((a, b) => {
    const bySeverity = SEVERITY_ORDER[a.severity] - SEVERITY_ORDER[b.severity];
    if (bySeverity !== 0) return bySeverity;
    return a.occurredAt < b.occurredAt ? 1 : -1;
  })
  .map((event) => ({
    id: event.id,
    severity: event.severity,
    severityTone: SEVERITY_TONE[event.severity],
    severityRank: SEVERITY_ORDER[event.severity],
    source: event.source,
    message: event.message,
    link: event.link,
    when: formatRelative(event.occurredAt, NOW),
  }));

const errors = platformEvents.filter((event) => event.severity === "error");
const warnings = platformEvents.filter((event) => event.severity === "warning");
const sources = new Set(platformEvents.map((event) => event.source));

const severityMix = (["error", "warning", "info"] as const).map((severity) => ({
  label: severity,
  value: platformEvents.filter((event) => event.severity === severity).length,
}));

const bySource = [...platformEvents.reduce((acc, event) => {
  acc.set(event.source, (acc.get(event.source) ?? 0) + 1);
  return acc;
}, new Map<string, number>())]
  .map(([source, count]) => ({ label: source, value: count }))
  .sort((a, b) => b.value - a.value);

export default function EventsPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Events"
        description="Everything the platform did on its own, kept separate from what your users did. If a metric moved and no user caused it, it starts here."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Events"
            value={formatNumber(platformEvents.length)}
            hint={`${formatNumber(sources.size)} sources`}
          />
          <StatCard
            label="Errors"
            value={formatNumber(errors.length)}
            tone={errors.length > 0 ? "danger" : "success"}
            hint="Failed operations"
          />
          <StatCard
            label="Warnings"
            value={formatNumber(warnings.length)}
            tone={warnings.length > 0 ? "warning" : "neutral"}
            hint="Degraded, not broken"
          />
          <StatCard
            label="Informational"
            value={formatNumber(platformEvents.length - errors.length - warnings.length)}
            tone="info"
            hint="Control-plane changes"
          />
        </section>

        <section className="grid gap-4 lg:grid-cols-3">
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>Events by source</CardTitle>
              <CardDescription>
                One source dominating the chart usually means a single component
                is generating the noise.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart data={bySource} formatValue={(value) => formatNumber(value)} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Severity mix</CardTitle>
              <CardDescription>
                Errors are worth a page; everything else is context.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart
                data={severityMix}
                centerLabel={formatNumber(platformEvents.length)}
                formatValue={(value) => `${formatNumber(value)} events`}
              />
            </CardContent>
          </Card>
        </section>

        <Card>
          <CardHeader>
            <CardTitle>All events</CardTitle>
            <CardDescription>
              Sorted by severity first, then newest, so failures stay at the top
              of the list.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={eventColumns}
              rows={rows}
              rowKey="id"
              pageSize={15}
              emptyMessage="No platform events in the retention window."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
