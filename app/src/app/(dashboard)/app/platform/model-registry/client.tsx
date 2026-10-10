"use client";

import * as React from "react";
import { RefreshCw, Loader2, ExternalLink, Database, CheckCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "@/components/ui/card";
import { toast } from "sonner";
import { createApiClient } from "@ai-governance/api-client";
import { CONTROL_API_BASE } from "@/lib/control-api";
import { DataTable, type Column } from "@/components/dashboard";
import { formatDate, formatNumber, formatCompact } from "@/lib/format";
import { toneFor } from "@/lib/org-tones";
import type { StatusTone } from "@/types";

const ACTIVE_TONE: Record<string, StatusTone> = {
  active: "success",
  inactive: "neutral",
};

const display = (kind: string) =>
  kind
    .split("_")
    .map((part) => (part ? part[0].toUpperCase() + part.slice(1) : part))
    .join(" ");

interface ModelRow {
  id: string;
  display_name: string;
  model_slug: string;
  provider_kind: string;
  is_active: boolean;
  updated_at: string;
}

interface SyncResult {
  synced: number;
  added: number;
  updated: number;
  errors: string[];
}

export function ModelRegistryClient({ initialRows }: { initialRows: ModelRow[] }) {
  const [rows, setRows] = React.useState(initialRows);
  const [isSyncing, setIsSyncing] = React.useState(false);
  const [syncProviderKind, setSyncProviderKind] = React.useState("");
  const [syncSecret, setSyncSecret] = React.useState("");
  const [syncResult, setSyncResult] = React.useState<SyncResult | null>(null);

  const api = React.useMemo(() => createApiClient({ baseUrl: CONTROL_API_BASE }), []);

  const availableProviders = [...new Set(rows.map((r) => r.provider_kind))].sort();

  const handleSyncModels = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!syncProviderKind || !syncSecret) return;
    try {
      setIsSyncing(true);
      const result = await api.models.sync({
        provider_kind: syncProviderKind,
        api_key: syncSecret,
      });
      const syncData: SyncResult = {
        synced: result.items.length,
        added: 0,
        updated: 0,
        errors: [],
      };
      setSyncResult(syncData);
      // Refresh the model list
      const registry = await api.models.list();
      setRows(registry.items);
      toast.success(`Synced ${result.items.length} models from ${display(syncProviderKind)}`);
    } catch (error) {
      toast.error("Model sync failed");
    } finally {
      setIsSyncing(false);
    }
  };

  const handleRefresh = async () => {
    try {
      const registry = await api.models.list();
      setRows(registry.items);
      toast.success("Model registry refreshed");
    } catch {
      toast.error("Failed to refresh model registry");
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap justify-between gap-2 items-center">
        <Card className="w-full max-w-md">
          <CardHeader className="pb-2">
            <CardTitle className="text-lg flex items-center gap-2">
              <ExternalLink className="size-5" /> Sync Models
            </CardTitle>
            <CardDescription>Fetch the latest model catalog from a provider.</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSyncModels} className="grid gap-4">
              <div className="grid gap-2">
                <Label htmlFor="syncProviderKind">Provider</Label>
                <Select value={syncProviderKind} onValueChange={setSyncProviderKind}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select provider" />
                  </SelectTrigger>
                  <SelectContent>
                    {availableProviders.length === 0 ? (
                      <SelectItem value="none" disabled>No providers available</SelectItem>
                    ) : (
                      availableProviders.map((kind) => (
                        <SelectItem key={kind} value={kind}>
                          {display(kind)}
                        </SelectItem>
                      ))
                    )}
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-2">
                <Label htmlFor="syncSecret">API Key / Secret</Label>
                <Input
                  id="syncSecret"
                  type="password"
                  value={syncSecret}
                  onChange={(e) => setSyncSecret(e.target.value)}
                  placeholder="sk-... or equivalent"
                  required
                  autoComplete="off"
                />
                <p className="text-xs text-muted-foreground">The secret is used only for this sync and never stored.</p>
              </div>
              <Button type="submit" disabled={isSyncing || !syncProviderKind || !syncSecret} className="w-full">
                {isSyncing ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Syncing...
                  </>
                ) : (
                  <>
                    <Database className="mr-2 h-4 w-4" />
                    Sync Models
                  </>
                )}
              </Button>
            </form>
          </CardContent>
        </Card>

        <Button variant="outline" onClick={handleRefresh}>
          <RefreshCw className="mr-2 h-4 w-4" /> Refresh Registry
        </Button>
      </div>

      {syncResult && (
        <Card className="border-green-500">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <CheckCircle className="size-5 text-green-500" /> Sync Complete
            </CardTitle>
            <CardDescription>Models synced from {display(syncProviderKind)}</CardDescription>
          </CardHeader>
          <CardContent className="grid grid-cols-3 gap-4 text-center">
            <div>
              <p className="text-2xl font-bold text-green-500">{syncResult.synced}</p>
              <p className="text-xs text-muted-foreground">Total Models</p>
            </div>
            <div>
              <p className="text-2xl font-bold text-blue-500">{syncResult.added}</p>
              <p className="text-xs text-muted-foreground">New Models</p>
            </div>
            <div>
              <p className="text-2xl font-bold text-purple-500">{syncResult.updated}</p>
              <p className="text-xs text-muted-foreground">Updated</p>
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Registered Models</CardTitle>
          <CardDescription>Every model the gateway can route to, grouped by the provider that serves it.</CardDescription>
        </CardHeader>
        <CardContent>
          <DataTable
            columns={[
              {
                key: "model",
                header: "Model",
                primary: "display_name",
                secondary: "model_slug",
                sort: "display_name",
              },
              { key: "provider", header: "Provider", primary: "provider", sort: "provider" },
              { key: "status", header: "Status", primary: "status", tone: "tone", sort: "status" },
              { key: "updated", header: "Updated", primary: "updated", sort: "updated", align: "right" },
            ]}
            rows={rows.map((r) => ({
              ...r,
              provider: display(r.provider_kind),
              status: r.is_active ? "active" : "inactive",
              tone: toneFor(ACTIVE_TONE, r.is_active ? "active" : "inactive"),
              updated: formatDate(r.updated_at),
            }))}
            rowKey="id"
            pageSize={10}
            emptyMessage="No models registered."
          />
        </CardContent>
      </Card>
    </div>
  );
}