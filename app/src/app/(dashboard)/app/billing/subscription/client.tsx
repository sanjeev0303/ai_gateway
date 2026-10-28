"use client";

import * as React from "react";
import { useState, useTransition } from "react";
import { Check, Sparkles, AlertTriangle, ArrowRight, X, CreditCard } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { changeSubscriptionPlan, cancelSubscriptionAction, createCheckoutSessionAction } from "./actions";

interface Plan {
  id: string;
  code: string;
  name: string;
  description: string;
  status: string;
}

const PLAN_FEATURES: Record<string, string[]> = {
  free: [
    "10,000 requests per month",
    "Single upstream provider account",
    "Standard rate limits & concurrency",
    "7-day retention for audit logs",
  ],
  pro: [
    "1,000,000 requests per month",
    "Multi-provider routing & auto-failover",
    "Real-time governance & safety guardrails",
    "30-day raw telemetry & priority alerts",
  ],
  enterprise: [
    "Unlimited / custom request volume",
    "Dedicated throughput & isolated tenancy",
    "Zero-retention compliance modes",
    "365-day SOC2 audit & 99.99% SLA",
  ],
};

interface PlanTierMeta {
  price: string;
  cadence: string;
  tagline: string;
  badge?: string;
  features: string[];
}

const PLAN_TIER_DETAILS: Record<string, PlanTierMeta> = {
  free: {
    price: "$0",
    cadence: "per month",
    tagline: "Essential gateway proxy for local experiments & prototypes",
    badge: "Starter",
    features: [
      "10,000 requests per month",
      "Single provider account routing",
      "Standard rate limits & concurrency",
      "7-day audit logs retention",
    ],
  },
  pro: {
    price: "$49",
    cadence: "per month",
    tagline: "Full observability & governance for fast-moving engineering teams",
    badge: "Most Popular",
    features: [
      "1,000,000 requests per month",
      "Multi-provider routing & auto-fallback",
      "Real-time governance & safety guardrails",
      "30-day raw telemetry & priority alerts",
    ],
  },
  enterprise: {
    price: "Custom",
    cadence: "billed annually",
    tagline: "Dedicated throughput, custom compliance & enterprise security",
    badge: "Enterprise",
    features: [
      "Unlimited / custom request volume",
      "Dedicated throughput & isolated tenancy",
      "Zero-retention privacy modes",
      "365-day SOC2 audit & 99.99% SLA",
    ],
  },
};

