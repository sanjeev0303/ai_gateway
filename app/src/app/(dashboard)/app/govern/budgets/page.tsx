"use client";

import { useState, useEffect, useCallback } from "react";
import type { Metadata } from "next";
import { Plus, Search, Filter, Trash2, Edit, AlertTriangle, ChevronDown, ChevronUp } from "lucide-react";

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
import { Textarea } from "@/components/ui/textarea";
import { Progress } from "@/components/ui/progress";
import { confirmAction } from "@/components/dashboard/confirm-dialog";
import { loadDirectory } from "@/components/dashboard/directory-actions";
import { SearchSelect } from "@/components/dashboard/search-select";
import type { Directory, DirectoryEntry } from "@/lib/directory";
import { toast } from "sonner";
import { fetchBudgetsAction, createBudgetAction, updateBudgetAction, deleteBudgetAction } from "./actions";
import { BudgetActionsCell } from "./actions-cell";
import type { Budget, CreateBudgetRequest, UpdateBudgetRequest, BudgetList } from "@ai-governance/api-client";

export interface BudgetRow {
  id: string;
  name: string;
  scope: string;
  scopeId: string;
  limitType: string;
  limitAmount: number;
  window: string;
  alertThreshold: number;
  enabled: boolean;
  currentSpend: number;
  currentTokens: number;
  currentRequests: number;
  percentageUsed: number;
  windowStart: string;
  windowEnd: string;
  createdAt: string;
}

function scopeOptions(directory: Directory | null, scope: string): readonly DirectoryEntry[] {
  if (!directory) return [];
  if (scope === "team") return directory.teams;
  if (scope === "project") return directory.projects;
  if (scope === "service_account") return directory.serviceAccounts;
  if (scope === "gateway_credential") return directory.credentials;
  return [];
}

const SCOPE_OPTIONS = [
  { value: "organization", label: "Organization" },
  { value: "team", label: "Team" },
  { value: "project", label: "Project" },
  { value: "gateway_credential", label: "Gateway key" },
  { value: "service_account", label: "Service account" },
] as const;

const LIMIT_TYPE_OPTIONS = [
  { value: "spend", label: "Spend (USD)" },
  { value: "tokens", label: "Tokens" },
  { value: "requests", label: "Requests" },
] as const;

const WINDOW_OPTIONS = [
  { value: "hourly", label: "Hourly" },
  { value: "daily", label: "Daily" },
  { value: "weekly", label: "Weekly" },
  { value: "monthly", label: "Monthly" },
] as const;

