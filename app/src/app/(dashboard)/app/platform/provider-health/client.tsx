"use client";

import * as React from "react";
import { RefreshCw, Loader2, AlertCircle, CheckCircle, AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "@/components/ui/card";
import { toast } from "sonner";
import { createApiClient } from "@ai-governance/api-client";
import { CONTROL_API_BASE } from "@/lib/control-api";
import { DataTable, type Column } from "@/components/dashboard";
import { formatRelative } from "@/lib/format";
import { toneFor } from "@/lib/org-tones";
import type { StatusTone } from "@/types";

const HEALTH_TONE: Record<string, StatusTone> = {
  healthy: "success",
  ok: "success",
  degraded: "warning",
  down: "danger",
  error: "danger",
  unknown: "neutral",
};

const display = (kind: string) =>
  kind
    .split("_")
    .map((part) => (part ? part[0].toUpperCase() + part.slice(1) : part))
    .join(" ");

interface HealthRow {
  provider_kind: string;
  status: string;
  status_reason?: string;
  last_error?: string;
  checked_at: string;
}

export function ProviderHealthClient({ initialRows }: { initialRows: HealthRow[] }) {
  const [rows, setRows] = React.useState(initialRows);
  const [isChecking, setIsChecking] = React.useState(false);
  const [checkProviderKind, setCheckProviderKind] = React.useState("");
  const [checkSecret, setCheckSecret] = React.useState("");
  const [lastResult, setLastResult] = React.useState<{
    provider_kind: string;
    status: string;
    status_reason?: string;
    last_error?: string;
    checked_at: string;
  } | null>(null);

  const api = React.useMemo(() => createApiClient({ baseUrl: CONTROL_API_BASE }), []);

  const availableProviders = [...new Set(rows.map((r) => r.provider_kind))].sort();

  const handleCheckHealth = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!checkProviderKind || !checkSecret) return;
    try {
      setIsChecking(true);
      const result = await api.providers.check({
        provider_kind: checkProviderKind,
        api_key: checkSecret,
      });
      setLastResult(result);
      // Refresh the health list
      const health = await api.providers.health();
      setRows(health.items);
      toast.success(`Health check completed: ${result.status}`);
    } catch (error) {
      toast.error("Health check failed");
    } finally {
      setIsChecking(false);
    }
  };

  const handleRefreshAll = async () => {
    try {
      const health = await api.providers.health();
      setRows(health.items);
      toast.success("Health data refreshed");
    } catch {
      toast.error("Failed to refresh health");
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap justify-between gap-2 items-center">
        <Card className="w-full max-w-md">
          <CardHeader className="pb-2">
            <CardTitle className="text-lg">Run Health Check</CardTitle>
            <CardDescription>Probe a provider with a specific credential.</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleCheckHealth} className="grid gap-4">
              <div className="grid gap-2">
                <Label htmlFor="checkProviderKind">Provider</Label>
                <Select value={checkProviderKind} onValueChange={setCheckProviderKind}>
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
                <Label htmlFor="checkSecret">API Key / Secret</Label>
                <Input
                  id="checkSecret"
                  type="password"
                  value={checkSecret}
                  onChange={(e) => setCheckSecret(e.target.value)}
                  placeholder="sk-... or equivalent"
                  required
                  autoComplete="off"
                />
                <p className="text-xs text-muted-foreground">The secret is used only for this probe and never stored.</p>
              </div>
              <Button type="submit" disabled={isChecking || !checkProviderKind || !checkSecret} className="w-full">
                {isChecking ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Checking...
                  </>
                ) : (
                  <>
                    <RefreshCw className="mr-2 h-4 w-4" />
                    Check Now
                  </>
                )}
              </Button>
            </form>
          </CardContent>
        </Card>

        <Button variant="outline" onClick={handleRefreshAll}>
          <RefreshCw className="mr-2 h-4 w-4" /> Refresh All
        </Button>
      </div>

      {lastResult && (
        <Card className="border-primary">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              {lastResult.status === "healthy" || lastResult.status === "ok" ? (
                <CheckCircle className="size-5 text-green-500" />
              ) : lastResult.status === "degraded" ? (
                <AlertTriangle className="size-5 text-yellow-500" />
              ) : (
                <AlertCircle className="size-5 text-red-500" />
              )}
              Last Check Result: {display(lastResult.provider_kind)}
            </CardTitle>
            <CardDescription>
              Ran at {formatRelative(lastResult.checked_at, new Date())}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            <div className="flex items-center gap-2">
              <span className="font-medium">Status:</span>
              <span className="px-2 py-1 rounded-full bg-primary/10 text-primary text-sm">
                {lastResult.status}
              </span>
            </div>
            {lastResult.status_reason && (
              <div className="flex items-start gap-2">
                <span className="font-medium text-sm">Reason:</span>
                <span className="text-sm text-muted-foreground">{lastResult.status_reason}</span>
              </div>
            )}
            {lastResult.last_error && (
              <div className="flex items-start gap-2">
                <span className="font-medium text-sm">Error:</span>
                <span className="text-sm text-destructive">{lastResult.last_error}</span>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Provider Health Status</CardTitle>
          <CardDescription>Outcome of the most recent probe for each provider.</CardDescription>
        </CardHeader>
        <CardContent>
          <DataTable
            columns={[
              { key: "provider", header: "Provider", primary: "provider", sort: "provider" },
              { key: "status", header: "Status", primary: "status", tone: "tone", sort: "status" },
              { key: "detail", header: "Reason", primary: "detail" },
              {
                key: "checked",
                header: "Last Probe",
                primary: "checked",
                sort: "checkedSort",
                align: "right",
              },
            ]}
            rows={rows.map((r) => ({
              ...r,
              provider: display(r.provider_kind),
              tone: toneFor(HEALTH_TONE, r.status),
              detail: r.last_error ?? r.status_reason ?? "—",
              checked: formatRelative(r.checked_at, new Date()),
              checkedSort: r.checked_at,
            }))}
            rowKey="provider_kind"
            pageSize={10}
            emptyMessage="No provider health has been recorded yet."
          />
        </CardContent>
      </Card>
    </div>
  );
}