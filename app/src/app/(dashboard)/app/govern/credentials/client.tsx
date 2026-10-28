"use client";

import * as React from "react";
import { Plus, Key, Copy, Check, Trash2, AlertTriangle, ShieldCheck } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toast } from "sonner";
import { createControlCredentialAction, revokeControlCredentialAction } from "./actions";

export function IssueCredentialDialog({
  triggerLabel = "Issue Control Key",
  triggerVariant = "default",
  triggerClassName = "",
}: {
  triggerLabel?: string;
  triggerVariant?: "default" | "outline" | "secondary" | "ghost" | "link";
  triggerClassName?: string;
} = {}) {
  const [open, setOpen] = React.useState(false);
  const [name, setName] = React.useState("");
  const [scope, setScope] = React.useState("control_plane");
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  const [createdSecret, setCreatedSecret] = React.useState<string | null>(null);
  const [copied, setCopied] = React.useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      toast.error("Please enter a credential name");
      return;
    }
    setIsSubmitting(true);
    const res = await createControlCredentialAction({ name, scope });
    setIsSubmitting(false);

    if (res.success && res.data) {
      setCreatedSecret(res.data.api_key);
      toast.success("Credential created successfully");
    } else {
      toast.error(res.error || "Failed to create credential");
    }
  };

  const handleCopy = () => {
    if (createdSecret) {
      navigator.clipboard.writeText(createdSecret);
      setCopied(true);
      toast.success("Secret copied to clipboard");
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const handleClose = () => {
    setOpen(false);
    setCreatedSecret(null);
    setName("");
    setCopied(false);
  };

  return (
    <>
      <Dialog open={open} onOpenChange={(v) => (v ? setOpen(true) : handleClose())}>
        <DialogTrigger asChild>
          <Button size="sm" variant={triggerVariant} className={`gap-1.5 ${triggerClassName}`}>
            <Plus className="h-4 w-4" /> {triggerLabel}
          </Button>
        </DialogTrigger>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Issue Control-Plane API Key</DialogTitle>
            <DialogDescription>
              Create an authentication key for managing organization policies, members, and models via the Control API.
            </DialogDescription>
          </DialogHeader>

          {createdSecret ? (
            <div className="space-y-4 py-3">
              <div className="rounded-lg border border-warning/50 bg-warning/10 p-4 space-y-2">
                <div className="flex items-center gap-2 text-warning font-semibold text-sm">
                  <AlertTriangle className="h-4 w-4" />
                  Save your secret now
                </div>
                <p className="text-xs text-muted-foreground">
                  This key is displayed only once. If you lose it, you will need to revoke it and generate a new one.
                </p>
                <div className="flex items-center gap-2 mt-2">
                  <code className="text-xs font-mono bg-background p-2 rounded flex-1 overflow-x-auto border select-all">
                    {createdSecret}
                  </code>
                  <Button size="sm" variant="outline" onClick={handleCopy} className="gap-1">
                    {copied ? <Check className="h-3.5 w-3.5 text-green-500" /> : <Copy className="h-3.5 w-3.5" />}
                    {copied ? "Copied" : "Copy"}
                  </Button>
                </div>
              </div>
              <DialogFooter>
                <Button onClick={handleClose}>Done</Button>
              </DialogFooter>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4 py-2">
              <div className="space-y-2">
                <Label htmlFor="key-name">Credential Name</Label>
                <Input
                  id="key-name"
                  placeholder="e.g. Terraform Control Key"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  required
                />
              </div>

              <div className="space-y-2">
                <Label>Scope</Label>
                <Select value={scope} onValueChange={setScope}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="control_plane">Control Plane Admin</SelectItem>
                    <SelectItem value="read_only">Read Only Auditor</SelectItem>
                    <SelectItem value="policy_manager">Policy Manager</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <DialogFooter className="pt-2">
                <Button type="button" variant="outline" onClick={handleClose}>
                  Cancel
                </Button>
                <Button type="submit" disabled={isSubmitting}>
                  {isSubmitting ? "Generating..." : "Generate Key"}
                </Button>
              </DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
}

export function CredentialRowActions({ id, status }: { id: string; status: string }) {
  const [loading, setLoading] = React.useState(false);

  const handleRevoke = async () => {
    if (!confirm("Are you sure you want to revoke this credential? Any system using this key will immediately be rejected.")) return;
    setLoading(true);
    const res = await revokeControlCredentialAction(id);
    setLoading(false);
    if (res.success) {
      toast.success("Credential revoked");
    } else {
      toast.error(res.error || "Failed to revoke");
    }
  };

  if (status !== "active") {
    return <span className="text-xs text-muted-foreground">—</span>;
  }

  return (
    <Button
      size="sm"
      variant="ghost"
      onClick={handleRevoke}
      disabled={loading}
      className="text-destructive hover:text-destructive text-xs h-7 px-2"
    >
      <Trash2 className="h-3.5 w-3.5 mr-1" />
      Revoke
    </Button>
  );
}

export function CredentialRowActionsWrapper({ row }: { row: { id: string; status: string } }) {
  return <CredentialRowActions id={row.id} status={row.status} />;
}

