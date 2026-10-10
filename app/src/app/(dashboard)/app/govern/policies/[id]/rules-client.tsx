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

import { toast } from "sonner";

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
              className="flex min-h-[80px] w-full rounded-md border border-input bg-transparent px-3 py-2 font-mono text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              rows={4}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="parameters">Parameters (JSON)</Label>
            <textarea
              id="parameters"
              value={parametersJson}
              onChange={(e) => setParametersJson(e.target.value)}
              className="flex min-h-[80px] w-full rounded-md border border-input bg-transparent px-3 py-2 font-mono text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
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
