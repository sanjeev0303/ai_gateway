import type { Metadata } from "next";

import type {
  SecurityEvent,
  SecurityEventCategory,
  SecurityEventSeverity,
} from "@ai-governance/api-client";
import {
  DataTable,
  EmptyState,
  PageShell,
  type Column,
} from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Events · Sentinel",
  description: "Platform-originated events: provider health transitions and failed usage rollups.",
};

interface EventRow {
  id: string;
  title: string;
  category: string;
  severity: string;
  severityTone: StatusTone;
  source: string;
  detail: string;
  occurredAt: string;
}

const SEVERITY_TONE: Record<SecurityEventSeverity, StatusTone> = {
  info: "info",
  warning: "warning",
  critical: "danger",
};

const CATEGORY_LABEL: Record<SecurityEventCategory, string> = {
  provider_incident: "Provider incident",
  pipeline_failure: "Pipeline failure",
};

/**
 * `detail` is an open JSON object, so render the recorded context rather than
 * hiding it: the provider kind and from/to status for an incident, the failing
 * stage for a pipeline failure.
 */
function summarizeDetail(detail: SecurityEvent["detail"]): string {
  const keys = Object.keys(detail ?? {});
  if (keys.length === 0) return "No detail recorded";
  return keys.map((key) => `${key}: ${JSON.stringify(detail[key])}`).join(" · ");
}

function toRow(event: SecurityEvent): EventRow {
  return {
    id: event.id,
    title: event.title,
    category: CATEGORY_LABEL[event.category],
    severity: event.severity,
    severityTone: SEVERITY_TONE[event.severity],
    source: event.source,
    detail: summarizeDetail(event.detail),
    occurredAt: event.created_at,
  };
}

const columns: readonly Column<EventRow>[] = [
  {
    key: "title",
    header: "Event",
    primary: "title",
    secondary: "category",
    sort: "title",
  },
  {
    key: "severity",
    header: "Severity",
    primary: "severity",
    tone: "severityTone",
    sort: "severity",
  },
  {
    key: "source",
    header: "Source",
    primary: "source",
    secondary: "detail",
    sort: "source",
  },
  {
    key: "occurredAt",
    header: "When",
    primary: "occurredAt",
    format: "relative",
    align: "right",
    sort: "occurredAt",
  },
];

async function loadSecurityEvents() {
  const { securityEvents } = await getApiClient();
  return securityEvents.list({ limit: 200 });
}

/**
 * Security events are platform-originated and append-only, so this view is
 * deliberately read-only: there is no control here that could acknowledge,
 * edit, or remove an event, matching the API.
 */
export default async function EventsPage() {
  const { items } = await loadSecurityEvents();
  const rows = items.map(toRow);

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Events"
        description="Platform-originated events: provider health transitions and failed usage rollups."
      >
        <Card>
          <CardHeader>
            <CardTitle>Platform event history</CardTitle>
            <CardDescription>
              Newest first, scoped to your organization. Only two things are recorded here: a
              provider changing health status, and a usage rollup failing. Changes made by people
              in the console are audit records instead. Events can never be edited or deleted, and
              nothing here can acknowledge or mute one.
            </CardDescription>
          </CardHeader>
          <CardContent>
            {rows.length === 0 ? (
              <EmptyState
                title="No incidents recorded"
                description="Nothing has failed since events began being captured, so there is nothing to list. Provider incidents appear here when a provider's reachability changes, and pipeline failures appear when a usage rollup cannot be completed. Gateway request activity is a different thing and is live under Observe → Live activity; current provider reachability is under Govern → Provider health."
              />
            ) : (
              <DataTable
                columns={columns}
                rows={rows}
                rowKey="id"
                pageSize={25}
                emptyMessage="No incidents recorded"
                toolbar={
                  <p className="text-muted-foreground text-sm">
                    {rows.length} event{rows.length === 1 ? "" : "s"}
                  </p>
                }
              />
            )}
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}