"use client";

import * as React from "react";
import {
  ShieldAlert,
  ShieldCheck,
  Plus,
  Play,
  Trash2,
  Edit2,
  Search,
  Filter,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  Radio,
  FileText,
  Clock,
  Sparkles,
  ToggleLeft,
  ToggleRight,
} from "lucide-react";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Textarea } from "@/components/ui/textarea";
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { StatCard } from "@/components/dashboard/stat-card";
import {
  createGuardrailAction,
  deleteGuardrailAction,
  testGuardrailAction,
  updateGuardrailAction,
} from "./actions";
import type {
  CreateGuardrailInput,
  Guardrail,
  GuardrailAction,
  GuardrailDetector,
  GuardrailEvent,
  GuardrailTargetType,
  TestGuardrailResponse,
} from "@ai-governance/api-client";
import { toast } from "sonner";
import { formatRelative } from "@/lib/format";

interface GuardrailsClientProps {
  initialGuardrails: Guardrail[];
  initialEvents: GuardrailEvent[];
}

export function GuardrailsClient({
  initialGuardrails,
  initialEvents,
}: GuardrailsClientProps) {
  const [guardrails, setGuardrails] = React.useState<Guardrail[]>(initialGuardrails);
  const [events, setEvents] = React.useState<GuardrailEvent[]>(initialEvents);
  const [search, setSearch] = React.useState("");
  const [detectorFilter, setDetectorFilter] = React.useState<string>("all");
  const [actionFilter, setActionFilter] = React.useState<string>("all");

  // Create Modal State
  const [isCreateOpen, setIsCreateOpen] = React.useState(false);
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  const [name, setName] = React.useState("");
  const [description, setDescription] = React.useState("");
  const [targetType, setTargetType] = React.useState<GuardrailTargetType>("global");
  const [targetId, setTargetId] = React.useState("");
  const [detector, setDetector] = React.useState<GuardrailDetector>("pii");
  const [action, setAction] = React.useState<GuardrailAction>("block");
  const [piiTypes, setPiiTypes] = React.useState<string[]>(["email", "phone", "ssn"]);
  const [secretTypes, setSecretTypes] = React.useState({
    detect_bearer_tokens: true,
    detect_api_keys: true,
    detect_private_keys: true,
  });
  const [keywords, setKeywords] = React.useState("confidential, internal-only, secret");
  const [maxTokens, setMaxTokens] = React.useState("4000");

  // Test Modal State
  const [testTarget, setTestTarget] = React.useState<Guardrail | null>(null);
  const [testPayload, setTestPayload] = React.useState("My contact is support@acme.com and SSN is 000-12-3456");
  const [testResult, setTestResult] = React.useState<TestGuardrailResponse | null>(null);
  const [isTesting, setIsTesting] = React.useState(false);

  // Edit Modal State
  const [editTarget, setEditTarget] = React.useState<Guardrail | null>(null);
  const [editName, setEditName] = React.useState("");
  const [editDescription, setEditDescription] = React.useState("");
  const [editAction, setEditAction] = React.useState<GuardrailAction>("block");

  // Stats calculation
  const totalGuardrails = guardrails.length;
  const activeGuardrails = guardrails.filter((g) => g.enabled).length;
  const blockGuardrails = guardrails.filter((g) => g.action === "block").length;
  const totalInterventions = events.length;

  const filteredGuardrails = guardrails.filter((g) => {
    const matchesSearch =
      g.name.toLowerCase().includes(search.toLowerCase()) ||
      g.description.toLowerCase().includes(search.toLowerCase());
    const matchesDetector = detectorFilter === "all" || g.detector === detectorFilter;
    const matchesAction = actionFilter === "all" || g.action === actionFilter;
    return matchesSearch && matchesDetector && matchesAction;
  });

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      toast.error("Please enter a name for the guardrail");
      return;
    }
    setIsSubmitting(true);

    let config: Record<string, unknown> = {};
    if (detector === "pii") {
      config = { types: piiTypes };
    } else if (detector === "secrets") {
      config = secretTypes;
    } else if (detector === "unsafe_keywords") {
      config = {
        keywords: keywords
          .split(",")
          .map((k) => k.trim())
          .filter(Boolean),
      };
    } else if (detector === "request_constraints") {
      config = { max_tokens: parseInt(maxTokens, 10) || 4000 };
    }

    const payload: CreateGuardrailInput = {
      name,
      description,
      target_type: targetType,
      target_id: targetId.trim() ? targetId.trim() : null,
      detector,
      action,
      configuration: config,
      enabled: true,
    };

    const res = await createGuardrailAction(payload);
    setIsSubmitting(false);

    if (res.success && res.data) {
      toast.success("Guardrail created successfully");
      setGuardrails([res.data, ...guardrails]);
      setIsCreateOpen(false);
      setName("");
      setDescription("");
      setTargetId("");
    } else {
      toast.error(res.error || "Failed to create guardrail");
    }
  };

  const handleToggleEnabled = async (g: Guardrail) => {
    const updated = {
      name: g.name,
      description: g.description,
      action: g.action,
      configuration: g.configuration,
      enabled: !g.enabled,
    };
    const res = await updateGuardrailAction(g.id, updated);
    if (res.success && res.data) {
      toast.success(`Guardrail ${!g.enabled ? "enabled" : "disabled"}`);
      setGuardrails(guardrails.map((item) => (item.id === g.id ? res.data! : item)));
    } else {
      toast.error(res.error || "Failed to update guardrail");
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Are you sure you want to delete this guardrail?")) return;
    const res = await deleteGuardrailAction(id);
    if (res.success) {
      toast.success("Guardrail deleted");
      setGuardrails(guardrails.filter((g) => g.id !== id));
    } else {
      toast.error(res.error || "Failed to delete guardrail");
    }
  };

  const handleRunTest = async () => {
    if (!testTarget) return;
    setIsTesting(true);
    setTestResult(null);

    const res = await testGuardrailAction(testTarget.id, testPayload);
    setIsTesting(false);

    if (res.success && res.data) {
      setTestResult(res.data);
      if (res.data.triggered) {
        toast.warning(`Detector triggered: ${res.data.matched_category || "match detected"}`);
      } else {
        toast.success("Payload passed all checks cleanly");
      }
    } else {
      toast.error(res.error || "Test execution failed");
    }
  };

  const handleOpenEdit = (g: Guardrail) => {
    setEditTarget(g);
    setEditName(g.name);
    setEditDescription(g.description);
    setEditAction(g.action);
  };

  const handleSaveEdit = async () => {
    if (!editTarget) return;
    setIsSubmitting(true);
    const res = await updateGuardrailAction(editTarget.id, {
      name: editName,
      description: editDescription,
      action: editAction,
      configuration: editTarget.configuration,
      enabled: editTarget.enabled,
    });
    setIsSubmitting(false);

    if (res.success && res.data) {
      toast.success("Guardrail updated");
      setGuardrails(guardrails.map((g) => (g.id === editTarget.id ? res.data! : g)));
      setEditTarget(null);
    } else {
      toast.error(res.error || "Failed to update guardrail");
    }
  };

  return (
    <div className="space-y-6">
      {/* Metrics Row */}
      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          label="Total Guardrails"
          value={String(totalGuardrails)}
          hint={`${activeGuardrails} active rules`}
        />
        <StatCard
          label="Enforcement Actions"
          value={String(blockGuardrails)}
          tone={blockGuardrails > 0 ? "danger" : "neutral"}
          hint="Strict blocking rules"
        />
        <StatCard
          label="Interventions Recorded"
          value={String(totalInterventions)}
          tone={totalInterventions > 0 ? "warning" : "success"}
          hint="Total triggered events"
        />
        <StatCard
          label="Active Detectors"
          value={String(new Set(guardrails.map((g) => g.detector)).size)}
          hint="Distinct detector engines"
        />
      </section>

      <Tabs defaultValue="rules" className="space-y-4">
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
          <TabsList>
            <TabsTrigger value="rules" className="gap-2">
              <ShieldCheck className="h-4 w-4" />
              Guardrail Rules ({guardrails.length})
            </TabsTrigger>
            <TabsTrigger value="events" className="gap-2">
              <ShieldAlert className="h-4 w-4" />
              Intervention Events ({events.length})
            </TabsTrigger>
          </TabsList>

          <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
            <DialogTrigger asChild>
              <Button className="gap-2">
                <Plus className="h-4 w-4" />
                Add Guardrail
              </Button>
            </DialogTrigger>
            <DialogContent className="max-w-lg">
              <DialogHeader>
                <DialogTitle>Create Guardrail Rule</DialogTitle>
                <DialogDescription>
                  Configure input and output inspection rules to prevent sensitive data leaks and unsafe prompts.
                </DialogDescription>
              </DialogHeader>

              <form onSubmit={handleCreate} className="space-y-4 py-2">
                <div className="space-y-2">
                  <Label htmlFor="g-name">Rule Name</Label>
                  <Input
                    id="g-name"
                    placeholder="e.g. Block PII & SSN"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    required
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="g-desc">Description</Label>
                  <Input
                    id="g-desc"
                    placeholder="Scans input text for sensitive personal data"
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                  />
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>Detector Engine</Label>
                    <Select
                      value={detector}
                      onValueChange={(v) => setDetector(v as GuardrailDetector)}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="pii">PII Scanner</SelectItem>
                        <SelectItem value="secrets">Secrets & Keys</SelectItem>
                        <SelectItem value="unsafe_keywords">Unsafe Keywords</SelectItem>
                        <SelectItem value="request_constraints">Token Constraints</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="space-y-2">
                    <Label>Enforcement Action</Label>
                    <Select
                      value={action}
                      onValueChange={(v) => setAction(v as GuardrailAction)}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="block">Block (403 Forbidden)</SelectItem>
                        <SelectItem value="warn">Warn (Log Event)</SelectItem>
                        <SelectItem value="redact">Redact Content</SelectItem>
                        <SelectItem value="allow">Allow (Observe)</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>Scope Target</Label>
                    <Select
                      value={targetType}
                      onValueChange={(v) => setTargetType(v as GuardrailTargetType)}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="global">Global (Entire Org)</SelectItem>
                        <SelectItem value="team">Team Scope</SelectItem>
                        <SelectItem value="project">Project Scope</SelectItem>
                        <SelectItem value="gateway_credential">Gateway Credential</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  {targetType !== "global" && (
                    <div className="space-y-2">
                      <Label>Target Scope ID</Label>
                      <Input
                        placeholder="UUID or identifier"
                        value={targetId}
                        onChange={(e) => setTargetId(e.target.value)}
                      />
                    </div>
                  )}
                </div>

                {/* Detector-specific Configuration */}
                <div className="rounded-lg border p-3 bg-muted/40 space-y-3">
                  <Label className="text-xs font-semibold uppercase text-muted-foreground">
                    Detector Parameters
                  </Label>

                  {detector === "pii" && (
                    <div className="space-y-2">
                      <p className="text-xs text-muted-foreground">
                        Select types of personal identifying information to detect:
                      </p>
                      <div className="flex flex-wrap gap-2">
                        {["email", "phone", "ssn"].map((type) => (
                          <Badge
                            key={type}
                            variant={piiTypes.includes(type) ? "default" : "outline"}
                            className="cursor-pointer"
                            onClick={() => {
                              setPiiTypes(
                                piiTypes.includes(type)
                                  ? piiTypes.filter((t) => t !== type)
                                  : [...piiTypes, type]
                              );
                            }}
                          >
                            {type.toUpperCase()}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  )}

                  {detector === "secrets" && (
                    <div className="space-y-2">
                      <p className="text-xs text-muted-foreground">
                        Select secrets to intercept:
                      </p>
                      <div className="flex flex-wrap gap-2">
                        <Badge
                          variant={secretTypes.detect_bearer_tokens ? "default" : "outline"}
                          className="cursor-pointer"
                          onClick={() =>
                            setSecretTypes((prev) => ({
                              ...prev,
                              detect_bearer_tokens: !prev.detect_bearer_tokens,
                            }))
                          }
                        >
                          Bearer Tokens
                        </Badge>
                        <Badge
                          variant={secretTypes.detect_api_keys ? "default" : "outline"}
                          className="cursor-pointer"
                          onClick={() =>
                            setSecretTypes((prev) => ({
                              ...prev,
                              detect_api_keys: !prev.detect_api_keys,
                            }))
                          }
                        >
                          API Keys
                        </Badge>
                        <Badge
                          variant={secretTypes.detect_private_keys ? "default" : "outline"}
                          className="cursor-pointer"
                          onClick={() =>
                            setSecretTypes((prev) => ({
                              ...prev,
                              detect_private_keys: !prev.detect_private_keys,
                            }))
                          }
                        >
                          Private Keys
                        </Badge>
                      </div>
                    </div>
                  )}

                  {detector === "unsafe_keywords" && (
                    <div className="space-y-2">
                      <Label htmlFor="keywords-input" className="text-xs">
                        Comma-separated keywords to disallow:
                      </Label>
                      <Input
                        id="keywords-input"
                        value={keywords}
                        onChange={(e) => setKeywords(e.target.value)}
                        placeholder="e.g. confidential, proprietary, leak"
                      />
                    </div>
                  )}

                  {detector === "request_constraints" && (
                    <div className="space-y-2">
                      <Label htmlFor="tokens-input" className="text-xs">
                        Max Allowed Token Count:
                      </Label>
                      <Input
                        id="tokens-input"
                        type="number"
                        value={maxTokens}
                        onChange={(e) => setMaxTokens(e.target.value)}
                        placeholder="4000"
                      />
                    </div>
                  )}
                </div>

                <DialogFooter className="pt-2">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => setIsCreateOpen(false)}
                  >
                    Cancel
                  </Button>
                  <Button type="submit" disabled={isSubmitting}>
                    {isSubmitting ? "Creating..." : "Create Guardrail"}
                  </Button>
                </DialogFooter>
              </form>
            </DialogContent>
          </Dialog>
        </div>

        {/* Tab 1: Guardrail Rules */}
        <TabsContent value="rules" className="space-y-4">
          {/* Filter Bar */}
          <div className="flex flex-col sm:flex-row gap-3">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search guardrail rules..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-9"
              />
            </div>
            <Select value={detectorFilter} onValueChange={setDetectorFilter}>
              <SelectTrigger className="w-[180px]">
                <SelectValue placeholder="Detector" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Detectors</SelectItem>
                <SelectItem value="pii">PII Scanner</SelectItem>
                <SelectItem value="secrets">Secrets</SelectItem>
                <SelectItem value="unsafe_keywords">Keywords</SelectItem>
                <SelectItem value="request_constraints">Constraints</SelectItem>
              </SelectContent>
            </Select>
            <Select value={actionFilter} onValueChange={setActionFilter}>
              <SelectTrigger className="w-[160px]">
                <SelectValue placeholder="Action" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Actions</SelectItem>
                <SelectItem value="block">Block</SelectItem>
                <SelectItem value="warn">Warn</SelectItem>
                <SelectItem value="redact">Redact</SelectItem>
                <SelectItem value="allow">Allow</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {filteredGuardrails.length === 0 ? (
            <Card>
              <CardContent className="py-12 text-center space-y-3">
                <ShieldCheck className="h-10 w-10 text-muted-foreground mx-auto" />
                <h3 className="text-lg font-medium">No Guardrail Rules Found</h3>
                <p className="text-sm text-muted-foreground max-w-sm mx-auto">
                  Protect your AI gateway requests by creating content, PII, or secrets detection rules.
                </p>
                <Button onClick={() => setIsCreateOpen(true)} className="gap-2">
                  <Plus className="h-4 w-4" />
                  Create First Guardrail
                </Button>
              </CardContent>
            </Card>
          ) : (
            <div className="grid gap-4">
              {filteredGuardrails.map((g) => (
                <Card key={g.id} className="transition-all hover:border-primary/50">
                  <CardHeader className="pb-3">
                    <div className="flex items-start justify-between gap-4">
                      <div>
                        <div className="flex items-center gap-2">
                          <CardTitle className="text-base font-semibold">
                            {g.name}
                          </CardTitle>
                          <Badge
                            variant={g.enabled ? "default" : "secondary"}
                            className="text-xs cursor-pointer"
                            onClick={() => handleToggleEnabled(g)}
                          >
                            {g.enabled ? "Active" : "Disabled"}
                          </Badge>
                          <Badge variant="outline" className="text-xs font-mono">
                            {g.detector.toUpperCase()}
                          </Badge>
                        </div>
                        {g.description && (
                          <CardDescription className="mt-1 text-sm">
                            {g.description}
                          </CardDescription>
                        )}
                      </div>

                      <div className="flex items-center gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          className="gap-1.5"
                          onClick={() => {
                            setTestTarget(g);
                            setTestResult(null);
                          }}
                        >
                          <Play className="h-3.5 w-3.5 text-blue-500" />
                          Test Rule
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleOpenEdit(g)}
                        >
                          <Edit2 className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleDelete(g.id)}
                          className="text-destructive hover:text-destructive"
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  </CardHeader>
                  <CardContent className="pt-0">
                    <div className="flex flex-wrap items-center gap-x-6 gap-y-2 text-xs text-muted-foreground border-t pt-3 mt-1">
                      <div>
                        <span className="font-medium text-foreground">Action: </span>
                        <Badge
                          variant={g.action === "block" ? "destructive" : "outline"}
                          className="ml-1 text-xs"
                        >
                          {g.action.toUpperCase()}
                        </Badge>
                      </div>
                      <div>
                        <span className="font-medium text-foreground">Scope: </span>
                        <span className="capitalize">{g.target_type}</span>
                        {g.target_id && ` (${g.target_id})`}
                      </div>
                      <div>
                        <span className="font-medium text-foreground">Config: </span>
                        <code className="text-xs bg-muted px-1.5 py-0.5 rounded">
                          {JSON.stringify(g.configuration)}
                        </code>
                      </div>
                      <div className="ml-auto text-muted-foreground">
                        Updated {formatRelative(g.updated_at, new Date())}
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </TabsContent>

        {/* Tab 2: Intervention Events */}
        <TabsContent value="events" className="space-y-4">
          {events.length === 0 ? (
            <Card>
              <CardContent className="py-12 text-center space-y-3">
                <ShieldAlert className="h-10 w-10 text-muted-foreground mx-auto" />
                <h3 className="text-lg font-medium">No Interventions Recorded</h3>
                <p className="text-sm text-muted-foreground max-w-sm mx-auto">
                  When guardrails detect sensitive data or policy violations, intervention events will appear here in real-time.
                </p>
              </CardContent>
            </Card>
          ) : (
            <Card>
              <CardHeader>
                <CardTitle>Recent Interventions & Violations</CardTitle>
                <CardDescription>
                  Audit trail of requests modified, warned, or blocked by active guardrails.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="divide-y rounded-md border">
                  {events.map((e) => (
                    <div
                      key={e.id}
                      className="p-4 flex items-center justify-between gap-4 hover:bg-muted/30 transition-colors"
                    >
                      <div className="space-y-1">
                        <div className="flex items-center gap-2">
                          <Badge
                            variant={e.action === "block" ? "destructive" : "outline"}
                            className="text-xs"
                          >
                            {e.action.toUpperCase()}
                          </Badge>
                          <span className="font-semibold text-sm">
                            Matched Category: {e.matched_category || "Sensitive Pattern"}
                          </span>
                          <span className="text-xs text-muted-foreground font-mono">
                            [{e.detector}]
                          </span>
                        </div>
                        <p className="text-xs text-muted-foreground">
                          Request ID: <code className="text-foreground">{e.request_id}</code> · Direction: <span className="capitalize">{e.direction}</span>
                        </p>
                      </div>
                      <div className="text-xs text-muted-foreground text-right">
                        {formatRelative(e.timestamp, new Date())}
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}
        </TabsContent>
      </Tabs>

      {/* Test Rule Dialog */}
      <Dialog open={!!testTarget} onOpenChange={(open) => !open && setTestTarget(null)}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Test Guardrail Detector</DialogTitle>
            <DialogDescription>
              Simulate an incoming prompt or response against &ldquo;{testTarget?.name}&rdquo;.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label>Test Prompt Payload</Label>
              <Textarea
                rows={4}
                value={testPayload}
                onChange={(e) => setTestPayload(e.target.value)}
                placeholder="Enter sample prompt containing keywords or sensitive data..."
              />
            </div>

            {testResult && (
              <div
                className={`p-4 rounded-lg border space-y-2 ${
                  testResult.triggered
                    ? "bg-destructive/10 border-destructive/30"
                    : "bg-green-500/10 border-green-500/30"
                }`}
              >
                <div className="flex items-center justify-between">
                  <span className="font-semibold text-sm">
                    {testResult.triggered
                      ? "⚠️ Guardrail Triggered!"
                      : "✅ Check Passed Cleanly"}
                  </span>
                  <Badge
                    variant={testResult.triggered ? "destructive" : "default"}
                  >
                    {testResult.triggered ? testResult.action?.toUpperCase() || "INTERCEPTED" : "ALLOWED"}
                  </Badge>
                </div>
                {testResult.triggered && (
                  <p className="text-xs text-muted-foreground">
                    Matched violation category:{" "}
                    <strong>{testResult.matched_category}</strong>
                  </p>
                )}
              </div>
            )}
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setTestTarget(null)}
            >
              Close
            </Button>
            <Button onClick={handleRunTest} disabled={isTesting}>
              {isTesting ? "Testing..." : "Execute Test"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Rule Dialog */}
      <Dialog open={!!editTarget} onOpenChange={(open) => !open && setEditTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit Guardrail Rule</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label>Rule Name</Label>
              <Input
                value={editName}
                onChange={(e) => setEditName(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label>Description</Label>
              <Input
                value={editDescription}
                onChange={(e) => setEditDescription(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label>Action</Label>
              <Select
                value={editAction}
                onValueChange={(v) => setEditAction(v as GuardrailAction)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="block">Block</SelectItem>
                  <SelectItem value="warn">Warn</SelectItem>
                  <SelectItem value="redact">Redact</SelectItem>
                  <SelectItem value="allow">Allow</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditTarget(null)}>
              Cancel
            </Button>
            <Button onClick={handleSaveEdit} disabled={isSubmitting}>
              {isSubmitting ? "Saving..." : "Save Changes"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
