"use client";

import * as React from "react";
import {
  MoreHorizontal,
  Plus,
  RotateCcw,
  Trash2,
  RefreshCw,
  ShieldCheck,
  Power,
  ExternalLink,
  CheckCircle2,
  AlertTriangle,
  XCircle,
  Database,
  Sparkles,
  Cpu,
  Layers,
  Activity,
} from "lucide-react";
import { DataTable, type Column } from "@/components/dashboard";
import { ProviderCredentialActionsCell } from "./provider-credential-actions-cell";
import { Button } from "@/components/ui/button";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
import { createApiClient } from "@ai-governance/api-client";
import { CONTROL_API_BASE } from "@/lib/control-api";
import { useRouter } from "next/navigation";
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { formatDate, formatNumber, formatUsd, microsToUsd } from "@/lib/format";
import { StatusBadge } from "@/components/dashboard/status-badge";

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

export function ProviderDetailClient({
  account,
  credentials,
  health,
  models = [],
  usage = [],
}: {
  account: any;
  credentials: any[];
  health: any[];
  models?: any[];
  usage?: any[];
}) {
  const router = useRouter();
  const api = React.useMemo(() => createApiClient({ baseUrl: CONTROL_API_BASE }), []);

  const [currentAccount, setCurrentAccount] = React.useState(account);
  const [creds, setCreds] = React.useState(credentials || []);
  const [healthData, setHealthData] = React.useState<HealthRow[]>(health || []);
  const [modelList, setModelList] = React.useState<any[]>(models || []);

  const [validationState, setValidationState] = React.useState<
    "connected" | "credential_invalid" | "provider_unreachable" | "discovery_unavailable" | "disabled"
  >(() => {
    if (account.status === "disabled") return "disabled";
    const h = (health || []).find((x: any) => x.provider_kind === account.provider_kind);
    if (!h) return "connected";
    if (h.status === "unhealthy" || h.status === "error") {
      const errStr = `${h.status_reason || ""} ${h.last_error || ""}`.toLowerCase();
      if (
        errStr.includes("credential") ||
        errStr.includes("auth") ||
        errStr.includes("401") ||
        errStr.includes("403") ||
        errStr.includes("key")
      ) {
        return "credential_invalid";
      }
      if (errStr.includes("discovery") || errStr.includes("model")) {
        return "discovery_unavailable";
      }
      return "provider_unreachable";
    }
    return "connected";
  });

  // Credential dialogs
  const [addCredOpen, setAddCredOpen] = React.useState(false);
  const [addCredName, setAddCredName] = React.useState("");
  const [addCredKind, setAddCredKind] = React.useState("api_key");
  const [addCredSecret, setAddCredSecret] = React.useState("");

  const [rotateCredId, setRotateCredId] = React.useState<string | null>(null);
  const [rotateCredSecret, setRotateCredSecret] = React.useState("");

  const [revokeCredId, setRevokeCredId] = React.useState<string | null>(null);
  const [deleteAccountOpen, setDeleteAccountOpen] = React.useState(false);
  const [isUpdating, setIsUpdating] = React.useState(false);

  // Operations loading
  const [validating, setValidating] = React.useState(false);
  const [checkHealthLoading, setCheckHealthLoading] = React.useState(false);
  const [syncLoading, setSyncLoading] = React.useState(false);

  // Add Credential
  const handleAddCredential = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setIsUpdating(true);
      const cred = await api.providers.createCredential(currentAccount.id, {
        name: addCredName.trim(),
        kind: "api_key",
        secret: addCredSecret.trim(),
      });
      setCreds((prev) => [cred, ...prev]);
      setValidationState("connected");
      toast.success("Credential added and validated");
      setAddCredOpen(false);
      setAddCredName("");
      setAddCredSecret("");
    } catch (error: any) {
      toast.error(error?.message || "Failed to add credential");
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
      setValidationState("connected");
      toast.success("Credential rotated successfully");
      setRotateCredId(null);
      setRotateCredSecret("");
    } catch (error: any) {
      toast.error(error?.message || "Failed to rotate credential");
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
    } catch (error: any) {
      toast.error(error?.message || "Failed to revoke credential");
    } finally {
      setIsUpdating(false);
    }
  };

  // Toggle Account Status (Disable / Enable)
  const handleToggleStatus = async () => {
    const nextStatus = currentAccount.status === "active" ? "disabled" : "active";
    try {
      setIsUpdating(true);
      const updated = await api.providers.updateAccount(currentAccount.id, { status: nextStatus });
      setCurrentAccount(updated);
      setValidationState(nextStatus === "disabled" ? "disabled" : "connected");
      toast.success(nextStatus === "active" ? "Provider connection enabled" : "Provider connection disabled");
    } catch (err: any) {
      toast.error(err?.message || "Failed to update connection status");
    } finally {
      setIsUpdating(false);
    }
  };

  // Delete Provider Account
  const handleDeleteAccount = async () => {
    try {
      setIsUpdating(true);
      await api.providers.deleteAccount(currentAccount.id);
      toast.success("Provider connection removed");
      router.push("/app/platform/provider-accounts");
    } catch (err: any) {
      toast.error(err?.message || "Failed to remove provider connection");
    } finally {
      setIsUpdating(false);
    }
  };

  // Test Connection / Validate Credential
  const handleTestConnection = async () => {
    try {
      setValidating(true);
      const res = await api.providers.validateAccount(currentAccount.id);
      if (res.valid) {
        setValidationState("connected");
        toast.success("Connection test successful! Stored credential is valid.");
      } else {
        setValidationState("credential_invalid");
        toast.error("Credential validation failed against upstream provider.");
      }
    } catch (err: any) {
      setValidationState("provider_unreachable");
      toast.error(err?.message || "Connection validation failed");
    } finally {
      setValidating(false);
    }
  };

  // Check Health via Account Stored Credential
  const handleCheckHealth = async () => {
    try {
      setCheckHealthLoading(true);
      const res = await api.providers.checkAccountHealth(currentAccount.id);
      setHealthData([res]);
      if (res.status === "healthy" || res.status === "ok") {
        setValidationState("connected");
      } else {
        const errStr = `${res.status_reason || ""} ${res.last_error || ""}`.toLowerCase();
        if (errStr.includes("credential") || errStr.includes("auth") || errStr.includes("401") || errStr.includes("403")) {
          setValidationState("credential_invalid");
        } else {
          setValidationState("provider_unreachable");
        }
      }
      toast.success(`Health probe completed: ${res.status}`);
    } catch (err: any) {
      setValidationState("provider_unreachable");
      toast.error(err?.message || "Health probe failed");
    } finally {
      setCheckHealthLoading(false);
    }
  };

  // Sync Models via Account Stored Credential
  const handleSyncModels = async () => {
    try {
      setSyncLoading(true);
      const res = await api.providers.syncAccountModels(currentAccount.id);
      const refreshedModels = await api.models.list();
      setModelList(refreshedModels.items || []);
      setValidationState("connected");
      toast.success(
        `Model catalog synced: ${res.total} models accessible (${res.models_created} new, ${res.models_updated} updated)`,
      );
    } catch (err: any) {
      setValidationState("discovery_unavailable");
      toast.error(err?.message || "Failed to discover models");
    } finally {
      setSyncLoading(false);
    }
  };

  // Refresh health from server
  const handleRefreshHealth = async () => {
    try {
      setCheckHealthLoading(true);
      const healthRes = await api.providers.health();
      setHealthData(healthRes.items);
      toast.success("Health records refreshed");
    } catch {
      toast.error("Failed to refresh health");
    } finally {
      setCheckHealthLoading(false);
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
    .filter((c) => c.provider_account_id === currentAccount.id)
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
    .filter((h) => h.provider_kind === currentAccount.provider_kind)
    .map((h) => ({
      ...h,
      statusTone: HEALTH_TONE[h.status] || "neutral",
    }));

  const modelsForProvider = React.useMemo(() => {
    return modelList.filter((m: any) => m.provider_kind === currentAccount.provider_kind);
  }, [modelList, currentAccount.provider_kind]);

  const providerUsage = React.useMemo(() => {
    return usage.find((u: any) => u.Provider === currentAccount.provider_kind) || {
      Requests: 0,
      TotalTokens: 0,
      CostMicros: 0,
    };
  }, [usage, currentAccount.provider_kind]);

  function ProviderCredentialActionsCellWrapper({ row }: { row: CredentialRow }) {
    return (
      <ProviderCredentialActionsCell
        row={row}
        isUpdating={isUpdating}
        onRotate={(id) => setRotateCredId(id)}
        onRevoke={(id) => setRevokeCredId(id)}
      />
    );
  }

  return (
    <div className="space-y-4">
      {/* Header Actions */}
      <div className="flex flex-wrap justify-between gap-3 items-center">
        <div className="flex items-center gap-2">
          <Badge variant="outline" className="text-sm font-medium">
            {currentAccount.provider_kind}
          </Badge>
          <Badge
            variant={currentAccount.status === "active" ? "default" : "secondary"}
            className="text-sm font-medium"
          >
            {currentAccount.status === "active" ? "Active" : "Disabled"}
          </Badge>
          {currentAccount.base_url && (
            <span className="text-xs text-muted-foreground truncate max-w-xs">
              {currentAccount.base_url}
            </span>
          )}
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {/* Test Connection Button */}
          <Button
            variant="outline"
            size="sm"
            onClick={handleTestConnection}
            disabled={validating || isUpdating || credentialRows.length === 0}
            title={credentialRows.length === 0 ? "Add a credential first" : "Validate connection against upstream provider"}
          >
            {validating ? (
              <RefreshCw className="mr-2 size-4 animate-spin" />
            ) : (
              <ShieldCheck className="mr-2 size-4 text-emerald-600 dark:text-emerald-400" />
            )}
            Test Connection
          </Button>

          {/* Discover Models Button */}
          <Button
            variant="outline"
            size="sm"
            onClick={handleSyncModels}
            disabled={syncLoading || isUpdating || credentialRows.length === 0}
            title={credentialRows.length === 0 ? "Add a credential first" : "Discover accessible models from provider"}
          >
            {syncLoading ? (
              <RefreshCw className="mr-2 size-4 animate-spin" />
            ) : (
              <RefreshCw className="mr-2 size-4 text-blue-600 dark:text-blue-400" />
            )}
            Discover Models
          </Button>

          {/* Enable / Disable Button */}
          <Button
            variant="outline"
            size="sm"
            onClick={handleToggleStatus}
            disabled={isUpdating}
          >
            <Power className="mr-2 size-4" />
            {currentAccount.status === "active" ? "Disable" : "Enable"}
          </Button>

          {/* Add Credential Button */}
          <Button size="sm" onClick={() => setAddCredOpen(true)} disabled={isUpdating}>
            <Plus className="mr-2 size-4" /> Add Credential
          </Button>

          {/* Delete Provider Account Button */}
          <Button
            variant="destructive"
            size="sm"
            onClick={() => setDeleteAccountOpen(true)}
            disabled={isUpdating}
          >
            <Trash2 className="mr-2 size-4" /> Remove
          </Button>
        </div>
      </div>

      {/* Connection Notice if Disabled */}
      {currentAccount.status === "disabled" && (
        <div className="flex items-center gap-2 p-3 text-sm rounded-md border border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-400">
          <AlertTriangle className="size-4 shrink-0" />
          <span>
            This provider connection is currently disabled. Incoming inference requests for this provider will be blocked until re-enabled.
          </span>
        </div>
      )}

      {/* Connection Validation Status Banner */}
      <div className="rounded-lg border p-4 bg-card shadow-xs">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <div className="flex items-start sm:items-center gap-3">
            {validationState === "connected" && (
              <>
                <CheckCircle2 className="size-5 text-emerald-500 shrink-0 mt-0.5 sm:mt-0" />
                <div>
                  <div className="text-sm font-semibold flex items-center gap-2">
                    <span>Provider connected</span>
                    <Badge variant="outline" className="bg-emerald-500/10 text-emerald-600 border-emerald-500/30 text-xs">
                      Ready
                    </Badge>
                  </div>
                  <div className="text-xs text-muted-foreground mt-0.5">
                    Connection verified against upstream API. Stored credentials valid and ready for inference routing.
                  </div>
                </div>
              </>
            )}
            {validationState === "credential_invalid" && (
              <>
                <XCircle className="size-5 text-destructive shrink-0 mt-0.5 sm:mt-0" />
                <div>
                  <div className="text-sm font-semibold flex items-center gap-2 text-destructive">
                    <span>Credential invalid</span>
                    <Badge variant="destructive" className="text-xs">
                      Action Required
                    </Badge>
                  </div>
                  <div className="text-xs text-muted-foreground mt-0.5">
                    Upstream provider rejected authentication (HTTP 401/403). Please rotate the API key below.
                  </div>
                </div>
              </>
            )}
            {validationState === "provider_unreachable" && (
              <>
                <AlertTriangle className="size-5 text-amber-500 shrink-0 mt-0.5 sm:mt-0" />
                <div>
                  <div className="text-sm font-semibold flex items-center gap-2 text-amber-600">
                    <span>Provider unreachable</span>
                    <Badge variant="outline" className="bg-amber-500/10 text-amber-600 border-amber-500/30 text-xs">
                      Degraded
                    </Badge>
                  </div>
                  <div className="text-xs text-muted-foreground mt-0.5">
                    Endpoint timed out or failed DNS resolution. Routing will degrade gracefully to fallback accounts.
                  </div>
                </div>
              </>
            )}
            {validationState === "discovery_unavailable" && (
              <>
                <AlertTriangle className="size-5 text-amber-500 shrink-0 mt-0.5 sm:mt-0" />
                <div>
                  <div className="text-sm font-semibold flex items-center gap-2 text-amber-600">
                    <span>Model discovery unavailable</span>
                    <Badge variant="outline" className="bg-amber-500/10 text-amber-600 border-amber-500/30 text-xs">
                      Catalog Stale
                    </Badge>
                  </div>
                  <div className="text-xs text-muted-foreground mt-0.5">
                    Authentication verified, but upstream model catalog discovery endpoint failed to respond.
                  </div>
                </div>
              </>
            )}
            {validationState === "disabled" && (
              <>
                <Power className="size-5 text-muted-foreground shrink-0 mt-0.5 sm:mt-0" />
                <div>
                  <div className="text-sm font-semibold flex items-center gap-2 text-muted-foreground">
                    <span>Provider disabled</span>
                  </div>
                  <div className="text-xs text-muted-foreground mt-0.5">
                    This provider connection is paused. Inference requests targeting this connection are currently denied.
                  </div>
                </div>
              </>
            )}
          </div>
          <div className="flex items-center gap-2 self-end sm:self-center">
            <Button
              variant="outline"
              size="sm"
              onClick={handleTestConnection}
              disabled={validating || isUpdating || credentialRows.length === 0}
              className="text-xs h-8"
            >
              {validating ? (
                <RefreshCw className="mr-1.5 size-3.5 animate-spin" />
              ) : (
                <ShieldCheck className="mr-1.5 size-3.5 text-emerald-600" />
              )}
              Re-verify
            </Button>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <Tabs defaultValue="models" className="w-full">
        <TabsList>
          <TabsTrigger value="models">Discovered Models ({modelsForProvider.length})</TabsTrigger>
          <TabsTrigger value="credentials">Credentials ({credentialRows.length})</TabsTrigger>
          <TabsTrigger value="health">Connection Health ({healthRows.length})</TabsTrigger>
          <TabsTrigger value="usage">Usage & Costs</TabsTrigger>
        </TabsList>

        {/* Models Tab */}
        <TabsContent value="models" className="mt-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-3">
              <div>
                <CardTitle>Discovered Upstream Models</CardTitle>
                <CardDescription>
                  Models identified on {currentAccount.name}. Only models that are discovered and authorized by governance policies are selectable.
                </CardDescription>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={handleSyncModels}
                disabled={syncLoading || isUpdating || credentialRows.length === 0}
                className="gap-1.5 text-xs h-8"
              >
                <RefreshCw className={`size-3.5 ${syncLoading ? "animate-spin" : ""}`} />
                {syncLoading ? "Discovering…" : "Refresh Models"}
              </Button>
            </CardHeader>
            <CardContent>
              {modelsForProvider.length === 0 ? (
                <div className="text-center py-8 text-muted-foreground">
                  <p>No models discovered yet from this provider account.</p>
                  <Button
                    variant="outline"
                    size="sm"
                    className="mt-3"
                    onClick={handleSyncModels}
                    disabled={credentialRows.length === 0 || syncLoading}
                  >
                    <Database className="mr-2 size-4" /> Discover Upstream Catalog
                  </Button>
                </div>
              ) : (
                <div className="rounded-md border">
                  <table className="w-full text-xs text-left">
                    <thead className="bg-muted/50 border-b">
                      <tr>
                        <th className="py-2.5 px-3 font-medium">Model</th>
                        <th className="py-2.5 px-3 font-medium">Capabilities</th>
                        <th className="py-2.5 px-3 font-medium">Context Limit</th>
                        <th className="py-2.5 px-3 font-medium">Status & Authorization</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y">
                      {modelsForProvider.map((m: any) => {
                        const isAuthorized = m.is_active !== false;
                        return (
                          <tr key={m.id || m.model_slug} className="hover:bg-muted/30">
                            <td className="py-2.5 px-3">
                              <div className="font-semibold text-foreground">{m.display_name}</div>
                              <code className="text-[11px] text-muted-foreground font-mono">{m.model_slug}</code>
                            </td>
                            <td className="py-2.5 px-3">
                              <div className="flex flex-wrap gap-1">
                                {m.supports_streaming && (
                                  <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-blue-500/10 text-blue-600">
                                    Stream
                                  </span>
                                )}
                                {m.supports_tools && (
                                  <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-purple-500/10 text-purple-600">
                                    Tools
                                  </span>
                                )}
                                {m.supports_reasoning && (
                                  <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-amber-500/10 text-amber-600">
                                    Reasoning
                                  </span>
                                )}
                              </div>
                            </td>
                            <td className="py-2.5 px-3 tabular-nums">
                              {m.context_limit && m.context_limit > 0 ? `${(m.context_limit / 1000).toFixed(0)}k tokens` : "—"}
                            </td>
                            <td className="py-2.5 px-3">
                              {isAuthorized ? (
                                <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-medium bg-emerald-500/10 text-emerald-600 border border-emerald-500/30">
                                  <CheckCircle2 className="size-3" /> Model authorized and ready
                                </span>
                              ) : (
                                <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-medium bg-amber-500/10 text-amber-600 border border-amber-500/30">
                                  <AlertTriangle className="size-3" /> Model discovered but unauthorized
                                </span>
                              )}
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Credentials Tab */}
        <TabsContent value="credentials" className="mt-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-3">
              <div>
                <CardTitle>Organization Provider Credentials</CardTitle>
                <CardDescription>
                  API credentials owned by your organization for connecting to {currentAccount.name}.
                  Secrets are validated against the provider before storage, encrypted securely, and never returned in full.
                </CardDescription>
              </div>
              <Button size="sm" onClick={() => setAddCredOpen(true)} disabled={isUpdating} className="gap-1.5 text-xs h-8">
                <Plus className="size-3.5" /> Add Credential
              </Button>
            </CardHeader>
            <CardContent>
              {credentialRows.length === 0 ? (
                <div className="text-center py-8 text-muted-foreground">
                  <p>No credentials configured for this provider account.</p>
                  <Button
                    variant="outline"
                    size="sm"
                    className="mt-3"
                    onClick={() => setAddCredOpen(true)}
                  >
                    <Plus className="mr-2 size-4" /> Add First Credential
                  </Button>
                </div>
              ) : (
                <DataTable
                  columns={credentialColumns}
                  rows={credentialRows}
                  rowKey="id"
                  pageSize={10}
                  emptyMessage="No credentials found."
                  ActionsComponent={ProviderCredentialActionsCellWrapper}
                />
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Health Tab */}
        <TabsContent value="health" className="mt-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <div>
                <CardTitle>Provider Health & Connectivity</CardTitle>
                <CardDescription>
                  Probe results and validation history for this provider connection.
                </CardDescription>
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleCheckHealth}
                  disabled={checkHealthLoading || credentialRows.length === 0}
                >
                  {checkHealthLoading ? (
                    <RefreshCw className="mr-2 size-4 animate-spin" />
                  ) : (
                    <ShieldCheck className="mr-2 size-4" />
                  )}
                  Probe Now
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleRefreshHealth}
                  disabled={checkHealthLoading}
                >
                  <RefreshCw className="mr-2 size-4" />
                  Refresh
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              {healthRows.length === 0 ? (
                <div className="text-center py-8 text-muted-foreground">
                  No health probe recorded yet. Click "Probe Now" to check connectivity using your stored credential.
                </div>
              ) : (
                <dl className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div>
                    <dt className="text-sm text-muted-foreground">Status</dt>
                    <dd className="mt-1">
                      <StatusBadge tone={healthRows[0].statusTone}>{healthRows[0].status}</StatusBadge>
                    </dd>
                  </div>
                  <div>
                    <dt className="text-sm text-muted-foreground">Last Checked</dt>
                    <dd className="mt-1 text-sm tabular-nums">
                      {healthRows[0].checked_at
                        ? healthRows[0].checked_at.slice(0, 19).replace("T", " ")
                        : "Never"}
                    </dd>
                  </div>
                  {healthRows[0].status_reason && (
                    <div className="col-span-1 sm:col-span-2">
                      <dt className="text-sm text-muted-foreground">Status Reason</dt>
                      <dd className="mt-1 text-sm">{healthRows[0].status_reason}</dd>
                    </div>
                  )}
                  {healthRows[0].last_error && (
                    <div className="col-span-1 sm:col-span-2">
                      <dt className="text-sm text-muted-foreground">Last Observed Error</dt>
                      <dd className="mt-1 text-sm text-destructive font-mono bg-destructive/10 p-2 rounded">
                        {healthRows[0].last_error}
                      </dd>
                    </div>
                  )}
                </dl>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Usage & Cost Tab */}
        <TabsContent value="usage" className="mt-4">
          <Card>
            <CardHeader>
              <CardTitle>Provider Usage & Cost Metrics</CardTitle>
              <CardDescription>
                Live metered requests, token consumption, and priced expenditures attributed to {currentAccount.name} over the last 30 days.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                <div className="rounded-lg border p-3.5 bg-muted/20">
                  <div className="text-xs font-medium text-muted-foreground">Total Requests</div>
                  <div className="text-2xl font-bold tracking-tight mt-1 tabular-nums">
                    {formatNumber(providerUsage.Requests || 0)}
                  </div>
                </div>
                <div className="rounded-lg border p-3.5 bg-muted/20">
                  <div className="text-xs font-medium text-muted-foreground">Total Tokens Processed</div>
                  <div className="text-2xl font-bold tracking-tight mt-1 tabular-nums">
                    {formatNumber(providerUsage.TotalTokens || 0)}
                  </div>
                </div>
                <div className="rounded-lg border p-3.5 bg-muted/20">
                  <div className="text-xs font-medium text-muted-foreground">Recorded Expenditure (USD)</div>
                  <div className="text-2xl font-bold tracking-tight text-primary mt-1 tabular-nums">
                    {formatUsd(microsToUsd(providerUsage.CostMicros || 0))}
                  </div>
                </div>
              </div>
              <p className="text-xs text-muted-foreground">
                Metrics reflect actual gateway inference traffic priced in real-time by the asynchronous rating worker.
              </p>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Dialog: Add Credential */}
      <CredentialActionDialog
        open={addCredOpen}
        onOpenChange={setAddCredOpen}
        title="Add Provider Credential"
        description="Enter the API key for this provider account. The secret is validated immediately and never logged or exposed."
        action={handleAddCredential}
        submitLabel="Save & Validate"
      >
        <div className="grid gap-4">
          <div className="grid gap-2">
            <Label htmlFor="addCredName">Name / Label</Label>
            <Input
              id="addCredName"
              value={addCredName}
              onChange={(e) => setAddCredName(e.target.value)}
              placeholder="e.g. Production Master Key"
              required
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="addCredKind">Kind</Label>
            <Select value={addCredKind} onValueChange={setAddCredKind}>
              <SelectTrigger id="addCredKind">
                <SelectValue placeholder="Select kind" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="api_key">API Key</SelectItem>
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
              placeholder="Enter provider API key"
              required
              autoComplete="off"
            />
            <p className="text-xs text-muted-foreground">
              The secret will be validated against the upstream provider before saving. It will never be returned to the browser in full.
            </p>
          </div>
        </div>
      </CredentialActionDialog>

      {/* Dialog: Rotate Credential */}
      <CredentialActionDialog
        open={!!rotateCredId}
        onOpenChange={(v) => {
          if (!v) setRotateCredId(null);
        }}
        title="Rotate Provider Credential"
        description="Replace the active secret with zero downtime. The new key is validated first; only after validation succeeds is the old key revoked."
        action={handleRotateCredential}
        submitLabel="Rotate Key"
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
              placeholder="Enter new provider API key"
              required
              autoComplete="off"
            />
            <p className="text-xs text-muted-foreground">
              The new key will be tested against the provider. If validation fails, rotation will abort safely.
            </p>
          </div>
        </div>
      </CredentialActionDialog>

      {/* Dialog: Revoke Credential */}
      <CredentialActionDialog
        open={!!revokeCredId}
        onOpenChange={(v) => {
          if (!v) setRevokeCredId(null);
        }}
        title="Revoke Credential"
        description="Are you sure you want to revoke this provider credential? It will be permanently invalidated and its secret destroyed."
        action={handleRevokeCredential}
        submitLabel="Revoke Key"
        submitVariant="destructive"
      >
        <p className="text-sm text-destructive font-medium">
          This action cannot be undone. In-flight requests using this key will immediately fail once evicted.
        </p>
      </CredentialActionDialog>

      {/* Dialog: Delete Provider Account */}
      <CredentialActionDialog
        open={deleteAccountOpen}
        onOpenChange={setDeleteAccountOpen}
        title="Remove Provider Connection"
        description={`Are you sure you want to remove the provider connection "${currentAccount.name}"?`}
        action={handleDeleteAccount}
        submitLabel="Delete Connection"
        submitVariant="destructive"
      >
        <p className="text-sm text-destructive font-medium">
          This will permanently delete this provider connection and destroy all associated credentials from secure storage.
        </p>
      </CredentialActionDialog>
    </div>
  );
}