const COLUMNS: readonly Column<BudgetRow>[] = [
  {
    key: "name",
    header: "Budget",
    primary: "name",
    secondary: "scope",
    sort: "name",
  },
  {
    key: "limitType",
    header: "Type",
    primary: "limitType",
    render: (row) => (
      <Badge variant="outline">
        {row.limitType === "spend" ? "$" : row.limitType === "tokens" ? "TK" : "RQ"}
        {row.limitType.charAt(0).toUpperCase() + row.limitType.slice(1)}
      </Badge>
    ),
    align: "left",
    sort: "limitType",
  },
  {
    key: "limitAmount",
    header: "Limit",
    primary: "limitAmount",
    render: (row) => {
      if (row.limitType === "spend") {
        return `$${row.limitAmount.toLocaleString()}`;
      }
      return row.limitAmount.toLocaleString();
    },
    align: "right",
    sort: "limitAmount",
  },
  {
    key: "window",
    header: "Window",
    primary: "window",
    sort: "window",
  },
  {
    key: "percentageUsed",
    header: "Usage",
    primary: "percentageUsed",
    render: (row) => (
      <div className="w-full max-w-50">
        <div className="mb-1 flex items-center justify-between gap-2 text-xs tabular-nums">
          <span>{row.percentageUsed.toFixed(1)}%</span>
          <span className="text-muted-foreground">
            {row.limitType === "spend"
              ? `$${row.currentSpend.toFixed(2)} / $${row.limitAmount.toFixed(2)}`
              : row.limitType === "tokens"
              ? `${row.currentTokens.toLocaleString()} / ${row.limitAmount.toLocaleString()}`
              : `${row.currentRequests.toLocaleString()} / ${row.limitAmount.toLocaleString()}`}
          </span>
        </div>
        <Progress
          value={Math.min(row.percentageUsed, 100)}
          aria-label={`${row.name}: ${row.percentageUsed.toFixed(0)}% of limit used`}
          className={`h-2 ${
            row.percentageUsed >= 100 ? "bg-red-500" :
            row.percentageUsed >= 80 ? "bg-yellow-500" : "bg-green-500"
          }`}
        />
      </div>
    ),
    align: "right",
  },
  {
    key: "alertThreshold",
    header: "Alert At",
    primary: "alertThreshold",
    render: (row) => `${(row.alertThreshold * 100).toFixed(0)}%`,
    align: "right",
  },
  {
    key: "percentageUsed",
    header: "Remaining",
    primary: "percentageUsed",
    render: (row) => {
      const remaining = row.limitAmount - (row.limitType === "spend" ? row.currentSpend : row.limitType === "tokens" ? row.currentTokens : row.currentRequests);
      const amount = row.limitType === "spend" ? `$${Math.abs(remaining).toFixed(2)}` : Math.abs(remaining).toLocaleString();
      if (remaining < 0) {
        return <span className="font-medium whitespace-nowrap text-rose-700 dark:text-rose-400">{amount} over limit</span>;
      }
      return amount;
    },
    align: "right",
  },
  {
    key: "windowEnd",
    header: "Resets",
    primary: "windowEnd",
    render: (row) => row.windowEnd ? new Date(row.windowEnd).toLocaleString() : "—",
    align: "left",
  },
  {
    key: "enabled",
    header: "Status",
    primary: "enabled",
    render: (row) => (
      <Badge variant={row.enabled ? "default" : "secondary"}>
        {row.enabled ? "Active" : "Disabled"}
      </Badge>
    ),
    align: "left",
    sort: "enabled",
  },
  {
    key: "createdAt",
    header: "Created",
    primary: "createdAt",
    render: (row) => new Date(row.createdAt).toLocaleDateString(),
    sort: "createdAt",
  },
];

