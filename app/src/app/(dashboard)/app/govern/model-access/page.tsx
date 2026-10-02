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
import { formatCompact, formatNumber, formatUsd } from "@/lib/format";
import {
  denials24h,
  grantAdmission24h,
  GRANT_TONE,
  models,
  modelGrants,
  planEntitlements,
} from "@/lib/demo";
import type { StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Model access · Sentinel",
  description:
    "Grants, denials, and plan entitlements that decide which principal may call which model.",
};

const modelName = new Map(models.map((model) => [model.id, model.displayName]));

const displayName = (modelId: string) => modelName.get(modelId) ?? modelId;

interface GrantRow {
  id: string;
  name: string;
  models: string;
  modelCount: number;
  principals: string;
  principalCount: number;
  scope: string;
  status: string;
  statusTone: StatusTone;
  requests: string;
  requestCount: number;
}

const grantColumns: readonly Column<GrantRow>[] = [
  { key: "name", header: "Grant", primary: "name", sort: "name" },
  { key: "models", header: "Models", primary: "models", sort: "modelCount" },
  { key: "principals", header: "Principals", primary: "principals", sort: "principalCount" },
  { key: "scope", header: "Scope", primary: "scope", sort: "scope" },
  { key: "status", header: "Status", primary: "status", tone: "statusTone", sort: "status" },
  { key: "requests", header: "Requests 24h", primary: "requests", align: "right", sort: "requestCount" },
];

const admission24h = grantAdmission24h(modelGrants);

const grantRows: GrantRow[] = modelGrants.map((grant) => ({
  id: grant.id,
  name: grant.name,
  models: formatNumber(grant.modelIds.length),
  modelCount: grant.modelIds.length,
  principals:
    grant.principalIds.length === 0
      ? "Organization-wide"
      : formatNumber(grant.principalIds.length),
  principalCount: grant.principalIds.length,
  scope: grant.scope,
  status: grant.status,
  statusTone: GRANT_TONE[grant.status],
  requests: formatNumber(admission24h.get(grant.id) ?? 0),
  requestCount: admission24h.get(grant.id) ?? 0,
}));

interface DenialRow {
  id: string;
  principal: string;
  model: string;
  reason: string;
  blocked: string;
  attempts: string;
  attemptCount: number;
}

const denialColumns: readonly Column<DenialRow>[] = [
  { key: "principal", header: "Principal", primary: "principal", sort: "principal" },
  { key: "model", header: "Model", primary: "model", sort: "model" },
  { key: "reason", header: "Reason", primary: "reason", sort: "reason" },
  { key: "blocked", header: "Blocked", primary: "blocked", sort: "blocked" },
  { key: "attempts", header: "Denied 24h", primary: "attempts", align: "right", sort: "attemptCount" },
];

const denialRows: DenialRow[] = [...denials24h()]
  .sort((a, b) => b.requests - a.requests)
  .map((denial) => ({
    id: denial.id,
    principal: denial.principal,
    model: denial.model,
    reason: denial.reason,
    blocked: denial.blockedAt.slice(0, 10),
    attempts: formatNumber(denial.requests),
    attemptCount: denial.requests,
  }));

interface EntitlementRow {
  id: string;
  tier: string;
  models: string;
  modelCount: number;
  cap: string;
  approval: string;
  approvalTone: StatusTone;
}

const entitlementColumns: readonly Column<EntitlementRow>[] = [
  { key: "tier", header: "Tier", primary: "tier", sort: "tier" },
  { key: "models", header: "Models included", primary: "models", sort: "modelCount" },
  { key: "cap", header: "Monthly cap", primary: "cap", align: "right" },
  { key: "approval", header: "Approval", primary: "approval", tone: "approvalTone", sort: "approval" },
];

const entitlementRows: EntitlementRow[] = planEntitlements.map((entitlement) => ({
  id: entitlement.tier,
  tier: entitlement.tier,
  models: entitlement.modelIds.map(displayName).join(", "),
  modelCount: entitlement.modelIds.length,
  cap: entitlement.spendCap === null ? "Uncapped" : formatUsd(entitlement.spendCap),
  approval: entitlement.requiresApproval ? "Required" : "Not required",
  approvalTone: entitlement.requiresApproval ? "warning" : "neutral",
}));

const active = modelGrants.filter((grant) => grant.status === "active");
const inFlight = modelGrants.filter(
  (grant) => grant.status === "pending" || grant.status === "expired",
);
const grantedRequests = [...admission24h.values()].reduce((acc, n) => acc + n, 0);
const deniedRequests = denialRows.reduce((acc, row) => acc + row.attemptCount, 0);

const demandByModel = new Map<string, number>();
for (const grant of active) {
  for (const modelId of grant.modelIds) {
    demandByModel.set(
      modelId,
      (demandByModel.get(modelId) ?? 0) + (admission24h.get(grant.id) ?? 0),
    );
  }
}

const topGranted = [...demandByModel.entries()]
  .sort((a, b) => b[1] - a[1])
  .slice(0, 8)
  .map(([modelId, value]) => ({ label: displayName(modelId), value }));

const tierMix = planEntitlements.map((entitlement) => ({
  label: entitlement.tier,
  value: entitlement.modelIds.length,
}));

export default function ModelAccessPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Model access"
        description="A request is admitted only when some active grant covers both the caller and the model. Denials are the audit trail for everything refused."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Active grants"
            value={formatNumber(active.length)}
            hint={`${formatNumber(modelGrants.length)} total`}
          />
          <StatCard
            label="Admitted 24h"
            value={formatCompact(grantedRequests)}
            tone="success"
            hint="Matched a grant"
          />
          <StatCard
            label="Denied 24h"
            value={formatNumber(deniedRequests)}
            tone={deniedRequests > 0 ? "danger" : "success"}
            hint="No covering grant"
          />
          <StatCard
            label="Pending or expired"
            value={formatNumber(inFlight.length)}
            tone={inFlight.length > 0 ? "warning" : "neutral"}
            hint="Grant lifecycle"
          />
        </section>

        <section className="grid gap-4 lg:grid-cols-3">
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>Admitted requests by model</CardTitle>
              <CardDescription>
                Traffic attributed to each model across all active grants.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart
                data={topGranted}
                formatValue={(value) => formatCompact(value)}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Models per tier</CardTitle>
              <CardDescription>Plan gating, independent of any team grant.</CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart
                data={tierMix}
                centerLabel={formatNumber(tierMix.length)}
                formatValue={(value) => `${formatNumber(value)} models`}
              />
            </CardContent>
          </Card>
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Access grants</CardTitle>
            <CardDescription>
              An empty principal list means the grant is organization-wide, which
              is the broadest thing this console can express.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={grantColumns}
              rows={grantRows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No grants configured."
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Denials</CardTitle>
            <CardDescription>
              A recurring denial is usually a missing grant rather than a
              deliberate block.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={denialColumns}
              rows={denialRows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No denied requests in the last 24 hours."
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Plan entitlements</CardTitle>
            <CardDescription>
              The ceiling each tier inherits before any workspace rule applies.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={entitlementColumns}
              rows={entitlementRows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No plan entitlements."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
