"use client";

import * as React from "react";

import { useState } from "react";
import { useRouter } from "next/navigation";

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

export function CreatePolicyDialog({ children }: { children: React.ReactNode }) {
  const api = React.useMemo(() => createApiClient({ baseUrl: CONTROL_API_BASE }), []);
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [enforcement, setEnforcement] = useState("active");

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setLoading(true);

    const formData = new FormData(e.currentTarget);
    const name = formData.get("name") as string;
    const description = formData.get("description") as string;
    const priority = parseInt(formData.get("priority") as string, 10);

    try {
      await api.policies.create({
        name,
        description,
        enforcement: enforcement as any,
        priority: isNaN(priority) ? 100 : priority,
      });
      toast.success("Policy created");
      setOpen(false);
      router.refresh();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to create policy");
    } finally {
      setLoading(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>New Policy</DialogTitle>
          <DialogDescription>
            Create a new gateway policy.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={onSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Name</Label>
            <Input id="name" name="name" required autoFocus placeholder="e.g. Rate Limit Tier 1" />
          </div>
          <div className="space-y-2">
            <Label htmlFor="description">Description</Label>
            <textarea
              id="description"
              name="description"
              placeholder="What this policy does..."
              rows={3}
              className="flex min-h-[80px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="enforcement">Enforcement</Label>
            <Select value={enforcement} onValueChange={setEnforcement}>
              <SelectTrigger>
                <SelectValue placeholder="Select enforcement mode" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="active">Active (Enforced)</SelectItem>
                <SelectItem value="permissive">Permissive (Audit Only)</SelectItem>
                <SelectItem value="disabled">Disabled (Ignored)</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="priority">Priority (lower runs first)</Label>
            <Input id="priority" name="priority" type="number" defaultValue="100" required />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)} disabled={loading}>
              Cancel
            </Button>
            <Button type="submit" disabled={loading}>
              Create policy
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
