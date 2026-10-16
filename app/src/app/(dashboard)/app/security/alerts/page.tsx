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

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ShieldAlert, Settings2 } from "lucide-react";
import { IncidentsClient } from "./incidents-client";

async function loadAlertData() {
  const { alerts } = await getApiClient();
  const [resAlerts, resEvents, resNotifs] = await Promise.allSettled([
    alerts.list(),
    alerts.listEvents(),
    alerts.listNotifications(),
  ]);

  return {
    rules: resAlerts.status === "fulfilled" ? resAlerts.value.items ?? [] : [],
    events: resEvents.status === "fulfilled" ? resEvents.value.items ?? [] : [],
    notifications: resNotifs.status === "fulfilled" ? resNotifs.value.items ?? [] : [],
  };
}

export default async function AlertsPage() {
  const { rules, events, notifications } = await loadAlertData();
  const rows = rules.map(toRow);

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Alerts & Incident Watchdog"
        description="Active incidents, continuous anomaly triggers, and watchdog rule configurations."
      >
        <Tabs defaultValue="incidents" className="space-y-4">
          <TabsList>
            <TabsTrigger value="incidents" className="gap-2">
              <ShieldAlert className="h-4 w-4" />
              Incidents & Dispatch ({events.filter((e) => e.status !== "resolved").length})
            </TabsTrigger>
            <TabsTrigger value="rules" className="gap-2">
              <Settings2 className="h-4 w-4" />
              Configured Rules ({rows.length})
            </TabsTrigger>
          </TabsList>

          <TabsContent value="incidents">
            <IncidentsClient
              initialEvents={events}
              initialNotifications={notifications}
            />
          </TabsContent>

          <TabsContent value="rules">
            <Card>
              <CardHeader>
                <CardTitle>Configured rules</CardTitle>
                <CardDescription>
                  Configured watchdog rules evaluated continuously by the Sentinel background engine against real gateway usage, spend, errors, and provider metrics.
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
          </TabsContent>
        </Tabs>
      </PageShell>
    </div>
  );
}