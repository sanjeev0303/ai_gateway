import type { Metadata } from "next";

import type { Alert, AlertSeverity, AlertStatus } from "@ai-governance/api-client";
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
import { AlertActionsMenu, CreateAlertDialog } from "./client";
import { AlertActionsCell } from "./actions-cell";

export const metadata: Metadata = {
  title: "Alerts · Sentinel",
  description: "Alert rules configured for your organization.",
};

export interface AlertRow {
  id: string;
  name: string;
  alertType: string;
  severity: string;
  severityTone: StatusTone;
  configuration: string;
  status: string;
  statusTone: StatusTone;
  lastTriggered: string;
  updatedAt: string;
  rawAlert: Alert;
}

const SEVERITY_TONE: Record<AlertSeverity, StatusTone> = {
  info: "info",
  warning: "warning",
  critical: "danger",
};

const STATUS_TONE: Record<AlertStatus, StatusTone> = {
  active: "success",
  muted: "neutral",
};

const ALERT_TYPE_LABEL: Record<Alert["alert_type"], string> = {
  spend_threshold: "Spend threshold",
  spend_exceeded: "Spend exceeded",
  request_rate_spike: "Request rate spike",
  error_rate_spike: "Error rate spike",
  provider_failure: "Provider failure",
  credential_failure: "Credential failure",
  policy_violation: "Policy violation",
  rate_limit_spike: "Rate limit spike",
  usage_anomaly: "Usage anomaly",
};

function summarizeConfiguration(alert: Alert): string {
  const keys = Object.keys(alert.configuration ?? {});
  if (keys.length === 0) return "No thresholds configured";
  return keys.map((key) => `${key}: ${JSON.stringify(alert.configuration[key])}`).join(" · ");
}

function toRow(alert: Alert): AlertRow {
  return {
    id: alert.id,
    name: alert.name,
    alertType: ALERT_TYPE_LABEL[alert.alert_type],
    severity: alert.severity,
    severityTone: SEVERITY_TONE[alert.severity],
    configuration: summarizeConfiguration(alert),
    status: alert.status,
    statusTone: STATUS_TONE[alert.status],
    lastTriggered: alert.last_triggered_at ?? "Never",
    updatedAt: alert.updated_at,
    rawAlert: alert, // Include raw alert for the actions menu
  };
}

const columns: readonly Column<AlertRow>[] = [
  { key: "name", header: "Rule", primary: "name", secondary: "alertType", sort: "name" },
  {
    key: "severity",
    header: "Severity",
    primary: "severity",
    tone: "severityTone",
    sort: "severity",
  },
  { key: "configuration", header: "Configuration", primary: "configuration", sort: "configuration" },
  { key: "status", header: "Status", primary: "status", tone: "statusTone", sort: "status" },
  { key: "lastTriggered", header: "Last triggered", primary: "lastTriggered", sort: "lastTriggered" },
  { key: "updatedAt", header: "Updated", primary: "updatedAt", format: "relative", align: "right", sort: "updatedAt" },
];

async function loadAlerts() {
  const { alerts } = await getApiClient();
  return alerts.list();
}

export default async function AlertsPage() {
  const { items } = await loadAlerts();
  const rows = items.map(toRow);

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Alerts"
        description="Alert rules configured for your organization."
      >
        <Card>
          <CardHeader>
            <CardTitle>Configured rules</CardTitle>
            <CardDescription>
              These rules only store configuration. No evaluator is wired to any rule type, so
              nothing is ever triggered and no notification is ever sent — &ldquo;Last
              triggered&rdquo; stays empty by design, not by failure.
            </CardDescription>
          </CardHeader>
          <CardContent>
            {rows.length === 0 ? (
              <EmptyState
                title="No alert rules configured"
                description="No alert rules exist yet."
              />
            ) : (
              <DataTable
                columns={columns}
                rows={rows}
                rowKey="id"
                pageSize={25}
                emptyMessage="No alert rules configured"
                ActionsComponent={AlertActionsCell}
                toolbar={
                  <div className="flex w-full items-center justify-between gap-4">
                    <p className="text-muted-foreground text-sm">
                      {rows.length} rule{rows.length === 1 ? "" : "s"}
                    </p>
                    <CreateAlertDialog />
                  </div>
                }
              />
            )}
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}