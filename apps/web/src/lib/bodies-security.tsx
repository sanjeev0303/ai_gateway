import { SpecView } from "@/lib/spec";

export function AuditLogsBody() {
  return (
    <SpecView
      blurb="Dense, append-only audit trail. Every row is one state-changing action against the control plane or gateway."
      needs={["GET /v1/audit/events"]}
      groups={[
        {
          title: "Columns",
          items: [
            "Timestamp",
            "Actor",
            "Action",
            "Resource",
            "Project",
            "Result",
            "Request ID",
          ],
        },
        { title: "Filters", items: ["actor", "action", "resource", "date", "severity"] },
        { title: "Event details drawer", items: ["full payload", "before/after diff"] },
      ]}
      note="Audit retention is not implemented, so this view must not imply that older events are retained. Severity is a property of the alert rules, not of the audit record itself."
    />
  );
}

export function AlertsBody() {
  return (
    <SpecView
      blurb="Alert rules and their lifecycle. A rule is muted to silence it, triggered when a threshold is crossed, and resolved when the condition clears."
      needs={["GET /v1/alerts", "GET /v1/alert-rules"]}
      groups={[
        { title: "States", items: ["active", "triggered", "resolved", "muted"] },
        {
          title: "Per rule",
          items: [
            "alert rule",
            "severity",
            "threshold",
            "last triggered",
            "notifications",
          ],
        },
      ]}
      note="Rate-limit and budget checks already run in the gateway hot path, but nothing yet raises a durable alert from them, so no alert history can be shown."
    />
  );
}

export function SecurityEventsBody() {
  return (
    <SpecView
      blurb="Security-relevant signals from the request path: policy denials, budget and rate rejections, and credential anomalies."
      needs={["GET /v1/security/events"]}
      groups={[
        {
          title: "Event types",
          items: [
            "policy denied",
            "rate limited",
            "budget exceeded",
            "credential revoked mid-flight",
            "provider authentication failure",
          ],
        },
        { title: "Columns", items: ["Timestamp", "Actor", "Project", "Credential", "Decision", "Severity"] },
      ]}
      note="This view is deliberately separate from Audit logs: audit records administrative intent, while security events record what the hot path refused."
    />
  );
}
