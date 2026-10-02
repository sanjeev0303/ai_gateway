import type { Metadata } from "next";

import { BarChart } from "@/components/charts";
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
import {
  formatCompact,
  formatNumber,
  formatPercent,
} from "@/lib/format";
import { policies } from "@/lib/mock-data";
import {
  POLICY_ENFORCEMENT_TONE,
  type PolicyCategory,
  type StatusTone,
} from "@/types";

export const metadata: Metadata = {
  title: "Policies · Sentinel",
  description: "Every guardrail rule bound to the gateway, with what it inspected and what it stopped.",
};

const CATEGORY_TONE: Record<PolicyCategory, StatusTone> = {
  pii_redaction: "warning",
  prompt_injection: "danger",
  jailbreak: "danger",
  toxicity: "warning",
  data_residency: "info",
  cost_cap: "info",
  model_allowlist: "neutral",
  retention: "neutral",
};

interface PolicyRow {
  id: string;
  name: string;
  description: string;
  category: string;
  categoryTone: StatusTone;
  enforcement: string;
  enforcementTone: StatusTone;
  scope: string;
  targets: string;
  targetCount: number;
  evaluated: string;
  blocked: string;
  blockRate: string;
  blockFraction: number;
  meterTone: StatusTone;
}

const policyColumns: readonly Column<PolicyRow>[] = [
  { key: "name", header: "Policy", primary: "name", secondary: "description", sort: "name" },
  { key: "category", header: "Category", primary: "category", tone: "categoryTone", sort: "category" },
  {
    key: "enforcement",
    header: "Enforcement",
    primary: "enforcement",
    tone: "enforcementTone",
    sort: "enforcement",
  },
  { key: "scope", header: "Scope", primary: "scope", sort: "scope" },
  {
    key: "targets",
    header: "Bound to",
    primary: "targets",
    sort: "targetCount",
  },
  {
    key: "evaluated",
    header: "Evaluated 24h",
    primary: "evaluated",
    align: "right",
  },
  { key: "blocked", header: "Blocked 24h", primary: "blocked", align: "right" },
  {
    key: "blockRate",
    header: "Block rate",
    primary: "blockRate",
    tone: "meterTone",
    meter: "blockFraction",
    meterMax: 1,
    align: "right",
    sort: "blockFraction",
  },
];

function meterTone(fraction: number): StatusTone {
  if (fraction >= 0.05) return "danger";
  if (fraction > 0) return "warning";
  return "neutral";
}

const rows: PolicyRow[] = policies.map((policy) => {
  const fraction = policy.evaluations24h > 0 ? policy.blocks24h / policy.evaluations24h : 0;
  return {
    id: policy.id,
    name: policy.name,
    description: policy.description,
    category: policy.category,
    categoryTone: CATEGORY_TONE[policy.category],
    enforcement: policy.enforcement,
    enforcementTone: POLICY_ENFORCEMENT_TONE[policy.enforcement],
    scope: policy.scope,
    targets: policy.targetIds.length === 0 ? "Everything" : formatNumber(policy.targetIds.length),
    targetCount: policy.targetIds.length,
    evaluated: formatNumber(policy.evaluations24h),
    blocked: formatNumber(policy.blocks24h),
    blockRate: formatPercent(fraction),
    blockFraction: fraction,
    meterTone: meterTone(fraction),
  };
});

const enforcing = policies.filter((policy) => policy.enforcement === "enforce");
const disabled = policies.filter((policy) => policy.enforcement === "disabled");
const totalEvaluations = policies.reduce((acc, p) => acc + p.evaluations24h, 0);
const totalBlocks = policies.reduce((acc, p) => acc + p.blocks24h, 0);

const blocksByPolicy = [...policies]
  .filter((policy) => policy.blocks24h > 0)
  .sort((a, b) => b.blocks24h - a.blocks24h)
  .map((policy) => ({ label: policy.name, value: policy.blocks24h }));

export default function PoliciesPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Policies"
        description="Rules evaluated inline on every request. Enforcement decides whether a match is logged, warned, or stopped."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Policies"
            value={formatNumber(policies.length)}
            hint={`${formatNumber(enforcing.length)} enforcing`}
          />
          <StatCard
            label="Evaluated 24h"
            value={formatCompact(totalEvaluations)}
            hint="Requests inspected"
          />
          <StatCard
            label="Blocked 24h"
            value={formatNumber(totalBlocks)}
            tone={totalBlocks > 0 ? "danger" : "success"}
            hint={totalEvaluations > 0 ? `${formatPercent(totalBlocks / totalEvaluations)} of traffic` : "No traffic yet"}
          />
          <StatCard
            label="Disabled"
            value={formatNumber(disabled.length)}
            tone={disabled.length > 0 ? "warning" : "neutral"}
            hint="Written but not running"
          />
        </section>

        {blocksByPolicy.length > 0 ? (
          <Card>
            <CardHeader>
              <CardTitle>Blocks by policy</CardTitle>
              <CardDescription>
                A policy at zero is either not matching or not inspecting anything
                useful. Check its scope before assuming it is healthy.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart data={blocksByPolicy} formatValue={(value) => formatNumber(value)} />
            </CardContent>
          </Card>
        ) : null}

        <Card>
          <CardHeader>
            <CardTitle>All policies</CardTitle>
            <CardDescription>
              Audit-only policies report matches without touching the request,
              which is the safe way to size a threshold before enforcing it.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={policyColumns}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No policies configured."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
