"use client";

import * as React from "react";
import { useState, useTransition } from "react";
import { Activity, RefreshCw, CheckCircle2, AlertTriangle, XCircle, Play } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { toast } from "sonner";
import { DataTable, type Column } from "@/components/dashboard";
import { formatRelative } from "@/lib/format";
import type { StatusTone } from "@/types";
import { probeProviderHealthAction, refreshHealthAction } from "./actions";

const HEALTH_TONE: Record<string, StatusTone> = {
  healthy: "success",
  ok: "success",
  degraded: "warning",
  unhealthy: "danger",
  down: "danger",
  error: "danger",
  unknown: "neutral",
};

const display = (kind: string) =>
  kind
    .split("_")
    .map((part) => (part ? part[0].toUpperCase() + part.slice(1) : part))
    .join(" ");

export interface HealthRow {
  provider_kind: string;
  status: string;
  status_reason?: string;
  last_error?: string;
  checked_at: string;
}

export function ProbeProviderDialog() {
  const [open, setOpen] = useState(false);
  const [providerKind, setProviderKind] = useState("anthropic");
  const [apiKey, setApiKey] = useState("sk-dummy-anthropic-key");
  const [isPending, startTransition] = useTransition();

  const handleProviderChange = (val: string) => {
    setProviderKind(val);
    if (val === "anthropic") setApiKey("sk-dummy-anthropic-key");
    else if (val === "openai") setApiKey("sk-dummy-openai-key");
    else if (val === "gemini") setApiKey("sk-dummy-gemini-key");
    else if (val === "huggingface") setApiKey("hf_dummy_test_token_12345");
    else setApiKey("");
  };

  const handleProbe = () => {
    startTransition(async () => {
      const res = await probeProviderHealthAction(providerKind, apiKey);
      if (res.error) {
        toast.error(res.error);
      } else if (res.data) {
        toast.success(`Probe succeeded: ${display(res.data.provider_kind)} is ${res.data.status.toUpperCase()}`);
        setOpen(false);
      }
    });
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button size="sm" className="gap-1.5 shadow-sm">
          <Activity className="size-4" /> Probe Provider
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Activity className="size-5 text-primary" /> Run Connectivity Probe
          </DialogTitle>
          <DialogDescription>
            Execute an active upstream heartbeat probe against a provider API endpoint.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="space-y-1.5">
            <Label htmlFor="probe-provider-kind">Provider Kind</Label>
            <Select value={providerKind} onValueChange={handleProviderChange}>
              <SelectTrigger id="probe-provider-kind">
                <SelectValue placeholder="Select provider" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="anthropic">Anthropic (Claude)</SelectItem>
                <SelectItem value="openai">OpenAI (GPT)</SelectItem>
                <SelectItem value="gemini">Google Gemini</SelectItem>
                <SelectItem value="huggingface">Hugging Face (Inference Providers)</SelectItem>
                <SelectItem value="azure_openai">Azure OpenAI</SelectItem>
                <SelectItem value="custom">Custom Upstream</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="probe-api-key">API Key or Test Secret</Label>
            <Input
              id="probe-api-key"
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="e.g. sk-ant-api03-... or sk-dummy-key"
            />
            <p className="text-xs text-muted-foreground">
              Tip: Local test mock keys like <code className="bg-muted px-1 py-0.5 rounded">sk-dummy-anthropic-key</code> validate without external network calls.
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button variant="ghost" onClick={() => setOpen(false)} disabled={isPending}>
            Cancel
          </Button>
          <Button onClick={handleProbe} disabled={isPending || !apiKey} className="gap-2">
            {isPending ? "Probing Upstream…" : "Run Probe"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function RowProbeButton({ providerKind }: { providerKind: string }) {
  const [isPending, startTransition] = useTransition();

  const handleQuickProbe = () => {
    const dummyKey = `sk-dummy-${providerKind}-key`;
    startTransition(async () => {
      const res = await probeProviderHealthAction(providerKind, dummyKey);
      if (res.error) {
        toast.error(res.error);
      } else if (res.data) {
        toast.success(`${display(providerKind)} is ${res.data.status.toUpperCase()}`);
      }
    });
  };

  return (
    <Button
      variant="outline"
      size="sm"
      className="text-xs h-7 gap-1"
      onClick={handleQuickProbe}
      disabled={isPending}
    >
      <Play className="size-3" />
      {isPending ? "Testing…" : "Probe now"}
    </Button>
  );
}

export function ProviderHealthClient({ initialRows }: { initialRows: HealthRow[] }) {
  const [rows, setRows] = useState(initialRows);
  const [isRefreshing, startRefreshing] = useTransition();

  const handleRefresh = () => {
    startRefreshing(async () => {
      const res = await refreshHealthAction();
      if (res.items && res.items.length > 0) {
        setRows(res.items);
        toast.success("Health statuses refreshed.");
      }
    });
  };

  const columns: readonly Column<HealthRow>[] = [
    {
      key: "provider_kind",
      header: "Provider",
      primary: "provider_kind",
      sort: "provider_kind",
      render: (row) => display(row.provider_kind),
    },
    {
      key: "status",
      header: "Status",
      primary: "status",
      sort: "status",
      render: (row) => (
        <span
          className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize ${
            row.status === "healthy" || row.status === "ok"
              ? "bg-emerald-500/10 text-emerald-600"
              : row.status === "degraded"
              ? "bg-amber-500/10 text-amber-600"
              : "bg-destructive/10 text-destructive"
          }`}
        >
          {row.status}
        </span>
      ),
    },
    {
      key: "status_reason",
      header: "Reason / Diagnosis",
      primary: "provider_kind",
      secondary: "last_error",
      render: (row) => (
        <div>
          <div className="text-sm">
            {row.status_reason || (row.status === "healthy" ? "All probes succeeding" : "Degraded connectivity")}
          </div>
          {row.last_error && (
            <div className="text-xs text-muted-foreground mt-0.5">{row.last_error}</div>
          )}
        </div>
      ),
    },
    {
      key: "checked_at",
      header: "Last Checked",
      primary: "checked_at",
      format: "relative",
      sort: "checked_at",
      align: "right",
    },
  ];

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-3">
        <div>
          <CardTitle>Upstream Provider Probes</CardTitle>
          <CardDescription>
            Recorded connectivity checks from the background health watchdog and manual probes.
          </CardDescription>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={handleRefresh}
            disabled={isRefreshing}
            className="gap-1.5 text-xs"
          >
            <RefreshCw className={`size-3.5 ${isRefreshing ? "animate-spin" : ""}`} />
            Refresh
          </Button>
          <ProbeProviderDialog />
        </div>
      </CardHeader>
      <CardContent>
        <DataTable
          columns={columns}
          rows={rows}
          rowKey="provider_kind"
          pageSize={10}
          emptyMessage="No providers probed yet."
          ActionsComponent={RowProbeButtonWrapper}
        />
      </CardContent>
    </Card>
  );
}

function RowProbeButtonWrapper({ row }: { row: { provider_kind: string } }) {
  return <RowProbeButton providerKind={row.provider_kind} />;
}