export default function BudgetsPage() {
  const [search, setSearch] = useState("");
  const [scopeFilter, setScopeFilter] = useState("");
  const [limitTypeFilter, setLimitTypeFilter] = useState("");
  const [enabledFilter, setEnabledFilter] = useState<"all" | "true" | "false">("all");
  const [budgets, setBudgets] = useState<Budget[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedBudget, setSelectedBudget] = useState<Budget | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [formData, setFormData] = useState<CreateBudgetRequest>({
    name: "",
    description: "",
    scope: "organization",
    scope_id: "",
    limit_type: "spend",
    limit_amount: 1000,
    window: "monthly",
    alert_threshold: 0.8,
    enabled: true,
  });
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const [directory, setDirectory] = useState<Directory | null>(null);
  useEffect(() => {
    loadDirectory().then(setDirectory).catch(() => setDirectory(null));
  }, []);

  const fetchBudgets = useCallback(async () => {
    setLoading(true);
    try {
      const params: any = {};
      if (scopeFilter) params.scope = scopeFilter;
      if (limitTypeFilter) params.limit_type = limitTypeFilter;
      if (enabledFilter !== "all") params.enabled = enabledFilter === "true";
      const items = await fetchBudgetsAction(params);
      setBudgets(items);
    } catch (err) {
      console.error("Failed to fetch budgets:", err);
    } finally {
      setLoading(false);
    }
  }, [scopeFilter, limitTypeFilter, enabledFilter]);

  useEffect(() => {
    fetchBudgets();
  }, [fetchBudgets]);

  const handleCreate = async () => {
    setIsEditing(false);
    setFormData({
      name: "",
      description: "",
      scope: "organization",
      scope_id: "",
      limit_type: "spend",
      limit_amount: 1000,
      window: "monthly",
      alert_threshold: 0.8,
      enabled: true,
    });
    setError("");
    setDialogOpen(true);
  };

  const handleEdit = (budget: Budget) => {
    setIsEditing(true);
    setSelectedBudget(budget);
    setFormData({
      name: budget.name ?? "",
      description: budget.description ?? "",
      scope: budget.scope ?? "organization",
      scope_id: budget.scope_id ?? "",
      limit_type: budget.limit_type ?? "spend",
      limit_amount: budget.limit_amount ?? 0,
      window: budget.window ?? "monthly",
      alert_threshold: budget.alert_threshold ?? 0.8,
      enabled: budget.enabled ?? true,
    });
    setError("");
    setDialogOpen(true);
  };

  const handleDelete = async (budget: Budget) => {
    const confirmed = await confirmAction({
      title: `Delete ${budget.name}?`,
      description: "The gateway stops enforcing this limit straight away. This cannot be undone.",
      confirmLabel: "Delete budget",
      destructive: true,
    });
    if (!confirmed) return;
    try {
      await deleteBudgetAction(budget.id ?? "");
      await fetchBudgets();
    } catch (err) {
      console.error("Failed to delete budget:", err);
      toast.error("The budget was not deleted. Check your connection and try again.");
    }
  };

  const handleSubmit = async () => {
    setSubmitting(true);
    setError("");
    try {
      if (isEditing && selectedBudget) {
        // Scope, limit type and window are fixed once a budget exists; the API
        // rejects a body that names them.
        const { name, description, limit_amount, alert_threshold, enabled } = formData;
        await updateBudgetAction(selectedBudget.id ?? "", { name, description, limit_amount, alert_threshold, enabled });
      } else {
        await createBudgetAction(formData);
      }
      setDialogOpen(false);
      await fetchBudgets();
    } catch (err: any) {
      setError(err.message || "Failed to save budget");
    } finally {
      setSubmitting(false);
    }
  };

  // Anything narrower than the organization needs a target before it can be saved.
  const scopeReady = formData.scope === "organization" || Boolean(formData.scope_id);

  const filteredBudgets = budgets.filter((budget) => {
    const nameStr = budget.name ?? "";
    const descStr = budget.description ?? "";
    const matchesSearch =
      nameStr.toLowerCase().includes(search.toLowerCase()) ||
      descStr.toLowerCase().includes(search.toLowerCase());
    const matchesScope = !scopeFilter || budget.scope === scopeFilter;
    const matchesLimitType = !limitTypeFilter || budget.limit_type === limitTypeFilter;
    const matchesEnabled = enabledFilter === "all" || budget.enabled === (enabledFilter === "true");
    return matchesSearch && matchesScope && matchesLimitType && matchesEnabled;
  });

  const rows: BudgetRow[] = filteredBudgets.map((budget) => ({
    id: budget.id ?? "",
    name: budget.name ?? "",
    scope: budget.scope === "organization" || !budget.scope
      ? "Organization"
      : (scopeOptions(directory, budget.scope).find((o) => o.id === budget.scope_id)?.name ?? budget.scope),
    scopeId: budget.scope_id ?? "",
    limitType: budget.limit_type ?? "spend",
    limitAmount: budget.limit_amount ?? 0,
    window: budget.window ?? "monthly",
    alertThreshold: budget.alert_threshold ?? 0.8,
    enabled: budget.enabled ?? true,
    currentSpend: budget.current_spend ?? 0,
    currentTokens: budget.current_tokens ?? 0,
    currentRequests: budget.current_requests ?? 0,
    percentageUsed: budget.percentage_used ?? 0,
    windowStart: budget.window_start ?? "",
    windowEnd: budget.window_end ?? "",
    createdAt: budget.created_at ?? "",
  }));

  function BudgetActionsCellWrapper({ row }: { row: BudgetRow }) {
    return <BudgetActionsCell row={row} budgets={budgets} onEdit={handleEdit} onDelete={handleDelete} />;
  }

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Budgets"
        description="Spend, token, and request caps enforced on the gateway request path."
        actions={
          <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
            <DialogTrigger asChild>
              <Button onClick={handleCreate}>
                <Plus aria-hidden /> New budget
              </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-125">
              <DialogHeader>
                <DialogTitle>{isEditing ? "Edit budget" : "Create budget"}</DialogTitle>
                <DialogDescription>
                  Define a spend, token, or request cap. Budgets are enforced on every request.
                </DialogDescription>
              </DialogHeader>
              <form onSubmit={(e) => { e.preventDefault(); handleSubmit(); }}>
                <div className="flex flex-col gap-5 pb-5">
                  <div className="grid gap-2">
                    <Label htmlFor="name">Name</Label>
                    <Input
                      id="name"
                      value={formData.name}
                      onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                      placeholder="Monthly spend cap…"
                      autoComplete="off"
                      required
                    />
                  </div>

                  <fieldset className="grid gap-3">
                    <legend className="mb-2 text-sm font-semibold">The limit</legend>
                    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
                      <div className="grid gap-2">
                        <Label htmlFor="limit_type">Measure</Label>
                        <Select
                          value={formData.limit_type}
                          disabled={isEditing}
                          onValueChange={(v) => setFormData({ ...formData, limit_type: v as any })}
                        >
                          <SelectTrigger id="limit_type" className="w-full">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {LIMIT_TYPE_OPTIONS.map((opt) => (
                              <SelectItem key={opt.value} value={opt.value}>
                                {opt.label}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                      <div className="grid gap-2">
                        <Label htmlFor="limit_amount">Amount</Label>
                        <Input
                          id="limit_amount"
                          type="number"
                          inputMode="decimal"
                          step={formData.limit_type === "spend" ? "0.01" : "1"}
                          min={formData.limit_type === "spend" ? "0.01" : "1"}
                          value={formData.limit_amount}
                          onChange={(e) => setFormData({ ...formData, limit_amount: parseFloat(e.target.value) || 0 })}
                          required
                        />
                      </div>
                      <div className="col-span-2 grid gap-2 sm:col-span-1">
                        <Label htmlFor="window">Every</Label>
                        <Select
                          value={formData.window}
                          disabled={isEditing}
                          onValueChange={(v) => setFormData({ ...formData, window: v as any })}
                        >
                          <SelectTrigger id="window" className="w-full">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {WINDOW_OPTIONS.map((opt) => (
                              <SelectItem key={opt.value} value={opt.value}>
                                {opt.label}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                    </div>
                    {isEditing && (
                      <p className="text-xs text-muted-foreground">
                        Measure, window and scope are fixed once a budget exists. Create a new budget to change them.
                      </p>
                    )}
                  </fieldset>

                  <fieldset className="grid gap-3">
                    <legend className="mb-2 text-sm font-semibold">Who it applies to</legend>
                    <div className="grid gap-3 sm:grid-cols-2">
                      <div className="grid gap-2">
                        <Label htmlFor="scope">Scope</Label>
                        <Select
                          value={formData.scope}
                          disabled={isEditing}
                          onValueChange={(v) => setFormData({ ...formData, scope: v as any, scope_id: "" })}
                        >
                          <SelectTrigger id="scope" className="w-full">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {SCOPE_OPTIONS.map((opt) => (
                              <SelectItem key={opt.value} value={opt.value}>
                                {opt.label}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                      {formData.scope !== "organization" && (
                        <div className="grid gap-2">
                          <Label htmlFor="scope_id">{SCOPE_OPTIONS.find((o) => o.value === formData.scope)?.label}</Label>
                          <SearchSelect
                            id="scope_id"
                            value={formData.scope_id ?? ""}
                            disabled={isEditing}
                            onChange={(value) => setFormData({ ...formData, scope_id: value })}
                            options={scopeOptions(directory, formData.scope ?? "organization")}
                            loading={!directory}
                            placeholder="Choose…"
                          />
                        </div>
                      )}
                    </div>
                  </fieldset>

                  <fieldset className="grid gap-3">
                    <legend className="mb-2 text-sm font-semibold">Alerts</legend>
                    <div className="grid grid-cols-[minmax(0,8rem)_minmax(0,1fr)] items-center gap-3">
                      <div className="relative">
                        <Input
                          id="alert_threshold"
                          aria-label="Alert threshold in percent"
                          type="number"
                          inputMode="numeric"
                          step="5"
                          min="0"
                          max="100"
                          className="pr-8"
                          value={Math.round((formData.alert_threshold ?? 0) * 100)}
                          onChange={(e) => setFormData({ ...formData, alert_threshold: (parseFloat(e.target.value) || 0) / 100 })}
                          required
                        />
                        <span aria-hidden className="absolute top-1/2 right-3 -translate-y-1/2 text-sm text-muted-foreground">%</span>
                      </div>
                      <p className="text-sm text-muted-foreground">of the limit raises an alert.</p>
                    </div>
                    <label htmlFor="enabled" className="flex cursor-pointer items-center gap-2.5 text-sm">
                      <input
                        type="checkbox"
                        id="enabled"
                        checked={formData.enabled}
                        onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
                        className="size-4 rounded accent-(--primary)"
                      />
                      Enforce this budget
                    </label>
                  </fieldset>

                  <div className="grid gap-2">
                    <Label htmlFor="description">Note (optional)</Label>
                    <Textarea
                      id="description"
                      value={formData.description}
                      onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                      rows={2}
                    />
                  </div>

                  {error && (
                    <p role="alert" className="rounded-xl bg-destructive/10 p-3 text-sm text-destructive">
                      {error}
                    </p>
                  )}
                </div>
                <DialogFooter>
                  <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
                    Cancel
                  </Button>
                  <Button type="submit" disabled={submitting || !scopeReady}>
                    {submitting ? "Saving…" : isEditing ? "Save budget" : "Create budget"}
                  </Button>
                </DialogFooter>
              </form>
            </DialogContent>
          </Dialog>
        }
      >
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              Budget caps
              <Badge variant="outline">{budgets.length} budgets</Badge>
            </CardTitle>
            <CardDescription>
              Budgets are compiled into policy rules and enforced on the gateway hot path.
              Changes take effect when the policy snapshot is rebuilt.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex flex-col sm:flex-row gap-4">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
                <Input
                  placeholder="Search name or description..."
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="pl-10"
                />
              </div>
              <Select value={scopeFilter} onValueChange={setScopeFilter}>
                <SelectTrigger className="w-full sm:w-45" aria-label="Filter by scope">
                  <SelectValue placeholder="All scopes" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">All scopes</SelectItem>
                  {SCOPE_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Select value={limitTypeFilter} onValueChange={setLimitTypeFilter}>
                <SelectTrigger className="w-full sm:w-40" aria-label="Filter by type">
                  <SelectValue placeholder="All types" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">All types</SelectItem>
                  {LIMIT_TYPE_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Select value={enabledFilter} onValueChange={(v) => setEnabledFilter(v as any)}>
                <SelectTrigger className="w-full sm:w-35" aria-label="Filter by status">
                  <SelectValue placeholder="All statuses" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All statuses</SelectItem>
                  <SelectItem value="true">Active</SelectItem>
                  <SelectItem value="false">Disabled</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <DataTable
              columns={COLUMNS}
              rows={rows}
              rowKey="id"
              loading={loading}
              emptyMessage="No budgets configured. Create one to get started."
              ActionsComponent={BudgetActionsCellWrapper}
            />

            {filteredBudgets.length === 0 && budgets.length > 0 && (
              <div className="text-center py-8 text-muted-foreground">
                No budgets match the current filters.
              </div>
            )}
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
