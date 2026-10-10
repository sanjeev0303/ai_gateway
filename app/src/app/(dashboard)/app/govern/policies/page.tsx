import type { Metadata } from "next";
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
import { CreatePolicyDialog } from "./policies-client";

export const metadata: Metadata = {
  title: "Policies · Sentinel",
  description: "Global AI gateway policies.",
};

interface PolicyRow {
  id: string;
  name: string;
  description: string;
  enforcement: string;
  priority: number;
  detailUrl: string;
}

const COLUMNS: readonly Column<PolicyRow>[] = [
  { key: "name", header: "Policy", primary: "name", secondary: "description", sort: "name", href: "detailUrl" },
  { key: "enforcement", header: "Enforcement", primary: "enforcement", align: "left", sort: "enforcement" },
  { key: "priority", header: "Priority", primary: "priority", format: "number", align: "right", sort: "priority" },
];

export default async function PoliciesPage() {
  const { policies } = await getApiClient();
  const policyList = await policies.list();

  const rows: PolicyRow[] = (policyList.items || []).map((pol) => {
    return {
      id: pol.id ?? "",
      name: pol.name ?? "",
      description: pol.description ?? "",
      enforcement: pol.enforcement ?? "",
      priority: pol.priority ?? 0,
      detailUrl: `/app/govern/policies/${pol.id}`,
    };
  });

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Policies"
        description="Global organizational AI gateway policies."
        actions={
          <CreatePolicyDialog>
            <Button size="sm">
              <Plus aria-hidden /> New policy
            </Button>
          </CreatePolicyDialog>
        }
      >
        <Card>
          <CardHeader>
            <CardTitle>All policies</CardTitle>
            <CardDescription>
              Policies are evaluated in priority order.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={rows}
              rowKey="id"
              emptyMessage="No policies created yet."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
