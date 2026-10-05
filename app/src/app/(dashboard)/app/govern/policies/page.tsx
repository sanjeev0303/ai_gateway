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
  title: "Policies · Sentinel",
  description: "Model allow and block rules are enforced on every request; there is no API to author them yet.",
};

export default function PoliciesPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Policies"
        description="Model allow and block rules are enforced on every request; there is no API to author them yet."
      >
        <Card>
          <CardHeader>
            <CardTitle>No policy authoring</CardTitle>
            <CardDescription>
              Rules are enforced, but the control API exposes no endpoint to create,
              edit, or list them.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <EmptyState
              title="Rules are enforced, not editable here"
              description="The gateway authorizes every request against a compiled snapshot of the organization's policy, applying model allow and block rules under deny-overrides, and refuses a blocked model with a 403 before spending rate or budget quota. Those rules reach the gateway from policy and rule records in the database; no control-plane endpoint exposes them, so there is nothing to list, inspect, or edit on this page. The provider and credential configuration a rule references is real and lives under Govern → Provider accounts and Credentials."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
