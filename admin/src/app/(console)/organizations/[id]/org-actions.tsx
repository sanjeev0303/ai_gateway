"use client";

import * as React from "react";
import { Ban, KeyRound, RotateCcw } from "lucide-react";

import { Button } from "@ai-governance/console-ui/ui/button";
import { Label } from "@ai-governance/console-ui/ui/label";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@ai-governance/console-ui/ui/select";
import { ReasonDialog } from "../../_components/reason-dialog";
import {
  reactivateOrganizationAction,
  revokeOrganizationKeysAction,
  setOrganizationPlanAction,
  suspendOrganizationAction,
} from "../../actions";

type Pending = "suspend" | "reactivate" | "plan" | "revoke" | null;

interface OrgActionsProps {
  orgId: string;
  orgName: string;
  suspended: boolean;
  currentPlan: string;
  activeKeys: number;
  plans: { code: string; name: string }[];
}

export function OrgActions({ orgId, orgName, suspended, currentPlan, activeKeys, plans }: OrgActionsProps) {
  const [open, setOpen] = React.useState<Pending>(null);
  const [plan, setPlan] = React.useState("");
  const planFieldId = React.useId();

  const dialogProps = (kind: Exclude<Pending, null>) => ({
    open: open === kind,
    onOpenChange: (next: boolean) => setOpen(next ? kind : null),
  });

  return (
    <>
      <Button variant="outline" onClick={() => setOpen("plan")}>
        Change plan
      </Button>
      <Button variant="outline" onClick={() => setOpen("revoke")} disabled={activeKeys === 0}>
        <KeyRound aria-hidden />
        Revoke all keys
      </Button>
      {suspended ? (
        <Button onClick={() => setOpen("reactivate")}>
          <RotateCcw aria-hidden />
          Reactivate organization
        </Button>
      ) : (
        <Button variant="destructive" onClick={() => setOpen("suspend")}>
          <Ban aria-hidden />
          Suspend organization
        </Button>
      )}

      <ReasonDialog
        {...dialogProps("suspend")}
        title={`Suspend ${orgName}?`}
        description="Its gateway keys stop working at once and its members are refused by the console. Nothing is deleted: reactivating restores everything."
        confirmLabel="Suspend organization"
        destructive
        successMessage={`${orgName} is suspended.`}
        onConfirm={(reason) => suspendOrganizationAction(orgId, reason)}
      />

      <ReasonDialog
        {...dialogProps("reactivate")}
        title={`Reactivate ${orgName}?`}
        description="Its members can sign in again and its gateway keys resolve again."
        confirmLabel="Reactivate organization"
        successMessage={`${orgName} is active again.`}
        onConfirm={(reason) => reactivateOrganizationAction(orgId, reason)}
      />

      <ReasonDialog
        {...dialogProps("revoke")}
        title={`Revoke all ${activeKeys} gateway keys?`}
        description="Every active key this organization holds is revoked for good. Applications using them fail until the organization issues new keys."
        confirmLabel="Revoke keys"
        destructive
        successMessage="Gateway keys revoked."
        onConfirm={(reason) => revokeOrganizationKeysAction(orgId, reason)}
      />

      <ReasonDialog
        {...dialogProps("plan")}
        title={`Change ${orgName}’s plan`}
        description="The new plan applies at once and no payment is taken. Use it for credits, trials and corrections."
        confirmLabel="Change plan"
        blocker={plan === "" || plan === currentPlan ? "Choose the plan to move to." : undefined}
        successMessage="Plan changed."
        onConfirm={(reason) => setOrganizationPlanAction(orgId, plan, reason)}
      >
        <div className="flex flex-col gap-2">
          <Label htmlFor={planFieldId}>New plan</Label>
          <Select value={plan} onValueChange={setPlan}>
            <SelectTrigger id={planFieldId} className="w-full">
              <SelectValue placeholder="Choose a plan" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {plans.map((option) => (
                  <SelectItem key={option.code} value={option.code} disabled={option.code === currentPlan}>
                    {option.name}
                    {option.code === currentPlan ? " (current)" : ""}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>
      </ReasonDialog>
    </>
  );
}
