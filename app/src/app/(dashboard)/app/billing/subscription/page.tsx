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
}

const planColumns: readonly Column<PlanRow>[] = [
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
    }));

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Subscription"
        description="The plan recorded for this organization and the billing cycle it runs on."
      >
        {subscription ? (
          <>
            <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
              <StatCard label="Plan" value={subscription.plan_name} hint={subscription.plan_code} />
              <StatCard
                label="Status"
                value={subscription.status}
                hint={
                  subscription.cancel_at_period_end
                    ? "Cancels at period end"
                    : "No cancellation recorded"
                }
              />
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
                <CardTitle>What this is</CardTitle>
                <CardDescription>
                  Local billing state, not a payment arrangement.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-2 text-sm text-muted-foreground">
                <p>
                  This plan and cycle are rows in the control plane&apos;s own tables. No
                  payment processor is integrated, so there is no card on file, no charge,
                  and no receipt — and nothing here means money has moved.
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
                description="A subscription is only recorded once one is created for this organization, so there is no plan, cycle, or cancellation state to render. What is already real is recorded consumption: per-request cost and volume are rolled up by the worker and available under Billing → Usage."
              />
            </CardContent>
          </Card>
        )}

        <Card>
          <CardHeader>
            <CardTitle>Plan catalog</CardTitle>
            <CardDescription>
              The same catalog for every organization. No prices are stored.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={planColumns}
              rows={activePlans}
              rowKey="id"
              pageSize={10}
              emptyMessage="No plans are marked active."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}