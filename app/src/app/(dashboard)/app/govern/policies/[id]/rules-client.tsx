"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

import * as React from "react";
import { createApiClient } from "@ai-governance/api-client";
import { CONTROL_API_BASE } from "@/lib/control-api";
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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

import { AlertCircle, CheckCircle2, ShieldCheck, Trash2 } from "lucide-react";
import { toast } from "sonner";
import type { PolicyValidationResult } from "@ai-governance/api-client";

export function ValidatePolicyButton({ id }: { id: string }) {
  const api = React.useMemo(() => createApiClient({ baseUrl: CONTROL_API_BASE }), []);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<PolicyValidationResult | null>(null);

  async function onValidate() {
    setLoading(true);
    try {
      const res = await api.policies.validate(id);
      setResult(res);
      setOpen(true);
      if (res.valid) {
        toast.success("Policy is valid");
      } else {
        toast.error("Policy validation detected issues");
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to validate policy");
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <Button variant="outline" size="sm" disabled={loading} onClick={onValidate}>
        <ShieldCheck className="mr-1.5 size-4" />
        {loading ? "Validating..." : "Validate policy"}
      </Button>
      {result && (
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogContent className="max-w-lg">
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                {result.valid ? (
                  <CheckCircle2 className="size-5 text-emerald-500" />
                ) : (
                  <AlertCircle className="size-5 text-destructive" />
                )}
                Policy Validation: {result.valid ? "Passed" : "Failed"}
              </DialogTitle>
              <DialogDescription>
                Verification of policy structure, rule definitions, and potential conflicts.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-2">
              <div className="grid grid-cols-2 gap-2 text-sm">
                <div>
                  <span className="text-muted-foreground">Enforcement:</span>{" "}
                  <span className="font-medium capitalize">{result.enforcement}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">Rules Checked:</span>{" "}
                  <span className="font-medium">{result.rules_count}</span>
                </div>
              </div>

              {result.errors.length > 0 && (
                <div className="rounded-md border border-destructive/30 bg-destructive/10 p-3 space-y-1.5">
                  <div className="font-semibold text-xs uppercase tracking-wider text-destructive">
                    Errors ({result.errors.length})
                  </div>
                  <ul className="list-disc list-inside text-sm text-destructive space-y-1">
                    {result.errors.map((err, i) => (
                      <li key={i}>{err}</li>
                    ))}
                  </ul>
                </div>
              )}

              {result.warnings.length > 0 && (
                <div className="rounded-md border border-amber-500/30 bg-amber-500/10 p-3 space-y-1.5">
                  <div className="font-semibold text-xs uppercase tracking-wider text-amber-500">
                    Warnings ({result.warnings.length})
                  </div>
                  <ul className="list-disc list-inside text-sm text-amber-600 dark:text-amber-400 space-y-1">
                    {result.warnings.map((warn, i) => (
                      <li key={i}>{warn}</li>
                    ))}
                  </ul>
                </div>
              )}

              {result.valid && result.errors.length === 0 && result.warnings.length === 0 && (
                <div className="rounded-md border border-emerald-500/30 bg-emerald-500/10 p-3 text-sm text-emerald-600 dark:text-emerald-400">
                  All rules are syntactically valid and conflict-free.
                </div>
              )}
            </div>
            <DialogFooter>
              <Button onClick={() => setOpen(false)}>Close</Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}
    </>
  );
}

export function RuleActionsCell({ row }: { row: { id: string; policy_id?: string } }) {
  const api = React.useMemo(() => createApiClient({ baseUrl: CONTROL_API_BASE }), []);
  const router = useRouter();
  const [loading, setLoading] = useState(false);

  async function onDelete() {
    if (!row.policy_id) return;
    if (!confirm("Are you sure you want to remove this rule?")) return;
    setLoading(true);
    try {
      await api.policies.deleteRule(row.policy_id, row.id);
      toast.success("Rule removed");
      router.refresh();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to remove rule");
      setLoading(false);
    }
  }

  return (
    <Button
      variant="ghost"
      size="sm"
      className="text-destructive hover:bg-destructive/10 hover:text-destructive h-8 px-2"
      disabled={loading}
      onClick={onDelete}
    >
      <Trash2 className="size-4" />
      <span className="sr-only">Delete</span>
    </Button>
  );
}

export function DeletePolicyButton({ id }: { id: string }) {
  const api = React.useMemo(() => createApiClient({ baseUrl: CONTROL_API_BASE }), []);
  const router = useRouter();
  const [loading, setLoading] = useState(false);

  async function onClick() {
    if (!confirm("Are you sure you want to delete this policy?")) return;
    setLoading(true);
    try {
      await api.policies.delete(id);
      toast.success("Policy deleted");
      router.push("/app/govern/policies");
      router.refresh();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete");
      setLoading(false);
    }
  }

  return (
    <Button variant="destructive" size="sm" disabled={loading} onClick={onClick}>
      Delete policy
    </Button>
  );
}

export function CreateRuleDialog({
  policyId,
  children,
}: {
  policyId: string;
  children: React.ReactNode;
}) {
  const api = React.useMemo(() => createApiClient({ baseUrl: CONTROL_API_BASE }), []);
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);

  const [ruleType, setRuleType] = useState("allowlist");
  const [enabled, setEnabled] = useState(true);
  const [conditionsJson, setConditionsJson] = useState("{}");
  const [parametersJson, setParametersJson] = useState("{}");

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setLoading(true);

    const formData = new FormData(e.currentTarget);
    const description = formData.get("description") as string;

    try {
      let conditions, parameters;
      try {
        conditions = JSON.parse(conditionsJson);
      } catch {
        throw new Error("Invalid conditions JSON");
      }
      try {
        parameters = JSON.parse(parametersJson);
      } catch {
        throw new Error("Invalid parameters JSON");
      }

      await api.policies.createRule(policyId, {
        rule_type: ruleType,
        description,
        enabled,
        conditions,
        parameters,
      });

      toast.success("Rule added");
      setOpen(false);
      router.refresh();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to add rule");
    } finally {
      setLoading(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle>New Policy Rule</DialogTitle>
          <DialogDescription>
            Add a rule to this policy.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={onSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="rule_type">Rule Type</Label>
            <Select value={ruleType} onValueChange={setRuleType}>
              <SelectTrigger>
                <SelectValue placeholder="Select type" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="allowlist">Allowlist</SelectItem>
                <SelectItem value="blocklist">Blocklist</SelectItem>
                <SelectItem value="rate_limit">Rate Limit</SelectItem>
                <SelectItem value="token_budget">Token Budget</SelectItem>
                <SelectItem value="budget">Cost Budget</SelectItem>
                <SelectItem value="content_filter">Content Filter</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="description">Description</Label>
            <Input id="description" name="description" placeholder="Optional notes" />
          </div>

          <div className="space-y-2">
            <Label htmlFor="conditions">Conditions (JSON)</Label>
            <textarea
              id="conditions"
              value={conditionsJson}
              onChange={(e) => setConditionsJson(e.target.value)}
              className="flex min-h-20 w-full rounded-md border border-input bg-transparent px-3 py-2 font-mono text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              rows={4}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="parameters">Parameters (JSON)</Label>
            <textarea
              id="parameters"
              value={parametersJson}
              onChange={(e) => setParametersJson(e.target.value)}
              className="flex min-h-20 w-full rounded-md border border-input bg-transparent px-3 py-2 font-mono text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              rows={4}
            />
          </div>

          <div className="flex items-center space-x-2">
            <input
              type="checkbox"
              id="enabled"
              checked={enabled}
              onChange={(e) => setEnabled(e.target.checked)}
              className="h-4 w-4"
            />
            <Label htmlFor="enabled">Enable this rule</Label>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)} disabled={loading}>
              Cancel
            </Button>
            <Button type="submit" disabled={loading}>
              Add rule
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
