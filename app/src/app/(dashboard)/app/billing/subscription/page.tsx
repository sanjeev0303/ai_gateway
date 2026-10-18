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
import { formatDate, formatNumber } from "@/lib/format";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@/types";
import {
  ChangePlanDialog,
  CancelSubscriptionButton,
  PlanActionsCellWrapper,
} from "./client";
import { CheckCircle2, AlertTriangle, XCircle, CreditCard } from "lucide-react";

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
  usageSummary: { Requests: number };
}> {
  const { billing } = await getApiClient();
  const [plansRes, usageRes] = await Promise.all([
    billing.listPlans(),
    billing.getUsage().catch(() => ({ Requests: 0 })),
  ]);
  const plans = plansRes.items;

  try {
    return {
      subscription: await billing.getSubscription(),
      plans,
      usageSummary: usageRes as any,
    };
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) {
      return {
        subscription: null,
        plans,
        usageSummary: usageRes as any,
      };
    }
    throw error;
  }
}

export default async function SubscriptionPage({
  searchParams,
}: {
  searchParams?: Promise<{ [key: string]: string | string[] | undefined }>;
}) {
  const params = searchParams ? await searchParams : {};
  const { subscription, plans, usageSummary } = await loadSubscription();

  const isCheckoutSuccess = params.success === "true";
  const isCheckoutCancelled = params.cancelled === "true";
  const checkoutError = typeof params.error === "string" ? params.error : null;

  const quotaLimit =
    subscription?.plan_code === "enterprise"
      ? 100_000_000
      : subscription?.plan_code === "pro"
      ? 1_000_000
      : 10_000;

  const currentRequests = usageSummary.Requests || 0;
  const usagePercent = Math.min(100, Math.round((currentRequests / quotaLimit) * 100));

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
        {/* Checkout & Payment Notification Banners */}
        {isCheckoutSuccess && (
          <div className="flex items-center gap-3 p-4 rounded-lg border border-emerald-500/30 bg-emerald-500/10 text-emerald-800 dark:text-emerald-300 mb-4">
            <CheckCircle2 className="size-5 shrink-0 text-emerald-600" />
            <div>
              <div className="font-semibold text-sm">Checkout Completed</div>
              <div className="text-xs">
                Your payment was verified successfully and your organization subscription has been upgraded.
              </div>
            </div>
          </div>
        )}
        {isCheckoutCancelled && (
          <div className="flex items-center gap-3 p-4 rounded-lg border border-border bg-muted/40 text-muted-foreground mb-4">
            <AlertTriangle className="size-5 shrink-0 text-amber-500" />
            <div>
              <div className="font-semibold text-sm text-foreground">Checkout Cancelled</div>
              <div className="text-xs">
                The checkout session was cancelled. No charges were made and your subscription remains unchanged.
              </div>
            </div>
          </div>
        )}
        {checkoutError && (
          <div className="flex items-center gap-3 p-4 rounded-lg border border-destructive/30 bg-destructive/10 text-destructive mb-4">
            <XCircle className="size-5 shrink-0" />
            <div>
              <div className="font-semibold text-sm">Payment Verification Error</div>
              <div className="text-xs">{checkoutError}</div>
            </div>
          </div>
        )}

        {/* Payment Failure Warning Banner */}
        {subscription && (subscription.status === "past_due" || subscription.status === "unpaid") && (
          <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 p-4 rounded-lg border border-destructive bg-destructive/10 text-destructive mb-4 shadow-xs">
            <div className="flex items-center gap-3">
              <AlertTriangle className="size-6 shrink-0 text-destructive" />
              <div>
                <div className="font-semibold text-sm">Action Required: Subscription Payment Failed</div>
                <div className="text-xs text-muted-foreground">
                  Your recent subscription renewal payment could not be settled ({subscription.status}). Please update payment details or re-run checkout to avoid request quota suspension.
                </div>
              </div>
            </div>
          </div>
        )}

        {subscription ? (
          <>
            <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
              <StatCard
                label="Plan"
                value={subscription.plan_name}
                hint={
                  subscription.payment_provider
                    ? `${subscription.plan_code} · ${subscription.payment_provider.toUpperCase()}`
                    : subscription.plan_code
                }
              />
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
                <CardTitle>Organization Entitlements & Billing Scope</CardTitle>
                <CardDescription>
                  Governance capabilities and gateway quotas granted by {subscription.plan_name}.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-3 text-sm text-muted-foreground">
                <div className="grid gap-3 sm:grid-cols-3">
                  <div className="rounded-md border p-3 bg-muted/20">
                    <div className="flex items-center justify-between">
                      <div className="text-xs font-semibold text-foreground">Monthly Request Quota</div>
                      <span className="text-[11px] font-mono font-medium text-muted-foreground">
                        {usagePercent}%
                      </span>
                    </div>
                    <div className="text-base font-bold text-primary mt-1">
                      {formatNumber(currentRequests)} / {formatNumber(quotaLimit)}
                    </div>
                    <div className="w-full bg-muted rounded-full h-1.5 mt-2 overflow-hidden">
                      <div
                        className={`h-full rounded-full transition-all ${
                          usagePercent > 90 ? "bg-destructive" : usagePercent > 75 ? "bg-amber-500" : "bg-primary"
                        }`}
                        style={{ width: `${usagePercent}%` }}
                      />
                    </div>
                  </div>
                  <div className="rounded-md border p-3 bg-muted/20">
                    <div className="text-xs font-semibold text-foreground">Payment Processor</div>
                    <div className="text-base font-bold text-primary mt-1 uppercase">
                      {subscription.payment_provider || "Local / System"}
                    </div>
                  </div>
                  <div className="rounded-md border p-3 bg-muted/20">
                    <div className="text-xs font-semibold text-foreground">Privacy & Guardrails</div>
                    <div className="text-base font-bold text-primary mt-1">
                      {subscription.plan_code === "enterprise"
                        ? "Full Content & Metadata"
                        : subscription.plan_code === "pro"
                        ? "Standard Guardrails"
                        : "Basic Policy"}
                    </div>
                  </div>
                </div>

                {subscription.provider_customer_id && (
                  <p className="rounded-md border bg-muted/40 p-3 text-xs">
                    <strong>Processor Reference:</strong> Customer ID:{" "}
                    <code>{subscription.provider_customer_id}</code>
                    {subscription.provider_subscription_id && (
                      <>
                        {" "}· Subscription ID: <code>{subscription.provider_subscription_id}</code>
                      </>
                    )}
                  </p>
                )}
                <p>
                  Recorded consumption for the current cycle is priced by the worker and shown under Billing → Usage.
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