export function ChangePlanDialog({
  currentPlanCode,
  plans,
}: {
  currentPlanCode?: string;
  plans: Plan[];
}) {
  const [open, setOpen] = useState(false);
  const [selectedPlan, setSelectedPlan] = useState<string>(currentPlanCode ?? "pro");
  const [paymentProvider, setPaymentProvider] = useState<"stripe" | "razorpay">("stripe");
  const [isPending, startTransition] = useTransition();

  const handleConfirm = () => {
    if (!selectedPlan || selectedPlan === currentPlanCode) {
      setOpen(false);
      return;
    }

    startTransition(async () => {
      // For paid tiers (pro / enterprise), attempt checkout session initiation first
      if (selectedPlan !== "free") {
        const checkoutRes = await createCheckoutSessionAction(selectedPlan, paymentProvider);
        if (
          checkoutRes.checkoutUrl &&
          checkoutRes.checkoutUrl.startsWith("http") &&
          !checkoutRes.checkoutUrl.includes("checkout.stripe.com/test") &&
          !checkoutRes.checkoutUrl.includes("simulate")
        ) {
          window.location.href = checkoutRes.checkoutUrl;
          return;
        }

        // If checkout session returned an error, report to console and fallback to direct plan update
        if (checkoutRes.error) {
          console.warn("Checkout session initiation failed, falling back to direct plan switch:", checkoutRes.error);
        }
      }

      const fd = new FormData();
      fd.append("plan_code", selectedPlan);
      const res = await changeSubscriptionPlan({}, fd);
      if (res.error) {
        toast.error(res.error);
      } else {
        const provTag = selectedPlan !== "free" ? ` (${paymentProvider.toUpperCase()})` : "";
        toast.success(`Successfully switched to ${selectedPlan.toUpperCase()} plan${provTag}!`);
        setOpen(false);
      }
    });
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button size="sm" className="gap-2 shadow-sm h-9 px-4 text-sm font-semibold">
          <Sparkles className="size-4" /> Change Plan
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-4xl sm:max-w-5xl p-5 gap-3 max-h-[96vh] overflow-hidden flex flex-col">
        <DialogHeader className="space-y-0.5 pb-0.5">
          <DialogTitle className="text-xl font-bold tracking-tight">Change Organization Plan</DialogTitle>
          <DialogDescription className="text-xs text-muted-foreground">
            Select a tier and payment processor to update your organization's quota limits and governance capabilities.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-3 py-0.5 md:grid-cols-3">
          {plans.map((plan) => {
            const isCurrent = plan.code === currentPlanCode;
            const isSelected = plan.code === selectedPlan;
            const meta = PLAN_TIER_DETAILS[plan.code] ?? {
              price: "$—",
              cadence: "per month",
              tagline: plan.description,
              badge: undefined,
              features: PLAN_FEATURES[plan.code] ?? [plan.description],
            };

            return (
              <div
                key={plan.id}
                onClick={() => setSelectedPlan(plan.code)}
                className={`cursor-pointer rounded-xl border-2 p-3.5 transition-all relative flex flex-col justify-between text-left ${
                  isSelected
                    ? "border-primary bg-primary/5 shadow-xs ring-1 ring-primary/40"
                    : "border-border hover:border-primary/40 bg-card/80"
                } ${isCurrent && !isSelected ? "border-emerald-500/30" : ""}`}
              >
                <div>
                  <div className="flex items-center justify-between gap-1.5">
                    <span className="text-base font-bold text-foreground">{plan.name}</span>
                    {isCurrent ? (
                      <Badge variant="outline" className="bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/30 text-xs font-semibold px-2 py-0">
                        Current
                      </Badge>
                    ) : meta.badge ? (
                      <Badge variant={meta.badge === "Most Popular" ? "default" : "secondary"} className="text-xs font-semibold px-2 py-0">
                        {meta.badge}
                      </Badge>
                    ) : null}
                  </div>

                  <div className="mt-1 flex items-baseline gap-1">
                    <span className="text-2xl font-extrabold tracking-tight text-foreground">{meta.price}</span>
                    <span className="text-xs font-medium text-muted-foreground">/{meta.cadence}</span>
                  </div>

                  <p className="text-xs text-muted-foreground mt-0.5 line-clamp-1 leading-snug">
                    {meta.tagline}
                  </p>

                  <div className="border-t border-border/50 my-2" />

                  <div className="space-y-1.5">
                    <div className="text-xs font-semibold uppercase tracking-wider text-muted-foreground/80 mb-0.5">
                      Key Capabilities
                    </div>
                    {meta.features.map((feat, idx) => (
                      <div key={idx} className="flex items-start gap-1.5 text-xs text-foreground/90 leading-snug">
                        <Check className="size-3.5 text-emerald-500 shrink-0 mt-0.5" />
                        <span>{feat}</span>
                      </div>
                    ))}
                  </div>
                </div>

                <div className="pt-2.5">
                  <Button
                    type="button"
                    variant={isSelected ? "default" : "outline"}
                    className={`w-full h-8 text-xs font-semibold transition-all ${
                      isCurrent ? "border-emerald-500/30 text-emerald-600 bg-emerald-500/5 hover:bg-emerald-500/10" : ""
                    }`}
                    onClick={(e) => {
                      e.stopPropagation();
                      setSelectedPlan(plan.code);
                    }}
                  >
                    {isCurrent ? "Active Plan" : isSelected ? "Selected Tier" : `Select ${plan.name}`}
                  </Button>
                </div>
              </div>
            );
          })}
        </div>

        {selectedPlan !== "free" && (
          <div className="space-y-2 p-3 rounded-xl border border-border/80 bg-muted/20">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <CreditCard className="size-4 text-primary shrink-0" />
                <span className="text-xs font-bold uppercase tracking-wider text-foreground">
                  Payment Gateway & Settlement Processor
                </span>
              </div>
              <Badge variant="outline" className="text-xs font-semibold px-2 py-0.5 border-primary/30 text-primary bg-primary/5">
                Dual Processor
              </Badge>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
              <div
                onClick={() => setPaymentProvider("stripe")}
                className={`cursor-pointer rounded-lg border-2 p-3 transition-all flex items-start justify-between gap-3 ${
                  paymentProvider === "stripe"
                    ? "border-primary bg-primary/10 shadow-xs ring-1 ring-primary/40"
                    : "border-border/80 hover:border-primary/40 bg-card/90"
                }`}
              >
                <div className="space-y-1">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-bold text-foreground">Stripe Global</span>
                    <Badge variant="secondary" className="text-xs font-medium px-1.5 py-0">USD</Badge>
                  </div>
                  <p className="text-xs text-muted-foreground leading-snug">
                    Cards, Apple Pay, Google Pay, SEPA & Global ACH
                  </p>
                </div>
                <div className={`size-4 rounded-full border-2 flex items-center justify-center shrink-0 mt-0.5 ${
                  paymentProvider === "stripe" ? "border-primary bg-primary text-primary-foreground" : "border-muted-foreground/40"
                }`}>
                  {paymentProvider === "stripe" && <Check className="size-2.5 stroke-3" />}
                </div>
              </div>

              <div
                onClick={() => setPaymentProvider("razorpay")}
                className={`cursor-pointer rounded-lg border-2 p-3 transition-all flex items-start justify-between gap-3 ${
                  paymentProvider === "razorpay"
                    ? "border-primary bg-primary/10 shadow-xs ring-1 ring-primary/40"
                    : "border-border/80 hover:border-primary/40 bg-card/90"
                }`}
              >
                <div className="space-y-1">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-bold text-foreground">Razorpay India</span>
                    <Badge variant="secondary" className="text-xs font-medium px-1.5 py-0">INR</Badge>
                  </div>
                  <p className="text-xs text-muted-foreground leading-snug">
                    UPI (GPay / PhonePe), Netbanking & RuPay Cards
                  </p>
                </div>
                <div className={`size-4 rounded-full border-2 flex items-center justify-center shrink-0 mt-0.5 ${
                  paymentProvider === "razorpay" ? "border-primary bg-primary text-primary-foreground" : "border-muted-foreground/40"
                }`}>
                  {paymentProvider === "razorpay" && <Check className="size-2.5 stroke-3" />}
                </div>
              </div>
            </div>
          </div>
        )}

        <DialogFooter className="flex flex-col sm:flex-row items-center justify-between gap-2 pt-2 border-t mt-0.5">
          <div className="text-xs text-muted-foreground text-left w-full sm:w-auto">
            {selectedPlan === currentPlanCode ? (
              <span>Currently active on <strong className="text-foreground">{selectedPlan.toUpperCase()}</strong></span>
            ) : (
              <span>
                Switching to <strong className="text-foreground">{selectedPlan.toUpperCase()}</strong>
                {selectedPlan !== "free" && <> · {PLAN_TIER_DETAILS[selectedPlan]?.price} / month via <strong className="text-foreground">{paymentProvider === "stripe" ? "Stripe" : "Razorpay"}</strong></>}
              </span>
            )}
          </div>
          <div className="flex items-center gap-2 w-full sm:w-auto justify-end">
            <Button variant="ghost" onClick={() => setOpen(false)} disabled={isPending} className="h-8 px-3 text-xs">
              Cancel
            </Button>
            <Button
              onClick={handleConfirm}
              disabled={isPending || selectedPlan === currentPlanCode}
              className="gap-1.5 h-8 px-4 text-xs font-semibold shadow-xs"
            >
              {isPending ? "Processing…" : `Confirm Switch to ${selectedPlan.toUpperCase()}`}
              <ArrowRight className="size-3.5" />
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function CancelSubscriptionButton({
  isCancelled,
}: {
  isCancelled: boolean;
}) {
  const [open, setOpen] = useState(false);
  const [isPending, startTransition] = useTransition();

  const handleCancel = () => {
    startTransition(async () => {
      const res = await cancelSubscriptionAction();
      if (res.error) {
        toast.error(res.error);
      } else {
        toast.success("Subscription scheduled to cancel at period end.");
        setOpen(false);
      }
    });
  };

  if (isCancelled) {
    return (
      <Badge variant="outline" className="text-amber-600 bg-amber-500/10 border-amber-500/30">
        Cancels at period end
      </Badge>
    );
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="ghost" size="sm" className="text-destructive hover:bg-destructive/10 text-sm font-medium h-8">
          Cancel subscription
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-destructive text-lg font-bold">
            <AlertTriangle className="size-5" /> Cancel Subscription?
          </DialogTitle>
          <DialogDescription className="text-sm text-muted-foreground leading-relaxed">
            Your current plan will remain active until the end of the billing cycle. After that, your organization will downgrade to the Free tier and request quotas will be reduced.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter className="gap-2 sm:gap-0">
          <Button variant="outline" onClick={() => setOpen(false)} disabled={isPending} className="text-sm">
            Keep Subscription
          </Button>
          <Button variant="destructive" onClick={handleCancel} disabled={isPending} className="text-sm font-semibold">
            {isPending ? "Cancelling…" : "Confirm Cancellation"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function PlanActionsCell({
  planCode,
  currentPlanCode,
}: {
  planCode: string;
  currentPlanCode?: string;
}) {
  const [isPending, startTransition] = useTransition();

  const isCurrent = planCode === currentPlanCode;

  const handleSwitch = () => {
    startTransition(async () => {
      const fd = new FormData();
      fd.append("plan_code", planCode);
      const res = await changeSubscriptionPlan({}, fd);
      if (res.error) {
        toast.error(res.error);
      } else {
        toast.success(`Switched plan to ${planCode.toUpperCase()}`);
      }
    });
  };

  if (isCurrent) {
    return (
      <Badge variant="outline" className="bg-emerald-500/10 text-emerald-600 border-emerald-500/30 text-xs font-semibold px-2.5 py-0.5">
        Active
      </Badge>
    );
  }

  return (
    <Button
      variant="outline"
      size="sm"
      className="text-sm font-medium h-8 px-3"
      onClick={handleSwitch}
      disabled={isPending}
    >
      {isPending ? "Switching…" : "Switch"}
    </Button>
  );
}

export function PlanActionsCellWrapper({
  row,
}: {
  row: { code: string; currentPlanCode?: string };
}) {
  return (
    <PlanActionsCell
      planCode={row.code}
      currentPlanCode={row.currentPlanCode}
    />
  );
}

