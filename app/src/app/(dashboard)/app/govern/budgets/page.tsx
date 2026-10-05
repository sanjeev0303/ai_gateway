import type { Metadata } from "next";

import { EmptyState, PageShell } from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export const metadata: Metadata = {
  title: "Budgets · Sentinel",
  description: "Spend caps are enforced on the request path; there is no API to author them yet.",
};

export default function BudgetsPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Budgets"
        description="Spend caps are enforced on the request path; there is no API to author them yet."
      >
        <Card>
          <CardHeader>
            <CardTitle>No budget authoring</CardTitle>
            <CardDescription>
              Caps are enforced, but the control API exposes no endpoint to create,
              edit, or list them.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <EmptyState
              title="Caps are enforced, not editable here"
              description="The gateway checks an organization's spend against its cap on every request, using the limit and window from the organization's compiled policy snapshot, and refuses the call with a 429 once the cap is reached. Those caps reach the gateway from policy records in the database; no control-plane endpoint exposes a budget, quota, or spend cap, so there is nothing to list or edit on this page. The spend a cap is measured against is already attributed per team and per project under Observe → Costs."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
