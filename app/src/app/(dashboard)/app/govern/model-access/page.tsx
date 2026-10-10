"use client";

import { useState, useEffect, useCallback } from "react";
import { Search, Filter, Save, Eye, Check, X, AlertTriangle, ChevronDown, ChevronUp } from "lucide-react";

import { DataTable, PageShell, type Column } from "@/components/dashboard";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
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
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { fetchModelAccessAction, updateModelAccessAction, previewModelAccessAction } from "./actions";
import type { ModelAccessEntry, UpdateModelAccessRequest, ModelAccessPreviewRequest, ModelAccessPreviewResponse } from "@ai-governance/api-client";

// API response types use snake_case
interface ModelAccessEntryAPI {
  provider?: string;
  model?: string;
  allowed?: boolean;
  source?: string;
  policy_id?: string;
  rule_id?: string;
  priority?: number;
  effective?: boolean;
}

interface ModelAccessPreviewResponseAPI {
  provider?: string;
  model?: string;
  current_decision?: "allow" | "block";
  preview_decision?: "allow" | "block";
  would_change?: boolean;
  blocking_rules?: ModelAccessEntryAPI[];
  allowing_rules?: ModelAccessEntryAPI[];
}

// Internal display types use camelCase
interface ModelAccessRow {
  id: string;
  provider: string;
  model: string;
  allowed: boolean;
  source: string;
  policyId: string;
  ruleId: string;
  priority: number;
  effective: boolean;
}

function toRow(entry: ModelAccessEntryAPI): ModelAccessRow {
  const provider = entry.provider ?? "";
  const model = entry.model ?? "";
  return {
    id: `${provider}/${model}`,
    provider,
    model,
    allowed: entry.allowed ?? false,
    source: entry.source ?? "",
    policyId: entry.policy_id ?? "",
    ruleId: entry.rule_id ?? "",
    priority: entry.priority ?? 0,
    effective: entry.effective ?? false,
  };
}

interface PreviewResponseDisplay {
  provider: string;
  model: string;
  currentDecision: "allow" | "block";
  previewDecision: "allow" | "block";
  wouldChange: boolean;
  blockingRules: ModelAccessRow[];
  allowingRules: ModelAccessRow[];
}

function toPreviewResponse(api: ModelAccessPreviewResponseAPI): PreviewResponseDisplay {
  return {
    provider: api.provider ?? "",
    model: api.model ?? "",
    currentDecision: api.current_decision ?? "allow",
    previewDecision: api.preview_decision ?? "allow",
    wouldChange: api.would_change ?? false,
    blockingRules: (api.blocking_rules ?? []).map(toRow),
    allowingRules: (api.allowing_rules ?? []).map(toRow),
  };
}

const SOURCE_BADGE_VARIANTS: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  allowlist: "default",
  blocklist: "destructive",
};

const COLUMNS: readonly Column<ModelAccessRow>[] = [
  {
    key: "provider",
    header: "Provider",
    primary: "provider",
    sort: "provider",
  },
  {
    key: "model",
    header: "Model",
    primary: "model",
    sort: "model",
  },
  {
    key: "allowed",
    header: "Status",
    primary: "allowed",
    render: (row) => (
      <Badge variant={row.allowed ? "default" : "destructive"} className="gap-1">
        {row.allowed ? <Check className="h-3 w-3" /> : <X className="h-3 w-3" />}
        {row.allowed ? "Allowed" : "Blocked"}
      </Badge>
    ),
    align: "left",
    sort: "allowed",
  },
  {
    key: "source",
    header: "Source",
    primary: "source",
    render: (row) => (
      <Badge variant={SOURCE_BADGE_VARIANTS[row.source] || "outline"}>
        {row.source === "allowlist" ? "Allowlist" : row.source === "blocklist" ? "Blocklist" : row.source}
      </Badge>
    ),
    align: "left",
    sort: "source",
  },
  {
    key: "effective",
    header: "Effective",
    primary: "effective",
    render: (row) => (
      <Badge variant={row.effective ? "default" : "secondary"}>
        {row.effective ? "Yes" : "No"}
      </Badge>
    ),
    align: "left",
    sort: "effective",
  },
  {
    key: "priority",
    header: "Priority",
    primary: "priority",
    render: (row) => row.priority.toString(),
    align: "right",
    sort: "priority",
  },
  {
    key: "policyId",
    header: "Policy ID",
    primary: "policyId",
    render: (row) => <span className="font-mono text-xs">{row.policyId.slice(0, 8)}...</span>,
    align: "left",
  },
];

