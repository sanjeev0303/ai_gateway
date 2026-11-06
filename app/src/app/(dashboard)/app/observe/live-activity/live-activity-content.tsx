"use client";

import * as React from "react";
import { useSearchParams } from "next/navigation";

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

interface Names {
  teams: Record<string, string>;
  projects: Record<string, string>;
  users: Record<string, string>;
}

const shortId = (id: string) => `${id.slice(0, 8)}…`;

const liveColumns: readonly Column<LiveRow>[] = [
  { key: "when", header: "When", primary: "when", sort: "whenTs" },
  { key: "who", header: "Sent by", primary: "who", secondary: "team", sort: "who" },
  {
    key: "route",
    header: "Route",
    primary: "model",
    secondary: "provider",
    sort: "model",
  },
  { key: "project", header: "Project", primary: "project", sort: "project" },
  {
    key: "state",
    header: "State",
    primary: "state",
    tone: "stateTone",
    sort: "state",
  },
  { key: "request", header: "Request", primary: "shortRequest", sort: "requestId" },
];

const STATUS_HINT: Record<string, string> = {
  connecting: "Connecting to the activity stream…",
  live: "Streaming from the gateway",
  reconnecting: "Stream dropped — reconnecting…",
  offline: "Stream disconnected",
};

function toRow(event: LiveActivityEvent, now: Date, names: Names): LiveRow {
  const whenTs = Date.parse(event.started_at);
  return {
    id: event.request_id,
    href: `/app/observe/live-activity/${event.request_id}`,
    when: Number.isNaN(whenTs) ? event.started_at : formatRelative(event.started_at, now),
    whenTs: Number.isNaN(whenTs) ? 0 : whenTs,
    requestId: event.request_id,
    shortRequest: shortId(event.request_id),
    who:
      event.user_name ??
      event.service_account ??
      (event.user_id ? (names.users[event.user_id] ?? shortId(event.user_id)) : "—"),
    project:
      event.project_name ??
      (event.project_id ? (names.projects[event.project_id] ?? shortId(event.project_id)) : "—"),
    team: event.team_id ? (names.teams[event.team_id] ?? shortId(event.team_id)) : "—",
    model: event.model,
    // Requests stopped before routing never reached a provider.
    provider: event.provider === "unknown" ? "Not routed" : event.provider,
    state: event.current_state,
    stateTone: toneFor(STATE_TONE, event.current_state),
    status: event.status,
    statusTone: toneFor(STATUS_TONE, event.status),
  };
}

interface LiveActivityContentProps {
  /** Server-rendered snapshot so the first paint is populated. */
  initialEvents: readonly LiveActivityEvent[];
  initialActiveUsers: number;
  names: Names;
}

/** The filter bar's "status" values, in the stream's own state vocabulary. */
const STATUS_TO_STATE: Record<string, string> = { succeeded: "completed", failed: "failed" };

export function LiveActivityContent({
  initialEvents,
  initialActiveUsers,
  names,
}: LiveActivityContentProps) {
  const searchParams = useSearchParams();
  const { events, status } = useLiveActivity({ initialEvents });

  // Recomputed as the stream updates, so relative ages stay honest.
  const rows = React.useMemo(() => {
    const now = new Date();
    const provider = searchParams.get("provider");
    const status = searchParams.get("status");
    const model = searchParams.get("model");
    const team = searchParams.get("team_id");
    const project = searchParams.get("project_id");
    const user = searchParams.get("user_id");
    return events
      .filter((event) => {
        if (provider && event.provider !== provider) return false;
        if (status && event.current_state !== (STATUS_TO_STATE[status] ?? status)) return false;
        if (model && event.model !== model) return false;
        if (team && event.team_id !== team) return false;
        if (project && event.project_id !== project) return false;
        if (user && event.user_id !== user) return false;
        return true;
      })
      .map((event) => toRow(event, now, names));
  }, [events, names, searchParams]);

  const inFlight = rows.filter((row) => row.state === "active").length;
  const failed = rows.filter((row) => row.state === "failed").length;

  return (
    <>
      <section className="grid gap-3 sm:grid-cols-3 sm:gap-4">
        <StatCard
          label="In flight"
          value={formatNumber(inFlight)}
          tone="info"
          hint="Streaming now"
        />
        <StatCard
          label="Active users"
          value={formatNumber(initialActiveUsers)}
          hint="When you opened this page"
        />
        <StatCard
          label="Failed"
          value={formatNumber(failed)}
          tone={failed > 0 ? "danger" : "success"}
          hint="Latest in this feed"
        />
      </section>

      <Card>
        <CardHeader>
          <CardTitle>Request stream</CardTitle>
          <CardDescription>
            {formatNumber(rows.length)} requests, newest first. Open one for its full details.
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
            emptyMessage="No requests match these filters."
            ActionsComponent={LiveActivityActionsCell}
          />
        </CardContent>
      </Card>
    </>
  );
}
