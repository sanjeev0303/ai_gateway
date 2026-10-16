import type { Metadata } from "next";

import { ApiError, type Plan, type Subscription } from "@ai-governance/api-client";

import {
  DataTable,
  EmptyState,
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
import { formatDate } from "@/lib/format";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@/types";
import {
  ChangePlanDialog,
  CancelSubscriptionButton,
  PlanActionsCellWrapper,
} from "./client";

export const metadata: Metadata = {
  title: "Subscription · Sentinel",
  description: "The plan recorded for this organization and the billing cycle it runs on.",
};

const PLAN_TONE: Record<Plan["status"], StatusTone> = {
  active: "success",
  inactive: "neutral",
};

interface PlanRow {
  id: string;
  name: string;
  code: string;
  status: Plan["status"];
  tone: StatusTone;
  description: string;
  currentPlanCode?: string;
}

const createPlanColumns = (currentCode?: string): readonly Column<PlanRow>[] => [
  { key: "name", header: "Plan", primary: "name", sort: "name" },
  { key: "code", header: "Code", primary: "code", sort: "code" },
  { key: "status", header: "Status", primary: "status", tone: "tone", sort: "status" },
  { key: "description", header: "Description", primary: "description" },
];

/**
 * The catalog is fetched first so it still renders when the organization has no
 * subscription: `getSubscription` rejects with a not-found error in that case,
 * which is an ordinary state rather than a failure.
 */
async function loadSubscription(): Promise<{
  subscription: Subscription | null;
  plans: Plan[];
}> {
  const { billing } = await getApiClient();
  const plans = (await billing.listPlans()).items;

  try {
    return { subscription: await billing.getSubscription(), plans };
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) {
      return { subscription: null, plans };
    }
    throw error;
  }
}

export default async function SubscriptionPage() {
  const { subscription, plans } = await loadSubscription();

  const activePlans: PlanRow[] = plans
    .filter((plan) => plan.status === "active")
    .map((plan) => ({
      id: plan.id,
      name: plan.name,
      code: plan.code,
      status: plan.status,
      tone: PLAN_TONE[plan.status],
      description: plan.description,
      currentPlanCode: subscription?.plan_code,
    }));

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Subscription"
        description="The plan recorded for this organization and the billing cycle it runs on."
        actions={
          <ChangePlanDialog
            currentPlanCode={subscription?.plan_code}
            plans={plans}
          />
        }
      >
        {subscription ? (
          <>
            <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
              <StatCard label="Plan" value={subscription.plan_name} hint={subscription.plan_code} />
              <div className="flex flex-col justify-between rounded-lg border bg-card p-4 shadow-sm">
                <div>
                  <div className="text-xs font-medium text-muted-foreground">Status</div>
                  <div className="text-2xl font-bold tracking-tight capitalize mt-1">
                    {subscription.status}
                  </div>
                </div>
                <div className="mt-3 flex items-center justify-between">
                  <span className="text-xs text-muted-foreground">
                    {subscription.cancel_at_period_end
                      ? "Cancels at period end"
                      : "Auto-renewing cycle"}
                  </span>
                  <CancelSubscriptionButton
                    isCancelled={subscription.cancel_at_period_end}
                  />
                </div>
              </div>
              <StatCard
                label="Period start"
                value={
                  subscription.current_period_start
                    ? formatDate(subscription.current_period_start)
                    : "—"
                }
                hint="Current billing cycle"
              />
              <StatCard
                label="Period end"
                value={
                  subscription.current_period_end ? formatDate(subscription.current_period_end) : "—"
                }
                hint="Current billing cycle"
              />
            </section>

            <Card>
              <CardHeader>
                <CardTitle>Organization Entitlements</CardTitle>
                <CardDescription>
                  Local billing state and governance capabilities granted by {subscription.plan_name}.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-2 text-sm text-muted-foreground">
                <p>
                  This plan and cycle govern policy enforcement limits, guardrail rules, and retention thresholds across all gateway routes.
                </p>
                <p>
                  Recorded consumption for the current cycle is priced by the worker and
                  shown under Billing → Usage.
                </p>
              </CardContent>
            </Card>
          </>
        ) : (
          <Card>
            <CardHeader>
              <CardTitle>No subscription on record</CardTitle>
              <CardDescription>
                This organization has no subscription row in the control plane.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <EmptyState
                title="There is no plan to show"
                description="Click 'Change Plan' above to initialize a subscription tier for this organization."
              />
            </CardContent>
          </Card>
        )}

        <Card>
          <CardHeader>
            <CardTitle>Plan catalog</CardTitle>
            <CardDescription>
              Compare available tiers and switch plans instantly with one click.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={createPlanColumns(subscription?.plan_code)}
              rows={activePlans}
              rowKey="id"
              pageSize={10}
              emptyMessage="No plans are marked active."
              ActionsComponent={PlanActionsCellWrapper}
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}