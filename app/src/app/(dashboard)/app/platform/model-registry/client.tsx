"use client";

import * as React from "react";
import { useState, useTransition, useMemo } from "react";
import { RefreshCw, ExternalLink, Search, Sparkles, Database, CheckCircle2, AlertTriangle, ShieldCheck } from "lucide-react";
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
import { formatDate } from "@/lib/format";
import type { StatusTone } from "@/types";
import { syncModelsAction, refreshModelsAction } from "./actions";

const ACTIVE_TONE: Record<string, StatusTone> = {
  active: "success",
  inactive: "neutral",
};

const display = (kind: string) =>
  kind
    .split("_")
    .map((part) => (part ? part[0].toUpperCase() + part.slice(1) : part))
    .join(" ");

export interface ModelRow {
  id: string;
  display_name: string;
  model_slug: string;
  provider_kind: string;
  capabilities?: string[];
  context_limit?: number;
  supports_streaming?: boolean;
  supports_tools?: boolean;
  supports_reasoning?: boolean;
  status?: string;
  is_active: boolean;
  updated_at: string;
}

export function SyncModelsDialog() {
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

  const handleSync = () => {
    startTransition(async () => {
      const res = await syncModelsAction(providerKind, apiKey);
      if (res.error) {
        toast.error(res.error);
      } else {
        toast.success(
          `Successfully synchronized ${res.syncedCount ?? 0} models from ${display(providerKind)}!`
        );
        setOpen(false);
      }
    });
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button size="sm" className="gap-1.5 shadow-sm">
          <Database className="size-4" /> Sync Models
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Database className="size-5 text-primary" /> Synchronize Upstream Model Catalog
          </DialogTitle>
          <DialogDescription>
            Query the upstream provider API for newly released models, context window adjustments, and deprecations.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="space-y-1.5">
            <Label htmlFor="sync-provider-kind">Provider</Label>
            <Select value={providerKind} onValueChange={handleProviderChange}>
              <SelectTrigger id="sync-provider-kind">
                <SelectValue placeholder="Select provider" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="anthropic">Anthropic (Claude)</SelectItem>
                <SelectItem value="openai">OpenAI (GPT & o-series)</SelectItem>
                <SelectItem value="gemini">Google Gemini</SelectItem>
                <SelectItem value="huggingface">Hugging Face (Inference Providers)</SelectItem>
                <SelectItem value="azure_openai">Azure OpenAI</SelectItem>
                <SelectItem value="custom">Custom Provider</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="sync-api-key">Provider API Key</Label>
            <Input
              id="sync-api-key"
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="e.g. sk-..."
            />
            <p className="text-[11px] text-muted-foreground">
              Probes live catalog directly. The key is verified in-flight and not persisted. Local test key <code className="bg-muted px-1 py-0.5 rounded">sk-dummy-{providerKind}-key</code> works immediately.
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button variant="ghost" onClick={() => setOpen(false)} disabled={isPending}>
            Cancel
          </Button>
          <Button onClick={handleSync} disabled={isPending || !apiKey} className="gap-2">
            {isPending ? "Synchronizing…" : "Start Sync"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function ModelRegistryClient({ initialRows }: { initialRows: ModelRow[] }) {
  const [rows, setRows] = useState(initialRows);
  const [search, setSearch] = useState("");
  const [providerFilter, setProviderFilter] = useState("all");
  const [statusFilter, setStatusFilter] = useState<"all" | "authorized" | "unauthorized">("all");
  const [isRefreshing, startRefreshing] = useTransition();

  const handleRefresh = () => {
    startRefreshing(async () => {
      const res = await refreshModelsAction();
      if (res.items && res.items.length > 0) {
        setRows(res.items);
        toast.success("Model registry refreshed.");
      }
    });
  };

  const providers = useMemo(() => {
    return Array.from(new Set(rows.map((r) => r.provider_kind))).sort();
  }, [rows]);

  const filteredRows = useMemo(() => {
    return rows.filter((row) => {
      const matchesSearch =
        search === "" ||
        row.display_name.toLowerCase().includes(search.toLowerCase()) ||
        row.model_slug.toLowerCase().includes(search.toLowerCase());
      const matchesProvider =
        providerFilter === "all" || row.provider_kind === providerFilter;
      const matchesStatus =
        statusFilter === "all" ||
        (statusFilter === "authorized" && row.is_active) ||
        (statusFilter === "unauthorized" && !row.is_active);
      return matchesSearch && matchesProvider && matchesStatus;
    });
  }, [rows, search, providerFilter, statusFilter]);

  const columns: readonly Column<ModelRow>[] = [
    {
      key: "display_name",
      header: "Model",
      primary: "display_name",
      secondary: "model_slug",
      sort: "display_name",
    },
    {
      key: "provider_kind",
      header: "Provider",
      primary: "provider_kind",
      sort: "provider_kind",
      render: (row) => display(row.provider_kind),
    },
    {
      key: "capabilities",
      header: "Capabilities",
      primary: "model_slug",
      render: (row) => (
        <div className="flex flex-wrap gap-1">
          {row.supports_streaming && (
            <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-blue-500/10 text-blue-600">
              Stream
            </span>
          )}
          {row.supports_tools && (
            <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-purple-500/10 text-purple-600">
              Tools
            </span>
          )}
          {row.supports_reasoning && (
            <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-amber-500/10 text-amber-600">
              Reason
            </span>
          )}
          {!row.supports_streaming && !row.supports_tools && !row.supports_reasoning && (
            <span className="text-xs text-muted-foreground">Standard</span>
          )}
        </div>
      ),
    },
    {
      key: "context_limit",
      header: "Context Limit",
      primary: "model_slug",
      render: (row) =>
        row.context_limit && row.context_limit > 0
          ? `${(row.context_limit / 1000).toFixed(0)}k tokens`
          : "—",
    },
    {
      key: "status",
      header: "Status & Authorization",
      primary: "model_slug",
      sort: "is_active",
      render: (row) => (
        row.is_active ? (
          <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-medium bg-emerald-500/10 text-emerald-600 border border-emerald-500/30">
            <CheckCircle2 className="size-3" /> Model authorized and ready
          </span>
        ) : (
          <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-medium bg-amber-500/10 text-amber-600 border border-amber-500/30">
            <AlertTriangle className="size-3" /> Model discovered but unauthorized
          </span>
        )
      ),
    },
    {
      key: "updated_at",
      header: "Last Synchronized",
      primary: "updated_at",
      format: "date",
      align: "right",
      sort: "updated_at",
    },
  ];

  return (
    <Card>
      <CardHeader className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between pb-3">
        <div>
          <CardTitle>Registered Models</CardTitle>
          <CardDescription>
            Upstream foundation models available for proxy routing and governance rule binding.
          </CardDescription>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <div className="relative w-48 sm:w-64">
            <Search className="absolute left-2.5 top-2.5 size-3.5 text-muted-foreground" />
            <Input
              type="search"
              placeholder="Filter models…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-8 text-xs h-8"
            />
          </div>
          <Select value={providerFilter} onValueChange={setProviderFilter}>
            <SelectTrigger className="w-32 h-8 text-xs">
              <SelectValue placeholder="All Providers" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Providers</SelectItem>
              {providers.map((p) => (
                <SelectItem key={p} value={p}>
                  {display(p)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select
            value={statusFilter}
            onValueChange={(val: any) => setStatusFilter(val)}
          >
            <SelectTrigger className="w-36 h-8 text-xs">
              <SelectValue placeholder="All Statuses" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Statuses</SelectItem>
              <SelectItem value="authorized">Authorized & Ready</SelectItem>
              <SelectItem value="unauthorized">Discovered / Unauthorized</SelectItem>
            </SelectContent>
          </Select>
          <Button
            variant="outline"
            size="sm"
            onClick={handleRefresh}
            disabled={isRefreshing}
            className="gap-1.5 text-xs h-8"
          >
            <RefreshCw className={`size-3.5 ${isRefreshing ? "animate-spin" : ""}`} />
            Refresh
          </Button>
          <SyncModelsDialog />
        </div>
      </CardHeader>
      <CardContent>
        <DataTable
          columns={columns}
          rows={filteredRows}
          rowKey="id"
          pageSize={10}
          emptyMessage={
            search || providerFilter !== "all"
              ? "No models match your search filters."
              : "No models registered yet. Click 'Sync Models' to import."
          }
        />
      </CardContent>
    </Card>
  );
}