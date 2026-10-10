"use client";

import * as React from "react";

import {
  DataTable,
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
import { toneFor } from "@/lib/org-tones";
import { useLiveActivity, type LiveActivityEvent } from "@/hooks/use-live-activity";
import { LiveActivityDrawer, type LiveRow } from "./live-activity-drawer";
import { LiveActivityActionsCell } from "./live-activity-actions-cell";
import type { StatusTone } from "@/types";

/** Mirrors domain.ActivityState: active, idle, completed, failed. */
const STATE_TONE: Record<string, StatusTone> = {
  active: "info",
  completed: "success",
  failed: "danger",
  idle: "neutral",
};

const STATUS_TONE: Record<string, StatusTone> = {
  success: "success",
  error: "danger",
  failed: "danger",
};

const liveColumns: readonly Column<LiveRow>[] = [
  { key: "when", header: "When", primary: "when", sort: "whenTs" },
  {
    key: "request",
    header: "Request",
    primary: "requestId",
    secondary: "who",
    sort: "requestId",
  },
  {
    key: "route",
    header: "Route",
    primary: "model",
    secondary: "provider",
    sort: "model",
  },
  { key: "project", header: "Project", primary: "project", sort: "project" },
  { key: "team", header: "Team", primary: "team", sort: "team" },
  {
    key: "state",
    header: "State",
    primary: "state",
    tone: "stateTone",
    sort: "state",
  },
  {
    key: "status",
    header: "Status",
    primary: "status",
    tone: "statusTone",
    sort: "status",
  },
];

const STATUS_HINT: Record<string, string> = {
  connecting: "Connecting to the activity stream…",
  live: "Streaming from the gateway",
  reconnecting: "Stream dropped — reconnecting…",
  offline: "Stream disconnected",
};

function toRow(event: LiveActivityEvent, now: Date): LiveRow {
  const whenTs = Date.parse(event.started_at);
  return {
    id: event.request_id,
    href: `/app/observe/live-activity/${event.request_id}`,
    when: Number.isNaN(whenTs) ? event.started_at : formatRelative(event.started_at, now),
    whenTs: Number.isNaN(whenTs) ? 0 : whenTs,
    requestId: event.request_id,
    who: event.user_name ?? event.service_account ?? event.user_id ?? "—",
    project: event.project_name ?? event.project_id ?? "—",
    team: event.team_id ?? "—",
    model: event.model,
    provider: event.provider,
    state: event.current_state,
    stateTone: toneFor(STATE_TONE, event.current_state),
    status: event.status,
    statusTone: toneFor(STATUS_TONE, event.status),
  };
}

interface LiveActivityContentProps {
  /** Server-rendered snapshot so the first paint is populated. */
  initialEvents: readonly LiveActivityEvent[];
  initialActiveRequests: number;
  initialActiveUsers: number;
}

export function LiveActivityContent({
  initialEvents,
  initialActiveRequests,
  initialActiveUsers,
}: LiveActivityContentProps) {
  const { events, status } = useLiveActivity({ initialEvents });

  // Recomputed as the stream updates, so relative ages stay honest.
  const rows = React.useMemo(() => {
    const now = new Date();
    return events.map((event) => toRow(event, now));
  }, [events]);

  const inFlight = rows.filter((row) => row.state === "active").length;
  const failed = rows.filter((row) => row.state === "failed").length;

  return (
    <>
      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-5">
        <StatCard
          label="Tracked requests"
          value={formatNumber(rows.length)}
          hint="In the live window"
        />
        <StatCard
          label="In flight"
          value={formatNumber(inFlight)}
          tone="info"
          hint="Streaming now"
        />
        <StatCard
          label="Active users"
          value={formatNumber(initialActiveUsers)}
          hint="Reported at page load"
        />
        <StatCard
          label="Active requests"
          value={formatNumber(initialActiveRequests)}
          hint="Reported at page load"
        />
        <StatCard
          label="Failed"
          value={formatNumber(failed)}
          tone={failed > 0 ? "danger" : "success"}
          hint="In the live window"
        />
      </section>

      <Card>
        <CardHeader>
          <CardTitle>Request stream</CardTitle>
          <CardDescription>
            {formatNumber(rows.length)} requests, newest first, with the request id for
            support escalation.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <p className="text-xs text-muted-foreground" aria-live="polite">
            {STATUS_HINT[status]}
          </p>
          <DataTable
            columns={liveColumns}
            rows={rows}
            rowKey="id"
            pageSize={12}
            emptyMessage="No requests recorded yet."
            ActionsComponent={LiveActivityActionsCell}
          />
        </CardContent>
      </Card>
    </>
  );
}
