import { SpecView } from "@/lib/spec";

import { useEffect, useState } from "react";
import type { ActivityEvent } from "@/lib/control-api";
import { listActivity, subscribeToActivity } from "@/lib/control-api";

export function LiveActivityBody() {
  const [events, setEvents] = useState<ActivityEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    let mounted = true;
    listActivity()
      .then((res) => {
        if (mounted) {
          setEvents(res.data || []);
          setLoading(false);
        }
      })
      .catch((err) => {
        if (mounted) {
          setError(err);
          setLoading(false);
        }
      });

    const unsubscribe = subscribeToActivity(
      (event) => {
        if (!mounted) return;
        setEvents((prev) => {
          if (event.current_state === "completed" || event.current_state === "failed") {
            return prev.filter((e) => e.request_id !== event.request_id);
          }
          const idx = prev.findIndex((e) => e.request_id === event.request_id);
          if (idx >= 0) {
            const next = [...prev];
            next[idx] = event;
            return next;
          }
          return [event, ...prev];
        });
      },
      (err) => console.error("SSE Error:", err)
    );

    return () => {
      mounted = false;
      unsubscribe();
    };
  }, []);

  if (loading) return <div className="p-8 text-center text-muted-foreground">Loading activity...</div>;
  if (error) return <div className="p-8 text-destructive">{error.message}</div>;

  return (
    <div className="flex flex-col gap-6">
      <div className="grid gap-6 md:grid-cols-4">
        <div className="border border-line rounded-lg p-4 bg-surface shadow-sm">
          <h3 className="text-sm font-medium text-muted-foreground mb-1">Active Requests</h3>
          <p className="text-3xl font-bold">{events.length}</p>
        </div>
        <div className="border border-line rounded-lg p-4 bg-surface shadow-sm">
          <h3 className="text-sm font-medium text-muted-foreground mb-1">Active Users</h3>
          <p className="text-3xl font-bold">
            {new Set(events.map((e) => e.user_id).filter(Boolean)).size}
          </p>
        </div>
        <div className="border border-line rounded-lg p-4 bg-surface shadow-sm">
          <h3 className="text-sm font-medium text-muted-foreground mb-1">Current Models</h3>
          <p className="text-3xl font-bold">
            {new Set(events.map((e) => e.model).filter(Boolean)).size}
          </p>
        </div>
        <div className="border border-line rounded-lg p-4 bg-surface shadow-sm">
          <h3 className="text-sm font-medium text-muted-foreground mb-1">Current Projects</h3>
          <p className="text-3xl font-bold">
            {new Set(events.map((e) => e.project_id).filter(Boolean)).size}
          </p>
        </div>
      </div>

      <div className="border border-line rounded-lg overflow-hidden">
        {events.length === 0 ? (
          <div className="p-12 text-center text-muted-foreground">
            No active requests. The gateway is currently idle.
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead className="bg-muted/50 text-left">
              <tr>
                <th className="p-3 border-b border-line font-medium">Request ID</th>
                <th className="p-3 border-b border-line font-medium">User</th>
                <th className="p-3 border-b border-line font-medium">Provider</th>
                <th className="p-3 border-b border-line font-medium">Model</th>
                <th className="p-3 border-b border-line font-medium">Started</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-line">
              {events.map((event) => (
                <tr key={event.request_id} className="hover:bg-muted/30 transition-colors">
                  <td className="p-3 font-mono text-xs">{event.request_id.split("-")[0]}...</td>
                  <td className="p-3">{event.user_name || event.user_id || "Unknown"}</td>
                  <td className="p-3">
                    <span className="inline-flex items-center px-2 py-0.5 rounded-full bg-primary/10 text-primary text-xs font-medium">
                      {event.provider}
                    </span>
                  </td>
                  <td className="p-3">{event.model}</td>
                  <td className="p-3 tabular-nums text-muted-foreground">
                    {new Date(event.started_at).toLocaleTimeString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}

export function UsageBody() {
  return (
    <SpecView
      blurb="Advanced usage filtering across every dimension the request log carries, with the full token breakdown the provider reported."
      needs={["GET /v1/usage/aggregates", "GET /v1/usage/daily"]}
      groups={[
        {
          title: "Dimensions",
          items: [
            "time",
            "provider",
            "model",
            "user",
            "team",
            "project",
            "service account",
            "credential",
          ],
        },
        {
          title: "Metrics",
          items: [
            "requests",
            "input tokens",
            "output tokens",
            "cached tokens",
            "thinking tokens",
            "tool tokens",
            "total tokens",
            "cost",
            "latency",
            "TTFT",
            "errors",
          ],
        },
        { title: "Views", items: ["chart", "table", "CSV export"] },
      ]}
      note="CSV export is offered only once the backend serves a safe export endpoint. It is never assembled in the browser from the current page of rows, because that would silently truncate the file."
    />
  );
}

export function CostsBody() {
  return (
    <SpecView
      blurb="Derived spend, broken out by every cost dimension, with the composition of what makes up the number."
      needs={["GET /v1/usage/cost", "GET /v1/price-sheets"]}
      groups={[
        { title: "Totals", items: ["total spend", "spend trend"] },
        {
          title: "Breakdowns",
          items: [
            "provider spend",
            "model spend",
            "team spend",
            "project spend",
            "user spend",
          ],
        },
        { title: "Cost composition", items: ["input", "output", "cached", "thinking", "tool"] },
        { title: "Basis labels", items: ["estimated", "actual / provider-reported"] },
      ]}
      note="Every figure on this page is derived by pricing llm_requests against the price sheet effective at request time. Nothing settles an invoice yet, so each value must carry its basis label. Estimated is never presented as provider billing."
    />
  );
}

export function ConversationsBody() {
  return (
    <SpecView
      blurb="Conversation index across the organization. Rows list metadata; message content is never part of this list."
      needs={["GET /v1/conversations"]}
      groups={[
        {
          title: "Columns",
          items: [
            "user",
            "project",
            "provider",
            "model",
            "title",
            "created",
            "updated",
            "message count",
            "privacy mode",
          ],
        },
      ]}
      note="Message content is readable only when the organization privacy policy allows it, and retention may already have removed it. Conversation retention and privacy modes are not implemented, so no content endpoint is assumed to exist here."
    />
  );
}
