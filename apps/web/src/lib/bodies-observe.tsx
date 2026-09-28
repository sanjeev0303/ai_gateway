import { SpecView } from "@/lib/spec";

export function LiveActivityBody() {
  return (
    <SpecView
      blurb="Recent gateway activity, streamed over SSE. Rows appear when a request starts and settle when it completes, fails, or is blocked by policy."
      needs={["GET /v1/requests/stream (SSE)", "GET /v1/requests"]}
      groups={[
        {
          title: "Request states",
          items: ["active", "streaming", "completed", "failed", "rate-limited", "blocked"],
        },
        {
          title: "Filters",
          items: ["user", "team", "project", "provider", "model", "status"],
        },
        {
          title: "Request inspector",
          items: [
            "request ID",
            "provider request ID",
            "user",
            "credential",
            "project",
            "provider",
            "model",
            "TTFT",
            "total latency",
            "usage",
            "cost",
            "policy decision",
          ],
        },
      ]}
      note="Privacy mode is not implemented, so no prompt or completion content is rendered here or in the request inspector. Only metadata is in scope for this view."
    />
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
