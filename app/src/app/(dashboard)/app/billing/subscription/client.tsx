"use client";

import * as React from "react";
import { useState, useTransition } from "react";
import { Check, Sparkles, AlertTriangle, ArrowRight, X } from "lucide-react";
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
import { changeSubscriptionPlan, cancelSubscriptionAction } from "./actions";

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
    "Community support",
  ],
  pro: [
    "1,000,000 requests per month",
    "Multi-provider routing & fallback",
    "Real-time governance policies & guardrails",
    "30-day raw usage telemetry retention",
    "Priority in-app notifications & webhooks",
    "Email & standard SLA support",
  ],
  enterprise: [
    "Unlimited / custom request volume",
    "Dedicated throughput & isolated tenancy",
    "Full content / metadata privacy modes",
    "365-day compliance audit retention",
    "Custom price rate sheet & discount tiers",
    "Dedicated Slack channel & 99.99% SLA",
  ],
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
  const [isPending, startTransition] = useTransition();

  const handleConfirm = () => {
    if (!selectedPlan || selectedPlan === currentPlanCode) {
      setOpen(false);
      return;
    }

    startTransition(async () => {
      const fd = new FormData();
      fd.append("plan_code", selectedPlan);
      const res = await changeSubscriptionPlan({}, fd);
      if (res.error) {
        toast.error(res.error);
      } else {
        toast.success(`Successfully switched to ${selectedPlan.toUpperCase()} plan!`);
        setOpen(false);
      }
    });
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button size="sm" className="gap-1.5 shadow-sm">
          <Sparkles className="size-4" /> Change Plan
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle className="text-xl">Change Organization Plan</DialogTitle>
          <DialogDescription>
            Select a tier that matches your expected monthly token consumption and governance requirements.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4 md:grid-cols-3">
          {plans.map((plan) => {
            const isCurrent = plan.code === currentPlanCode;
            const isSelected = plan.code === selectedPlan;
            const features = PLAN_FEATURES[plan.code] ?? [plan.description];

            return (
              <Card
                key={plan.id}
                onClick={() => setSelectedPlan(plan.code)}
                className={`cursor-pointer transition-all border-2 relative flex flex-col justify-between ${
                  isSelected
                    ? "border-primary shadow-md bg-accent/20"
                    : "border-border hover:border-primary/50"
                }`}
              >
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-lg font-semibold">{plan.name}</CardTitle>
                    {isCurrent && (
                      <Badge variant="outline" className="text-xs bg-emerald-500/10 text-emerald-600 border-emerald-500/30">
                        Current
                      </Badge>
                    )}
                  </div>
                  <CardDescription className="text-xs mt-1 min-h-[32px] line-clamp-2">
                    {plan.description}
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-2 text-xs flex-1">
                  <div className="font-semibold text-muted-foreground uppercase text-[10px] tracking-wider mb-2">
                    Included Capabilities
                  </div>
                  {features.map((feat, idx) => (
                    <div key={idx} className="flex items-start gap-1.5 text-muted-foreground">
                      <Check className="size-3.5 text-emerald-500 shrink-0 mt-0.5" />
                      <span>{feat}</span>
                    </div>
                  ))}
                </CardContent>
                <CardFooter className="pt-3 border-t">
                  <Button
                    type="button"
                    size="sm"
                    variant={isSelected ? "default" : "outline"}
                    className="w-full text-xs"
                    onClick={(e) => {
                      e.stopPropagation();
                      setSelectedPlan(plan.code);
                    }}
                  >
                    {isCurrent ? "Active Plan" : isSelected ? "Selected" : "Select"}
                  </Button>
                </CardFooter>
              </Card>
            );
          })}
        </div>

        <DialogFooter className="gap-2 sm:gap-0">
          <Button variant="ghost" onClick={() => setOpen(false)} disabled={isPending}>
            Cancel
          </Button>
          <Button
            onClick={handleConfirm}
            disabled={isPending || selectedPlan === currentPlanCode}
            className="gap-2"
          >
            {isPending ? "Updating…" : `Confirm Switch to ${selectedPlan.toUpperCase()}`}
            <ArrowRight className="size-4" />
          </Button>
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
        <Button variant="ghost" size="sm" className="text-destructive hover:bg-destructive/10 text-xs">
          Cancel subscription
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-destructive">
            <AlertTriangle className="size-5" /> Cancel Subscription?
          </DialogTitle>
          <DialogDescription>
            Your current plan will remain active until the end of the billing cycle. After that, your organization will downgrade to the Free tier and request quotas will be reduced.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)} disabled={isPending}>
            Keep Subscription
          </Button>
          <Button variant="destructive" onClick={handleCancel} disabled={isPending}>
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
      <Badge variant="outline" className="bg-emerald-500/10 text-emerald-600 border-emerald-500/30 text-xs">
        Active
      </Badge>
    );
  }

  return (
    <Button
      variant="outline"
      size="sm"
      className="text-xs h-7"
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

