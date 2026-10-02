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
import { NOW, auditLogs } from "@/lib/mock-data";
import {
  AUDIT_ACTION_TONE,
  type AuditAction,
  type AuditActorType,
  type StatusTone,
} from "@/types";

export const metadata: Metadata = {
  title: "Audit logs · Sentinel",
  description: "Append-only record of every actor, action, and resource touched in the console.",
};

const ACTOR_TONE: Record<AuditActorType, StatusTone> = {
  user: "info",
  api_key: "neutral",
  system: "success",
};

const RISKY_ACTIONS: readonly AuditAction[] = [
  "delete",
  "export",
  "policy_change",
  "permission_change",
];

interface AuditRow {
  id: string;
  when: string;
  actor: string;
  actorType: string;
  actorTypeTone: StatusTone;
  email: string;
  action: string;
  actionTone: StatusTone;
  resource: string;
  details: string;
  origin: string;
  country: string;
  mfa: string;
  mfaTone: StatusTone;
}

const auditColumns: readonly Column<AuditRow>[] = [
  { key: "actor", header: "Actor", primary: "actor", secondary: "email", sort: "actor" },
  { key: "actorType", header: "Type", primary: "actorType", tone: "actorTypeTone", sort: "actorType" },
  { key: "action", header: "Action", primary: "action", tone: "actionTone", sort: "action" },
  { key: "resource", header: "Resource", primary: "resource", sort: "resource" },
  { key: "details", header: "Details", primary: "details", sort: "details" },
  { key: "origin", header: "Origin", primary: "origin", secondary: "country", sort: "origin" },
  { key: "mfa", header: "MFA", primary: "mfa", tone: "mfaTone", sort: "mfa" },
  { key: "when", header: "When", primary: "when", sort: "when" },
];

const rows: AuditRow[] = [...auditLogs]
  .sort((a, b) => (a.occurredAt < b.occurredAt ? 1 : -1))
  .map((entry) => ({
    id: entry.id,
    when: formatRelative(entry.occurredAt, NOW),
    actor: entry.actorName,
    actorType: entry.actorType,
    actorTypeTone: ACTOR_TONE[entry.actorType],
    email: entry.actorEmail ?? "—",
    action: entry.action,
    actionTone: AUDIT_ACTION_TONE[entry.action],
    resource: entry.resource,
    details: entry.details,
    origin: entry.ipAddress,
    country: entry.ipCountry,
    mfa: entry.mfaVerified ? "Verified" : "Not required",
    mfaTone: entry.mfaVerified ? "success" : "neutral",
  }));

const risky = auditLogs.filter((entry) => RISKY_ACTIONS.includes(entry.action));
const byApiKey = auditLogs.filter((entry) => entry.actorType === "api_key");
const uniqueActors = new Set(auditLogs.map((entry) => entry.actorId)).size;
const foreignOrigins = new Set(auditLogs.map((entry) => entry.ipCountry)).size;

const actionVolume = [...auditLogs.reduce((acc, entry) => {
  acc.set(entry.action, (acc.get(entry.action) ?? 0) + 1);
  return acc;
}, new Map<AuditAction, number>())]
  .map(([action, count]) => ({ label: action.replace("_", " "), value: count }))
  .sort((a, b) => b.value - a.value)
  .slice(0, 8);

const actorMix = (["user", "api_key", "system"] as const).map((actorType) => ({
  label: actorType.replace("_", " "),
  value: auditLogs.filter((entry) => entry.actorType === actorType).length,
}));

export default function AuditLogsPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Audit logs"
        description="Append-only. Entries cannot be edited or deleted, so a gap here means a retention problem rather than a missing row."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Entries"
            value={formatNumber(auditLogs.length)}
            hint="Retention window"
          />
          <StatCard
            label="Distinct actors"
            value={formatNumber(uniqueActors)}
            hint="Humans, keys, and services"
          />
          <StatCard
            label="Privileged actions"
            value={formatNumber(risky.length)}
            tone={risky.length > 0 ? "warning" : "neutral"}
            hint="Exports, deletes, policy edits"
          />
          <StatCard
            label="Origin countries"
            value={formatNumber(foreignOrigins)}
            tone={foreignOrigins > 1 ? "warning" : "success"}
            hint="A second country is worth a look"
          />
        </section>

        <section className="grid gap-4 lg:grid-cols-3">
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>Action volume</CardTitle>
              <CardDescription>
                A spike in exports or permission changes is the signature of a
                review that went sideways.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart data={actionVolume} formatValue={(value) => formatNumber(value)} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Actor types</CardTitle>
              <CardDescription>
                {formatNumber(byApiKey.length)} entries came from API keys
                rather than a signed-in person.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart
                data={actorMix}
                centerLabel={formatNumber(auditLogs.length)}
                formatValue={(value) => `${formatNumber(value)} entries`}
              />
            </CardContent>
          </Card>
        </section>

        <Card>
          <CardHeader>
            <CardTitle>All entries</CardTitle>
            <CardDescription>
              Newest first. Timestamps are shown relative to the snapshot clock,
              with the absolute value available on hover of the row detail.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={auditColumns}
              rows={rows}
              rowKey="id"
              pageSize={15}
              emptyMessage="No audit entries in the retention window."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
