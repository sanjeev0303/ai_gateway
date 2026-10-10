"use client";

import { useState } from "react";
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
import { fetchBudgetsAction, createBudgetAction, updateBudgetAction, deleteBudgetAction } from "./actions";
import type { Budget, CreateBudgetRequest, UpdateBudgetRequest, BudgetList } from "@ai-governance/api-client";

interface BudgetRow {
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

const SCOPE_OPTIONS = [
  { value: "organization", label: "Organization" },
  { value: "team", label: "Team" },
  { value: "project", label: "Project" },
  { value: "gateway_credential", label: "Gateway Credential" },
  { value: "service_account", label: "Service Account" },
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
      <div className="w-full max-w-[200px]">
        <div className="flex items-center justify-between text-xs mb-1">
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
      if (row.limitType === "spend") {
        return `$${remaining.toFixed(2)}`;
      }
      return remaining.toLocaleString();
    },
    align: "right",
  },
  {
    key: "windowStart",
    header: "Window start",
    primary: "windowStart",
    render: (row) => row.windowStart ? new Date(row.windowStart).toLocaleString() : "—",
    align: "left",
  },
  {
    key: "windowEnd",
    header: "Window end (rollover)",
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

  const fetchBudgets = async () => {
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
  };

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
    if (!confirm(`Delete budget "${budget.name}"? This cannot be undone.`)) return;
    try {
      await deleteBudgetAction(budget.id ?? "");
      await fetchBudgets();
    } catch (err) {
      console.error("Failed to delete budget:", err);
      alert("Failed to delete budget");
    }
  };

  const handleSubmit = async () => {
    setSubmitting(true);
    setError("");
    try {
      if (isEditing && selectedBudget) {
        await updateBudgetAction(selectedBudget.id ?? "", formData);
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
    scope: budget.scope ?? "organization",
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
            <DialogContent className="sm:max-w-[500px]">
              <DialogHeader>
                <DialogTitle>{isEditing ? "Edit budget" : "Create budget"}</DialogTitle>
                <DialogDescription>
                  Define a spend, token, or request cap. Budgets are enforced on every request.
                </DialogDescription>
              </DialogHeader>
              <form onSubmit={(e) => { e.preventDefault(); handleSubmit(); }}>
                <div className="grid gap-4 py-4">
                  <div className="grid gap-2">
                    <Label htmlFor="name">Name *</Label>
                    <Input
                      id="name"
                      value={formData.name}
                      onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                      placeholder="e.g., Monthly spend cap"
                      required
                    />
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="description">Description</Label>
                    <Textarea
                      id="description"
                      value={formData.description}
                      onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                      placeholder="Optional description"
                      rows={2}
                    />
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="limit_type">Limit type *</Label>
                    <Select
                      value={formData.limit_type}
                      onValueChange={(v) => setFormData({ ...formData, limit_type: v as any })}
                    >
                      <SelectTrigger>
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
                    <Label htmlFor="limit_amount">Limit amount *</Label>
                    <Input
                      id="limit_amount"
                      type="number"
                      step={formData.limit_type === "spend" ? "0.01" : "1"}
                      min="0.01"
                      value={formData.limit_amount}
                      onChange={(e) => setFormData({ ...formData, limit_amount: parseFloat(e.target.value) || 0 })}
                      placeholder={formData.limit_type === "spend" ? "1000.00" : "100000"}
                      required
                    />
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="window">Time window *</Label>
                    <Select
                      value={formData.window}
                      onValueChange={(v) => setFormData({ ...formData, window: v as any })}
                    >
                      <SelectTrigger>
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
                  <div className="grid gap-2">
                    <Label htmlFor="scope">Scope *</Label>
                    <Select
                      value={formData.scope}
                      onValueChange={(v) => setFormData({ ...formData, scope: v as any })}
                    >
                      <SelectTrigger>
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
                  <div className="grid gap-2">
                    <Label htmlFor="scope_id">Scope ID</Label>
                    <Input
                      id="scope_id"
                      value={formData.scope_id}
                      onChange={(e) => setFormData({ ...formData, scope_id: e.target.value })}
                      placeholder="UUID (required for team, project, credential, service_account)"
                    />
                    <p className="text-xs text-muted-foreground">
                      Required when scope is not "organization"
                    </p>
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="alert_threshold">Alert threshold *</Label>
                    <Input
                      id="alert_threshold"
                      type="number"
                      step="0.05"
                      min="0"
                      max="1"
                      value={formData.alert_threshold}
                      onChange={(e) => setFormData({ ...formData, alert_threshold: parseFloat(e.target.value) || 0 })}
                      placeholder="0.8"
                      required
                    />
                    <p className="text-xs text-muted-foreground">
                      Alert when {((formData.alert_threshold ?? 0) * 100).toFixed(0)}% of limit is reached
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="enabled"
                      checked={formData.enabled}
                      onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
                      className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
                    />
                    <Label htmlFor="enabled" className="cursor-pointer">
                      Enabled
                    </Label>
                  </div>
                  {error && (
                    <div className="p-3 text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg">
                      {error}
                    </div>
                  )}
                </div>
                <DialogFooter>
                  <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
                    Cancel
                  </Button>
                  <Button type="submit" disabled={submitting}>
                    {submitting ? "Saving..." : isEditing ? "Update" : "Create"}
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
                <SelectTrigger className="w-full sm:w-[180px]">
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
                <SelectTrigger className="w-full sm:w-[160px]">
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
                <SelectTrigger className="w-full sm:w-[140px]">
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
              actions={(row) => (
                <div className="flex justify-end gap-2">
                  <Button variant="outline" size="sm" onClick={() => handleEdit(budgets.find(b => b.id === row.id)!)}>Edit</Button>
                  <Button variant="destructive" size="sm" onClick={() => handleDelete(budgets.find(b => b.id === row.id)!)}>Delete</Button>
                </div>
              )}
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