export default function ModelAccessPage() {
  const [search, setSearch] = useState("");
  const [providerFilter, setProviderFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState<"all" | "allowed" | "blocked">("all");
  const [sourceFilter, setSourceFilter] = useState<"all" | "allowlist" | "blocklist">("all");
  const [entries, setEntries] = useState<ModelAccessEntryAPI[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [previewOpen, setPreviewOpen] = useState(false);
  const [previewData, setPreviewData] = useState<PreviewResponseDisplay | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [pendingChanges, setPendingChanges] = useState<UpdateModelAccessRequest["entries"]>([]);
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);
  const [confirmDialogOpen, setConfirmDialogOpen] = useState(false);
  const [confirmMessage, setConfirmMessage] = useState("");
  const [confirmAction, setConfirmAction] = useState<() => void>(() => {});

  const providers = [...new Set(entries.map((e) => e.provider).filter(Boolean))].sort() as string[];

  const fetchModelAccess = useCallback(async () => {
    setLoading(true);
    try {
      const items = await fetchModelAccessAction();
      setEntries(items);
    } catch (err) {
      console.error("Failed to fetch model access:", err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchModelAccess();
  }, [fetchModelAccess]);

  const handlePreview = async (provider: string, model: string, action: "allow" | "block") => {
    setPreviewLoading(true);
    try {
      const result = await previewModelAccessAction({ provider, model, action });
      setPreviewData(toPreviewResponse(result));
      setPreviewOpen(true);
    } catch (err) {
      console.error("Failed to preview model access:", err);
    } finally {
      setPreviewLoading(false);
    }
  };

  const handleBulkAction = (action: "allow" | "block") => {
    const filtered = entries.filter((entry) => {
      const matchesSearch =
        entry.provider?.toLowerCase().includes(search.toLowerCase()) ||
        entry.model?.toLowerCase().includes(search.toLowerCase());
      const matchesProvider = !providerFilter || entry.provider === providerFilter;
      const matchesStatus = statusFilter === "all" || (statusFilter === "allowed" ? entry.allowed : !entry.allowed);
      const matchesSource = sourceFilter === "all" || entry.source === sourceFilter;
      return matchesSearch && matchesProvider && matchesStatus && matchesSource;
    });

    const changes: UpdateModelAccessRequest["entries"] = filtered
      .filter((entry) => {
        const currentAllowed = entry.allowed ?? false;
        return (action === "allow" && !currentAllowed) || (action === "block" && currentAllowed);
      })
      .map((entry) => ({
        provider: entry.provider ?? "",
        model: entry.model ?? "",
        action,
      }));

    if (changes.length === 0) {
      alert(`All filtered models are already ${action}ed.`);
      return;
    }

    setConfirmMessage(
      `This will ${action} ${changes.length} model${changes.length > 1 ? "s" : ""}. Are you sure?`
    );
    setConfirmAction(() => {
      setPendingChanges((prev) => {
        const merged = [...prev];
        changes.forEach((change) => {
          const idx = merged.findIndex((m) => m.provider === change.provider && m.model === change.model);
          if (idx >= 0) {
            merged[idx] = change;
          } else {
            merged.push(change);
          }
        });
        return merged;
      });
      setHasUnsavedChanges(true);
    });
    setConfirmDialogOpen(true);
  };

  const handleSave = async () => {
    if (pendingChanges.length === 0) return;
    setSaving(true);
    try {
      await updateModelAccessAction({ entries: pendingChanges });
      setPendingChanges([]);
      setHasUnsavedChanges(false);
      await fetchModelAccess();
    } catch (err) {
      console.error("Failed to save model access:", err);
      alert("Failed to save changes");
    } finally {
      setSaving(false);
    }
  };

  const handleDiscard = () => {
    setPendingChanges([]);
    setHasUnsavedChanges(false);
  };

  const handleRowAction = (entry: ModelAccessEntryAPI) => {
    const newAction = entry.allowed ? "block" : "allow";
    handlePreview(entry.provider ?? "", entry.model ?? "", newAction);
  };

  const filteredEntries = entries.filter((entry) => {
    const matchesSearch =
      entry.provider?.toLowerCase().includes(search.toLowerCase()) ||
      entry.model?.toLowerCase().includes(search.toLowerCase());
    const matchesProvider = !providerFilter || entry.provider === providerFilter;
    const matchesStatus = statusFilter === "all" || (statusFilter === "allowed" ? entry.allowed : !entry.allowed);
    const matchesSource = sourceFilter === "all" || entry.source === sourceFilter;
    return matchesSearch && matchesProvider && matchesStatus && matchesSource;
  });

  const rows: ModelAccessRow[] = filteredEntries.map(toRow);

  const allowedCount = entries.filter((e) => e.allowed).length;
  const blockedCount = entries.filter((e) => !e.allowed).length;

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Model access"
        description="Configure which models are allowed or blocked for your organization."
        actions={
          <div className="flex items-center gap-2">
            {hasUnsavedChanges && (
              <>
                <Button variant="outline" onClick={handleDiscard}>
                  Discard changes
                </Button>
                <Button onClick={handleSave} disabled={saving}>
                  {saving ? "Saving..." : "Save changes"}
                </Button>
              </>
            )}
            <Dialog open={previewOpen} onOpenChange={setPreviewOpen}>
              <DialogTrigger asChild>
                <Button variant="outline" disabled={previewLoading}>
                  <Eye className="h-4 w-4 mr-2" /> Preview
                </Button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-[600px]">
                <DialogHeader>
                  <DialogTitle>Preview model access change</DialogTitle>
                  <DialogDescription>
                    See how this change would affect model access before applying.
                  </DialogDescription>
                </DialogHeader>
                {previewData && (
                  <div className="space-y-4 py-4">
                    <div className="grid gap-4">
                      <div>
                        <Label>Provider</Label>
                        <p className="font-mono">{previewData.provider}</p>
                      </div>
                      <div>
                        <Label>Model</Label>
                        <p className="font-mono">{previewData.model}</p>
                      </div>
                      <div className="grid gap-2">
                        <Label>Current decision</Label>
                        <Badge variant={previewData.currentDecision === "allow" ? "default" : "destructive"}>
                          {previewData.currentDecision === "allow" ? "Allowed" : "Blocked"}
                        </Badge>
                      </div>
                      <div className="grid gap-2">
                        <Label>Preview decision</Label>
                        <Badge variant={previewData.previewDecision === "allow" ? "default" : "destructive"}>
                          {previewData.previewDecision === "allow" ? "Allowed" : "Blocked"}
                        </Badge>
                      </div>
                      <div className="grid gap-2">
                        <Label>Would change</Label>
                        <Badge variant={previewData.wouldChange ? "default" : "secondary"}>
                          {previewData.wouldChange ? "Yes" : "No"}
                        </Badge>
                      </div>
                    </div>
                    <Separator />
                    {previewData.blockingRules.length > 0 && (
                      <div className="space-y-2">
                        <Label className="text-destructive">Blocking rules</Label>
                        <ul className="text-sm space-y-1">
                          {previewData.blockingRules.map((rule, i) => (
                            <li key={i} className="flex items-center gap-2 text-muted-foreground">
                              <X className="h-3 w-3 text-destructive" />
                              {rule.provider}/{rule.model} (priority {rule.priority})
                            </li>
                          ))}
                        </ul>
                      </div>
                    )}
                    {previewData.allowingRules.length > 0 && (
                      <div className="space-y-2">
                        <Label className="text-green-600">Allowing rules</Label>
                        <ul className="text-sm space-y-1">
                          {previewData.allowingRules.map((rule, i) => (
                            <li key={i} className="flex items-center gap-2 text-muted-foreground">
                              <Check className="h-3 w-3 text-green-600" />
                              {rule.provider}/{rule.model} (priority {rule.priority})
                            </li>
                          ))}
                        </ul>
                      </div>
                    )}
                  </div>
                )}
                <DialogFooter>
                  <Button variant="outline" onClick={() => setPreviewOpen(false)}>
                    Close
                  </Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
            <Dialog open={confirmDialogOpen} onOpenChange={setConfirmDialogOpen}>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle><AlertTriangle className="h-5 w-5 text-yellow-600 mr-2" /> Confirm action</DialogTitle>
                  <DialogDescription>{confirmMessage}</DialogDescription>
                </DialogHeader>
                <DialogFooter>
                  <Button variant="outline" onClick={() => setConfirmDialogOpen(false)}>
                    Cancel
                  </Button>
                  <Button variant="destructive" onClick={() => { confirmAction(); setConfirmDialogOpen(false); }}>
                    Confirm
                  </Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>
        }
      >
        <div className="grid gap-4 sm:grid-cols-3">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total models</CardTitle>
              <Badge variant="outline">{entries.length}</Badge>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold">{entries.length}</div>
              <p className="text-xs text-muted-foreground">Unique provider/model combinations</p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Allowed</CardTitle>
              <Badge variant="default">{allowedCount}</Badge>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold text-green-600">{allowedCount}</div>
              <p className="text-xs text-muted-foreground">Models explicitly allowed</p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Blocked</CardTitle>
              <Badge variant="destructive">{blockedCount}</Badge>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold text-red-600">{blockedCount}</div>
              <p className="text-xs text-muted-foreground">Models explicitly blocked</p>
            </CardContent>
          </Card>
        </div>

        <Card className="mt-4">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              Model access matrix
              <Badge variant="outline">{filteredEntries.length} / {entries.length}</Badge>
            </CardTitle>
            <CardDescription>
              Manage model allowlist and blocklist rules. Changes are saved to the "Model Access Control" policy.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex flex-col sm:flex-row gap-4">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                <Input
                  placeholder="Search provider or model..."
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="pl-10"
                />
              </div>
              <Select value={providerFilter} onValueChange={setProviderFilter}>
                <SelectTrigger className="w-full sm:w-[180px]">
                  <SelectValue placeholder="All providers" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">All providers</SelectItem>
                  {providers.map((provider) => (
                    <SelectItem key={provider} value={provider}>
                      {provider}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v as "all" | "allowed" | "blocked")}>
                <SelectTrigger className="w-full sm:w-[160px]">
                  <SelectValue placeholder="All statuses" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All statuses</SelectItem>
                  <SelectItem value="allowed">Allowed</SelectItem>
                  <SelectItem value="blocked">Blocked</SelectItem>
                </SelectContent>
              </Select>
              <Select value={sourceFilter} onValueChange={(v) => setSourceFilter(v as "all" | "allowlist" | "blocklist")}>
                <SelectTrigger className="w-full sm:w-[160px]">
                  <SelectValue placeholder="All sources" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All sources</SelectItem>
                  <SelectItem value="allowlist">Allowlist</SelectItem>
                  <SelectItem value="blocklist">Blocklist</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="flex gap-2">
              <Button variant="outline" onClick={() => handleBulkAction("allow")} disabled={filteredEntries.length === 0}>
                <Check className="h-4 w-4 mr-2" /> Bulk allow
              </Button>
              <Button variant="outline" onClick={() => handleBulkAction("block")} disabled={filteredEntries.length === 0}>
                <X className="h-4 w-4 mr-2" /> Bulk block
              </Button>
            </div>

            <DataTable
              columns={COLUMNS}
              rows={rows}
              rowKey="id"
              loading={loading}
              emptyMessage="No model access rules configured."
              actions={(row) => (
                <Button variant="outline" size="sm" onClick={() => handleRowAction(entries.find(e => e.provider === row.provider && e.model === row.model)!)}>
                  {row.allowed ? "Block" : "Allow"}
                </Button>
              )}
            />

            {filteredEntries.length === 0 && entries.length > 0 && (
              <div className="text-center py-8 text-muted-foreground">
                No models match the current filters.
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>How model access works</CardTitle>
            <CardDescription>
              Model access is enforced on the gateway hot path via compiled policy snapshots.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-sm text-muted-foreground">
            <p>
              The gateway consults a compiled, immutable policy snapshot on every request. That snapshot includes model allowlist and blocklist rules drawn from the
              <code className="font-mono">policies</code> and <code className="font-mono">policy_rules</code> tables. A blocked model receives a 403 before any upstream call is made.
            </p>
            <p>
              Because enforcement runs from the snapshot rather than live database reads, a grant or revocation takes effect when the snapshot is next rebuilt (typically within 5 minutes), not at the instant a row is changed.
            </p>
            <p>
              <strong>Allowlist mode:</strong> Only explicitly allowed models can be accessed. All others are denied.
            </p>
            <p>
              <strong>Blocklist mode:</strong> Explicitly blocked models are denied. All others are allowed.
            </p>
            <p>
              <strong>Priority:</strong> Higher priority rules take precedence. The "Model Access Control" policy has priority 50.
            </p>
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}