"use client";

import * as React from "react";
import { MoreHorizontal, Plus, Settings, RotateCcw, Trash2, RefreshCw, ExternalLink } from "lucide-react";
import { DataTable, type Column } from "@/components/dashboard";
import { ProviderCredentialActionsCell } from "./provider-credential-actions-cell";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
import { createApiClient } from "@ai-governance/api-client";
import { CONTROL_API_BASE } from "@/lib/control-api";
import { useRouter } from "next/navigation";
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { formatDate, formatNumber, formatUsd, microsToUsd } from "@/lib/format";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { Separator } from "@/components/ui/separator";

export interface CredentialRow {
  id: string;
  name: string;
  kind: string;
  fingerprint: string;
  tail: string;
  status: string;
  statusTone: "success" | "warning" | "danger" | "neutral";
  created: string;
}

const CREDENTIAL_STATUS_TONE: Record<string, "success" | "warning" | "danger" | "neutral"> = {
  active: "success",
  revoked: "neutral",
  expired: "danger",
};

interface CredentialActionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string;
  children: React.ReactNode;
  action: (e: React.FormEvent) => Promise<void>;
  submitLabel?: string;
  submitVariant?: "default" | "destructive";
}

function CredentialActionDialog({
  open,
  onOpenChange,
  title,
  description,
  children,
  action,
  submitLabel = "Confirm",
  submitVariant = "default",
}: CredentialActionDialogProps) {
  const [isPending, setIsPending] = React.useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsPending(true);
    try {
      await action(e);
      onOpenChange(false);
    } catch {
      // error handled in action
    } finally {
      setIsPending(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {open ? (
        <DialogContent>
          <form onSubmit={handleSubmit}>
            <DialogHeader>
              <DialogTitle>{title}</DialogTitle>
              <DialogDescription>{description}</DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">{children}</div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={isPending}>
                Cancel
              </Button>
              <Button type="submit" variant={submitVariant} disabled={isPending}>
                {isPending && <RefreshCw className="mr-2 h-4 w-4 animate-spin" />}
                {submitLabel}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      ) : null}
    </Dialog>
  );
}

interface HealthRow {
  provider_kind: string;
  status: string;
  status_reason?: string;
  last_error?: string;
  checked_at: string;
}

const HEALTH_TONE: Record<string, "success" | "warning" | "danger" | "neutral"> = {
  healthy: "success",
  ok: "success",
  degraded: "warning",
  down: "danger",
  error: "danger",
  unknown: "neutral",
};

interface ModelRow {
  id: string;
  provider_kind: string;
  model_slug: string;
  display_name: string;
  is_active: boolean;
  created_at: string;
}

export function ProviderDetailClient({
  account,
  credentials,
  health,
}: {
  account: any;
  credentials: any[];
  health: any[];
}) {
  const router = useRouter();
  const api = React.useMemo(() => createApiClient({ baseUrl: CONTROL_API_BASE }), []);

  const [creds, setCreds] = React.useState(credentials || []);
  const [healthData, setHealthData] = React.useState(health || []);

  // Credential dialogs
  const [addCredOpen, setAddCredOpen] = React.useState(false);
  const [addCredName, setAddCredName] = React.useState("");
  const [addCredKind, setAddCredKind] = React.useState("openai");
  const [addCredSecret, setAddCredSecret] = React.useState("");

  const [rotateCredId, setRotateCredId] = React.useState<string | null>(null);
  const [rotateCredSecret, setRotateCredSecret] = React.useState("");

  const [revokeCredId, setRevokeCredId] = React.useState<string | null>(null);
  const [isUpdating, setIsUpdating] = React.useState(false);

  // Health check
  const [checkHealthLoading, setCheckHealthLoading] = React.useState(false);

  // Sync models
  const [syncModelsOpen, setSyncModelsOpen] = React.useState(false);
  const [syncCredentialId, setSyncCredentialId] = React.useState("");
  const [syncLoading, setSyncLoading] = React.useState(false);

  // Add Credential
  const handleAddCredential = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setIsUpdating(true);
      const cred = await api.providers.createCredential(account.id, {
        name: addCredName,
        kind: addCredKind,
        secret: addCredSecret,
      });
      setCreds((prev) => [cred, ...prev]);
      toast.success("Credential added");
      setAddCredOpen(false);
      setAddCredName("");
      setAddCredSecret("");
    } catch (error) {
      toast.error("Failed to add credential");
    } finally {
      setIsUpdating(false);
    }
  };

  // Rotate Credential
  const handleRotateCredential = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!rotateCredId) return;
    try {
      setIsUpdating(true);
      const cred = await api.providers.rotateCredential(rotateCredId, { secret: rotateCredSecret });
      setCreds((prev) => prev.map((c) => (c.id === rotateCredId ? cred : c)));
      toast.success("Credential rotated");
      setRotateCredId(null);
      setRotateCredSecret("");
    } catch (error) {
      toast.error("Failed to rotate credential");
    } finally {
      setIsUpdating(false);
    }
  };

  // Revoke Credential
  const handleRevokeCredential = async () => {
    if (!revokeCredId) return;
    try {
      setIsUpdating(true);
      await api.providers.removeCredential(revokeCredId);
      setCreds((prev) => prev.filter((c) => c.id !== revokeCredId));
      toast.success("Credential revoked");
      setRevokeCredId(null);
    } catch (error) {
      toast.error("Failed to revoke credential");
    } finally {
      setIsUpdating(false);
    }
  };

  // Check Health
  const handleCheckHealth = async () => {
    // Find active credential for this account's provider kind
    const activeCred = creds.find((c) => c.status === "active" && c.kind === account.provider_kind);
    if (!activeCred) {
      toast.error("No active credential found for health check");
      return;
    }

    // We need the secret - but we don't have it stored!
    // The health check requires a secret. We'll need to prompt or use a stored one.
    // For now, we can't do this without the secret. Let's show a dialog asking for it.
    // Actually, the API has a check endpoint that takes the secret directly.
    // But we don't have the secret. This is a limitation.
    toast.error("Health check requires the credential secret. Use the 'Check now' button on the health page with a valid secret.");
  };

  // Refresh health from server
  const handleRefreshHealth = async () => {
    try {
      setCheckHealthLoading(true);
      const health = await api.providers.health();
      setHealthData(health.items);
      toast.success("Health refreshed");
    } catch {
      toast.error("Failed to refresh health");
    } finally {
      setCheckHealthLoading(false);
    }
  };

  // Sync Models
  const handleSyncModels = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!syncCredentialId) return;
    try {
      setSyncLoading(true);
      // We need the secret for the credential - this is a limitation
      // The sync endpoint requires the secret
      toast.error("Model sync requires the credential secret. Use the model registry page with a valid secret.");
    } catch {
      toast.error("Failed to sync models");
    } finally {
      setSyncLoading(false);
    }
  };

  const credentialColumns: readonly Column<CredentialRow>[] = [
    { key: "name", header: "Credential", primary: "name", secondary: "kind", sort: "name" },
    { key: "fingerprint", header: "Fingerprint", primary: "fingerprint", sort: "fingerprint" },
    { key: "tail", header: "Tail", primary: "tail", sort: "tail" },
    { key: "status", header: "Status", primary: "status", tone: "statusTone", sort: "status" },
    { key: "created", header: "Created", primary: "created", sort: "created", align: "right" },
  ];

  const credentialRows: CredentialRow[] = creds
    .filter((c) => c.provider_account_id === account.id)
    .map((c) => ({
      id: c.id,
      name: c.name,
      kind: c.kind,
      fingerprint: c.fingerprint,
      tail: c.tail,
      status: c.status,
      statusTone: CREDENTIAL_STATUS_TONE[c.status] || "neutral",
      created: formatDate(c.created_at),
    }));

  const healthRows: (HealthRow & { statusTone: "success" | "warning" | "danger" | "neutral" })[] = healthData
    .filter((h) => h.provider_kind === account.provider_kind)
    .map((h) => ({
      ...h,
      statusTone: HEALTH_TONE[h.status] || "neutral",
    }));

  function ProviderCredentialActionsCellWrapper({ row }: { row: CredentialRow }) {
    return <ProviderCredentialActionsCell row={row} isUpdating={isUpdating} onRotate={setRotateCredId} onRevoke={setRevokeCredId} />;
  }

  return (
    <div className="space-y-4">
      {/* Header Actions */}
      <div className="flex flex-wrap justify-between gap-2 items-center">
        <div className="flex gap-2">
          <Badge variant="outline">{account.provider_kind}</Badge>
          <Badge variant={account.status === "active" ? "default" : "secondary"}>{account.status}</Badge>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setAddCredOpen(true)}>
            <Plus className="mr-2 size-4" /> Add Credential
          </Button>
        </div>
      </div>

      {/* Tabs */}
      <Tabs defaultValue="credentials" className="w-full">
        <TabsList>
          <TabsTrigger value="credentials">Credentials ({credentialRows.length})</TabsTrigger>
          <TabsTrigger value="health">Health ({healthRows.length})</TabsTrigger>
        </TabsList>

        {/* Credentials Tab */}
        <TabsContent value="credentials" className="mt-4">
          <Card>
            <CardHeader>
              <CardTitle>Credentials</CardTitle>
              <CardDescription>
                API credentials for this provider account. Secrets are never displayed after creation.
              </CardDescription>
            </CardHeader>
            <CardContent>
              {credentialRows.length === 0 ? (
                <div className="text-center py-8 text-muted-foreground">
                  No credentials configured. Click "Add Credential" to add one.
                </div>
              ) : (
                <DataTable
                  columns={credentialColumns}
                  rows={credentialRows}
                  rowKey="id"
                  pageSize={10}
                  emptyMessage="No credentials."
                  ActionsComponent={ProviderCredentialActionsCellWrapper}
                />
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Health Tab */}
        <TabsContent value="health" className="mt-4">
          <div className="flex justify-between items-center mb-4">
            <Card>
              <CardHeader>
                <CardTitle>Provider Health</CardTitle>
                <CardDescription>Last probe result for this provider.</CardDescription>
              </CardHeader>
              <CardContent>
                {healthRows.length === 0 ? (
                  <div className="text-center py-8 text-muted-foreground">
                    No health data available. Click "Refresh" to run a probe.
                  </div>
                ) : (
                  <dl className="grid grid-cols-2 gap-4">
                    <div>
                      <dt className="text-sm text-muted-foreground">Status</dt>
                      <dd className="mt-1">
                        <StatusBadge tone={healthRows[0].statusTone}>{healthRows[0].status}</StatusBadge>
                      </dd>
                    </div>
                    <div>
                      <dt className="text-sm text-muted-foreground">Last Checked</dt>
                      <dd className="mt-1 tabular-nums">
                        {healthRows[0].checked_at ? healthRows[0].checked_at.slice(0, 19).replace("T", " ") : "Never"}
                      </dd>
                    </div>
                    {healthRows[0].status_reason && (
                      <div className="col-span-2">
                        <dt className="text-sm text-muted-foreground">Status Reason</dt>
                        <dd className="mt-1 text-sm">{healthRows[0].status_reason}</dd>
                      </div>
                    )}
                    {healthRows[0].last_error && (
                      <div className="col-span-2">
                        <dt className="text-sm text-muted-foreground">Last Error</dt>
                        <dd className="mt-1 text-sm text-destructive">{healthRows[0].last_error}</dd>
                      </div>
                    )}
                  </dl>
                )}
              </CardContent>
            </Card>
            <Button onClick={handleRefreshHealth} disabled={checkHealthLoading}>
              {checkHealthLoading ? <RefreshCw className="mr-2 h-4 w-4 animate-spin" /> : <RefreshCw className="mr-2 h-4 w-4" />}
              Refresh
            </Button>
          </div>
        </TabsContent>
      </Tabs>

      {/* Dialogs */}
      <CredentialActionDialog
        open={addCredOpen}
        onOpenChange={setAddCredOpen}
        title="Add Credential"
        description="Create a new API credential for this provider account."
        action={handleAddCredential}
        submitLabel="Add Credential"
      >
        <div className="grid gap-4">
          <div className="grid gap-2">
            <Label htmlFor="addCredName">Name</Label>
            <Input
              id="addCredName"
              value={addCredName}
              onChange={(e) => setAddCredName(e.target.value)}
              placeholder="e.g. Production Key"
              required
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="addCredKind">Kind</Label>
            <Select value={addCredKind} onValueChange={setAddCredKind}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="openai">OpenAI</SelectItem>
                <SelectItem value="anthropic">Anthropic</SelectItem>
                <SelectItem value="google">Google</SelectItem>
                <SelectItem value="cohere">Cohere</SelectItem>
                <SelectItem value="mistral">Mistral</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="grid gap-2">
            <Label htmlFor="addCredSecret">Secret (API Key)</Label>
            <Input
              id="addCredSecret"
              type="password"
              value={addCredSecret}
              onChange={(e) => setAddCredSecret(e.target.value)}
              placeholder="sk-... or equivalent"
              required
              autoComplete="off"
            />
            <p className="text-xs text-muted-foreground">The secret will be validated before saving and never displayed again.</p>
          </div>
        </div>
      </CredentialActionDialog>

      <CredentialActionDialog
        open={!!rotateCredId}
        onOpenChange={(v) => { if (!v) setRotateCredId(null); }}
        title="Rotate Credential"
        description="Replace the secret for this credential. The old secret will be invalidated immediately."
        action={handleRotateCredential}
        submitLabel="Rotate"
        submitVariant="default"
      >
        <div className="grid gap-4">
          <div className="grid gap-2">
            <Label htmlFor="rotateCredSecret">New Secret (API Key)</Label>
            <Input
              id="rotateCredSecret"
              type="password"
              value={rotateCredSecret}
              onChange={(e) => setRotateCredSecret(e.target.value)}
              placeholder="sk-... or equivalent"
              required
              autoComplete="off"
            />
            <p className="text-xs text-muted-foreground">The new secret will be validated before saving and never displayed again.</p>
          </div>
        </div>
      </CredentialActionDialog>

      <CredentialActionDialog
        open={!!revokeCredId}
        onOpenChange={(v) => { if (!v) setRevokeCredId(null); }}
        title="Revoke Credential"
        description="Are you sure you want to revoke this credential? It will no longer be usable for routing."
        action={handleRevokeCredential}
        submitLabel="Revoke"
        submitVariant="destructive"
      >
        <p className="text-sm text-muted-foreground">This action cannot be undone.</p>
      </CredentialActionDialog>

      <CredentialActionDialog
        open={syncModelsOpen}
        onOpenChange={setSyncModelsOpen}
        title="Sync Models"
        description="Sync the model catalog from the provider using a credential."
        action={handleSyncModels}
        submitLabel="Sync"
      >
        <div className="grid gap-4">
          <div className="grid gap-2">
            <Label htmlFor="syncCredentialId">Credential</Label>
            <Select value={syncCredentialId} onValueChange={setSyncCredentialId}>
              <SelectTrigger>
                <SelectValue placeholder="Select a credential" />
              </SelectTrigger>
              <SelectContent>
                {credentialRows.length === 0 ? (
                  <SelectItem value="none" disabled>No credentials available</SelectItem>
                ) : (
                  credentialRows.map((c) => (
                    <SelectItem key={c.id} value={c.id}>
                      {c.name} ({c.fingerprint}...{c.tail})
                    </SelectItem>
                  ))
                )}
              </SelectContent>
            </Select>
          </div>
        </div>
      </CredentialActionDialog>
    </div>
  );
}