import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { Plus } from "lucide-react";

import { DataTable, PageShell, type Column } from "@/components/dashboard";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { getApiClient } from "@/lib/session";
import { CreateRuleDialog, DeletePolicyButton, RuleActionsCell, ValidatePolicyButton } from "./rules-client";

export async function generateMetadata({
  params,
}: {
  params: Promise<{ id: string }>;
}): Promise<Metadata> {
  const { id } = await params;
  const { policies } = await getApiClient();
  try {
    const policy = await policies.get(id);
    return { title: `${policy.name} · Sentinel` };
  } catch (err: any) {
    if (err?.status === 404) return { title: "Not Found" };
    throw err;
  }
}

interface RuleRow {
  id: string;
  policy_id: string;
  rule_type: string;
  description: string;
  enabled: boolean;
}

const COLUMNS: readonly Column<RuleRow>[] = [
  { key: "rule_type", header: "Type", primary: "rule_type", sort: "rule_type" },
  { key: "description", header: "Description", primary: "description" },
  { key: "enabled", header: "Enabled", primary: "enabled", align: "right" },
];

export default async function PolicyDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const { policies } = await getApiClient();

  const [policy, ruleList] = await Promise.all([
    policies.get(id).catch((err) => {
      if (err?.status === 404) notFound();
      throw err;
    }),
    policies.listRules(id),
  ]);

  const rows: RuleRow[] = (ruleList.items || []).map((r) => ({
    id: r.id ?? "",
    policy_id: id,
    rule_type: r.rule_type ?? "",
    description: r.description || "No description",
    enabled: r.enabled ?? false,
  }));

  return (
    <div className="p-4 sm:p-6 space-y-6">
      <PageShell
        title={policy.name ?? "Policy"}
        description={policy.description || "No description provided."}
        actions={
          <div className="flex gap-2">
            <ValidatePolicyButton id={policy.id ?? ""} />
            <DeletePolicyButton id={policy.id ?? ""} />
          </div>
        }
      >
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <div>
              <CardTitle>Rules</CardTitle>
              <CardDescription>
                Rules applied when this policy is evaluated.
              </CardDescription>
            </div>
            <CreateRuleDialog policyId={id}>
              <Button size="sm">
                <Plus aria-hidden /> Add rule
              </Button>
            </CreateRuleDialog>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={rows}
              rowKey="id"
              ActionsComponent={RuleActionsCell}
              emptyMessage="No rules configured."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
