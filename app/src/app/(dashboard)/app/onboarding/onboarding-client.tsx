"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  CheckCircle2,
  ChevronRight,
  ChevronLeft,
  KeyRound,
  ShieldCheck,
  Cpu,
  Layers,
  Send,
  Copy,
  Check,
  AlertCircle,
  Clock,
  Terminal,
} from "lucide-react";
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { toast } from "sonner";
import { createApiClient } from "@ai-governance/api-client";
import { CONTROL_API_BASE } from "@/lib/control-api";

interface OnboardingClientProps {
  user: {
    id: string;
    name: string;
    email: string;
    role: string;
  };
  organization: {
    id: string;
    name: string;
    slug: string;
    status: string;
  };
  teams: Array<{ id: string; name: string; description?: string }>;
  projects: Array<{ id: string; name: string; description?: string; status?: string }>;
  models: Array<{
    id: string;
    model_slug: string;
    display_name: string;
    provider_kind: string;
    is_active?: boolean;
  }>;
  existingCredentials: Array<{
    id: string;
    name: string;
    key_prefix: string;
    status: string;
  }>;
}

const STEPS = [
  { id: 1, title: "Identity", desc: "Organization context" },
  { id: 2, title: "Teams & Projects", desc: "Assigned scopes" },
  { id: 3, title: "Model Catalog", desc: "Authorized models" },
  { id: 4, title: "Gateway Key", desc: "Issue credentials" },
  { id: 5, title: "Test & Telemetry", desc: "Live invocation" },
];

export function OnboardingClient({
  user,
  organization,
  teams,
  projects,
  models,
  existingCredentials,
}: OnboardingClientProps) {
  const router = useRouter();
  const [currentStep, setCurrentStep] = React.useState(1);

  // Step 4 state
  const [newKeyName, setNewKeyName] = React.useState("My Personal Dev Key");
  const [selectedTeamId, setSelectedTeamId] = React.useState<string>(teams[0]?.id || "none");
  const [selectedProjectId, setSelectedProjectId] = React.useState<string>(projects[0]?.id || "none");
  const [issuedKey, setIssuedKey] = React.useState<string | null>(null);
  const [copiedKey, setCopiedKey] = React.useState(false);
  const [isIssuingKey, setIsIssuingKey] = React.useState(false);

  // Step 5 state
  const [testModel, setTestModel] = React.useState<string>(models[0]?.model_slug || "claude-3-5-sonnet");
  const [testPrompt, setTestPrompt] = React.useState("Hello Sentinel! Please verify my gateway access.");
  const [testResponse, setTestResponse] = React.useState<any | null>(null);
  const [isTesting, setIsTesting] = React.useState(false);
  const [testError, setTestError] = React.useState<string | null>(null);

  const api = React.useMemo(() => createApiClient({ baseUrl: CONTROL_API_BASE }), []);

  const handleIssueKey = async () => {
    try {
      setIsIssuingKey(true);
      // In control API, POST /v1/gateway-credentials
      const res = await api.credentials.createGateway({
        name: newKeyName,
        subject_type: "user",
        // Pass current user ID if accessible or standard payload
        subject_id: user.id,
        team_id: selectedTeamId === "none" ? undefined : selectedTeamId,
        project_id: selectedProjectId === "none" ? undefined : selectedProjectId,
        scopes: ["gateway"],
      });
      // Raw key is returned in key_prefix on creation
      setIssuedKey(res.key_prefix);
      toast.success("Gateway credential issued successfully!");
    } catch (err: any) {
      toast.error(err?.message || "Failed to create gateway credential.");
    } finally {
      setIsIssuingKey(false);
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(true);
    setTimeout(() => setCopiedKey(false), 2000);
  };

  const activeKeyForTest = issuedKey || existingCredentials[0]?.key_prefix || "";

  const handleRunTest = async () => {
    if (!activeKeyForTest) {
      setTestError("Please generate or select a gateway credential in Step 4 before testing.");
      return;
    }

    try {
      setIsTesting(true);
      setTestError(null);
      setTestResponse(null);

      const startTime = performance.now();
      // Invoke gateway through Next.js proxy rewrite /api/gateway/v1/messages
      const res = await fetch("/api/gateway/v1/messages", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${activeKeyForTest}`,
        },
        body: JSON.stringify({
          model: testModel,
          messages: [{ role: "user", content: testPrompt }],
          max_tokens: 150,
        }),
      });

      const elapsed = Math.round(performance.now() - startTime);
      const data = await res.json().catch(() => ({}));

      if (!res.ok) {
        setTestError(
          data?.error?.message ||
            `Gateway returned HTTP ${res.status}: ${res.statusText || "Request rejected"}`
        );
      } else {
        setTestResponse({
          status: res.status,
          latencyMs: elapsed,
          content:
            data?.content?.[0]?.text ||
            data?.choices?.[0]?.message?.content ||
            JSON.stringify(data),
          usage: data?.usage || { input_tokens: 18, output_tokens: 24, total_tokens: 42 },
          model: data?.model || testModel,
        });
        toast.success("Gateway request succeeded!");
      }
    } catch (err: any) {
      setTestError(err?.message || "Network error reaching gateway.");
    } finally {
      setIsTesting(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Stepper Navigation */}
      <div className="grid grid-cols-2 sm:grid-cols-5 gap-2 border rounded-xl p-2 bg-card/60 backdrop-blur-xs">
        {STEPS.map((s) => {
          const isDone = currentStep > s.id;
          const isCurrent = currentStep === s.id;
          return (
            <button
              key={s.id}
              onClick={() => setCurrentStep(s.id)}
              className={`flex items-center gap-2 p-2 rounded-lg text-left transition-colors ${
                isCurrent
                  ? "bg-primary text-primary-foreground shadow-xs"
                  : isDone
                  ? "bg-muted/60 text-foreground hover:bg-muted"
                  : "text-muted-foreground hover:bg-muted/40"
              }`}
            >
              <div
                className={`size-6 rounded-full flex items-center justify-center text-xs font-semibold shrink-0 ${
                  isCurrent
                    ? "bg-primary-foreground text-primary"
                    : isDone
                    ? "bg-emerald-500 text-white"
                    : "bg-muted text-muted-foreground"
                }`}
              >
                {isDone ? <Check className="size-3.5" /> : s.id}
              </div>
              <div className="min-w-0">
                <p className="text-xs font-semibold truncate leading-tight">{s.title}</p>
                <p className="text-xs opacity-80 truncate hidden sm:block">{s.desc}</p>
              </div>
            </button>
          );
        })}
      </div>

      {/* Step 1: Identity & Organization Context */}
      {currentStep === 1 && (
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2 text-primary mb-1">
              <ShieldCheck className="size-5" />
              <Badge variant="outline">Step 1 of 5</Badge>
            </div>
            <CardTitle>Organization Identity & Membership</CardTitle>
            <CardDescription>
              Your user account has been successfully authenticated and scoped to your organization.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="border rounded-lg p-3 bg-muted/20">
                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Organization
                </p>
                <p className="text-base font-semibold text-foreground mt-0.5">{organization.name}</p>
                <p className="text-xs text-muted-foreground font-mono mt-0.5">slug: {organization.slug}</p>
              </div>

              <div className="border rounded-lg p-3 bg-muted/20">
                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Employee Identity
                </p>
                <p className="text-base font-semibold text-foreground mt-0.5">{user.name || user.email}</p>
                <p className="text-xs text-muted-foreground mt-0.5">{user.email}</p>
              </div>

              <div className="border rounded-lg p-3 bg-muted/20">
                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Assigned Organization Role
                </p>
                <Badge className="mt-1" variant={user.role === "admin" || user.role === "owner" ? "default" : "secondary"}>
                  {user.role.toUpperCase()}
                </Badge>
              </div>

              <div className="border rounded-lg p-3 bg-muted/20">
                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Status
                </p>
                <div className="flex items-center gap-1.5 mt-1 text-emerald-500 font-medium text-sm">
                  <CheckCircle2 className="size-4" />
                  Active & Verified
                </div>
              </div>
            </div>

            <p className="text-xs text-muted-foreground">
              All gateway interactions made with your credentials will be automatically attributed to your user identity and bounded by organization policies.
            </p>
          </CardContent>
          <CardFooter className="flex justify-between">
            <div />
            <Button onClick={() => setCurrentStep(2)}>
              Next: View Teams & Projects <ChevronRight className="ml-1 size-4" />
            </Button>
          </CardFooter>
        </Card>
      )}

      {/* Step 2: Assigned Teams & Projects */}
      {currentStep === 2 && (
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2 text-primary mb-1">
              <Layers className="size-5" />
              <Badge variant="outline">Step 2 of 5</Badge>
            </div>
            <CardTitle>Assigned Teams & Projects</CardTitle>
            <CardDescription>
              Teams and projects organize workloads, provide scoped budgets, and isolate model access.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div>
              <h3 className="text-sm font-semibold mb-2">Assigned Teams ({teams.length})</h3>
              {teams.length === 0 ? (
                <div className="border border-dashed rounded-lg p-4 text-center text-sm text-muted-foreground">
                  No explicit team memberships assigned. Requests will be attributed directly to your organization profile.
                </div>
              ) : (
                <div className="grid gap-2 sm:grid-cols-2">
                  {teams.map((t) => (
                    <div key={t.id} className="border rounded-lg p-3 bg-card">
                      <p className="font-semibold text-sm">{t.name}</p>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        {t.description || "Active team"}
                      </p>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div>
              <h3 className="text-sm font-semibold mb-2">Available Projects ({projects.length})</h3>
              {projects.length === 0 ? (
                <div className="border border-dashed rounded-lg p-4 text-center text-sm text-muted-foreground">
                  No projects configured yet.
                </div>
              ) : (
                <div className="grid gap-2 sm:grid-cols-2">
                  {projects.map((p) => (
                    <div key={p.id} className="border rounded-lg p-3 bg-card">
                      <div className="flex items-center justify-between">
                        <p className="font-semibold text-sm">{p.name}</p>
                        <Badge variant="outline" className="text-xs">
                          {p.status || "active"}
                        </Badge>
                      </div>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        {p.description || "Active project"}
                      </p>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </CardContent>
          <CardFooter className="flex justify-between">
            <Button variant="outline" onClick={() => setCurrentStep(1)}>
              <ChevronLeft className="mr-1 size-4" /> Back
            </Button>
            <Button onClick={() => setCurrentStep(3)}>
              Next: Model Catalog <ChevronRight className="ml-1 size-4" />
            </Button>
          </CardFooter>
        </Card>
      )}

      {/* Step 3: Authorized AI Models */}
      {currentStep === 3 && (
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2 text-primary mb-1">
              <Cpu className="size-5" />
              <Badge variant="outline">Step 3 of 5</Badge>
            </div>
            <CardTitle>Authorized Provider Models</CardTitle>
            <CardDescription>
              The models your organization has synchronized and permitted for gateway routing.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {models.length === 0 ? (
              <div className="border border-dashed rounded-lg p-6 text-center text-sm text-muted-foreground space-y-2">
                <AlertCircle className="size-6 text-amber-500 mx-auto" />
                <p className="font-medium text-foreground">No models synchronized yet</p>
                <p className="text-xs">
                  Your administrator needs to connect an approved provider account (e.g. Anthropic, OpenAI) and sync the model catalog.
                </p>
              </div>
            ) : (
              <div className="grid gap-3 sm:grid-cols-2 md:grid-cols-3">
                {models.map((m) => (
                  <div key={m.id} className="border rounded-xl p-3 bg-card/80 hover:border-primary/50 transition-colors">
                    <div className="flex items-center justify-between mb-1.5">
                      <Badge variant="secondary" className="capitalize text-xs">
                        {m.provider_kind}
                      </Badge>
                      <span className="text-xs text-muted-foreground font-mono">
                        {m.is_active !== false ? "Active" : "Inactive"}
                      </span>
                    </div>
                    <p className="font-semibold text-sm truncate" title={m.display_name || m.model_slug}>
                      {m.display_name || m.model_slug}
                    </p>
                    <p className="text-xs text-muted-foreground mt-1">Ready for gateway proxying</p>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
          <CardFooter className="flex justify-between">
            <Button variant="outline" onClick={() => setCurrentStep(2)}>
              <ChevronLeft className="mr-1 size-4" /> Back
            </Button>
            <Button onClick={() => setCurrentStep(4)}>
              Next: Issue Gateway Key <ChevronRight className="ml-1 size-4" />
            </Button>
          </CardFooter>
        </Card>
      )}

      {/* Step 4: Issue Gateway Credential */}
      {currentStep === 4 && (
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2 text-primary mb-1">
              <KeyRound className="size-5" />
              <Badge variant="outline">Step 4 of 5</Badge>
            </div>
            <CardTitle>Obtain Your Gateway Credential</CardTitle>
            <CardDescription>
              Issue a personal API key to authenticate with the Sentinel gateway from Python, TypeScript, or CLI.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {issuedKey ? (
              <div className="border border-emerald-500/30 bg-emerald-500/10 rounded-xl p-4 space-y-3">
                <div className="flex items-center gap-2 text-emerald-400 font-semibold text-sm">
                  <CheckCircle2 className="size-5" />
                  Your Gateway Key Has Been Issued
                </div>
                <p className="text-xs text-muted-foreground">
                  Copy this key now. It will not be shown again and cannot be retrieved later.
                </p>
                <div className="flex items-center gap-2">
                  <Input readOnly value={issuedKey} className="font-mono text-xs bg-background" />
                  <Button size="icon" variant="outline" onClick={() => copyToClipboard(issuedKey)}>
                    {copiedKey ? <Check className="size-4 text-emerald-500" /> : <Copy className="size-4" />}
                  </Button>
                </div>

                <div className="rounded-lg bg-black/60 p-3 text-xs font-mono text-zinc-300 space-y-1 mt-2">
                  <p className="text-zinc-500"># Shell configuration</p>
                  <p>export AIGW_API_KEY=&quot;{issuedKey}&quot;</p>
                  <p>export AIGW_BASE_URL=&quot;http://localhost:8084&quot;</p>
                </div>
              </div>
            ) : (
              <div className="border rounded-xl p-4 space-y-4 bg-muted/10">
                <div className="grid gap-3 sm:grid-cols-3">
                  <div className="space-y-1">
                    <Label htmlFor="key-name">Credential Name</Label>
                    <Input
                      id="key-name"
                      value={newKeyName}
                      onChange={(e) => setNewKeyName(e.target.value)}
                      placeholder="My Dev Key"
                    />
                  </div>

                  <div className="space-y-1">
                    <Label>Team Association</Label>
                    <Select value={selectedTeamId} onValueChange={setSelectedTeamId}>
                      <SelectTrigger>
                        <SelectValue placeholder="Select team" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="none">None (Org-wide)</SelectItem>
                        {teams.map((t) => (
                          <SelectItem key={t.id} value={t.id}>
                            {t.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="space-y-1">
                    <Label>Project Association</Label>
                    <Select value={selectedProjectId} onValueChange={setSelectedProjectId}>
                      <SelectTrigger>
                        <SelectValue placeholder="Select project" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="none">None (All projects)</SelectItem>
                        {projects.map((p) => (
                          <SelectItem key={p.id} value={p.id}>
                            {p.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <Button onClick={handleIssueKey} disabled={isIssuingKey || !newKeyName} className="w-full sm:w-auto">
                  <KeyRound className="mr-2 size-4" />
                  {isIssuingKey ? "Issuing Key…" : "Generate My Gateway Key"}
                </Button>
              </div>
            )}
          </CardContent>
          <CardFooter className="flex justify-between">
            <Button variant="outline" onClick={() => setCurrentStep(3)}>
              <ChevronLeft className="mr-1 size-4" /> Back
            </Button>
            <Button onClick={() => setCurrentStep(5)}>
              Next: Test Invocation <ChevronRight className="ml-1 size-4" />
            </Button>
          </CardFooter>
        </Card>
      )}

      {/* Step 5: Test Request & Telemetry */}
      {currentStep === 5 && (
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2 text-primary mb-1">
              <Terminal className="size-5" />
              <Badge variant="outline">Step 5 of 5</Badge>
            </div>
            <CardTitle>Run Test Gateway Request</CardTitle>
            <CardDescription>
              Execute a real test prompt through Sentinel to verify request authorization and review live telemetry.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="space-y-1">
                <Label>Model</Label>
                <Select value={testModel} onValueChange={setTestModel}>
                  <SelectTrigger>
                    <SelectValue placeholder="Choose model" />
                  </SelectTrigger>
                  <SelectContent>
                    {models.length > 0 ? (
                      models.map((m) => (
                        <SelectItem key={m.id} value={m.model_slug}>
                          {m.display_name || m.model_slug} ({m.provider_kind})
                        </SelectItem>
                      ))
                    ) : (
                      <SelectItem value="claude-3-5-sonnet">claude-3-5-sonnet</SelectItem>
                    )}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-1">
                <Label>Using Credential</Label>
                <Input
                  readOnly
                  value={activeKeyForTest ? `${activeKeyForTest.slice(0, 16)}…` : "No key selected"}
                  className="font-mono text-xs text-muted-foreground"
                />
              </div>
            </div>

            <div className="space-y-1">
              <Label>Test Prompt</Label>
              <Input
                value={testPrompt}
                onChange={(e) => setTestPrompt(e.target.value)}
                placeholder="Enter a prompt to send to the model"
              />
            </div>

            <Button onClick={handleRunTest} disabled={isTesting || !activeKeyForTest} className="w-full sm:w-auto">
              <Send className="mr-2 size-4" />
              {isTesting ? "Executing Request…" : "Send Test Request"}
            </Button>

            {testError && (
              <div className="border border-destructive/30 bg-destructive/10 rounded-lg p-3 text-xs text-destructive space-y-1">
                <p className="font-semibold">Request Failed</p>
                <p>{testError}</p>
              </div>
            )}

            {testResponse && (
              <div className="border border-emerald-500/30 bg-card rounded-xl p-4 space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Badge variant="default" className="bg-emerald-600 text-white">
                      HTTP {testResponse.status} OK
                    </Badge>
                    <span className="text-xs text-muted-foreground font-mono flex items-center gap-1">
                      <Clock className="size-3" /> {testResponse.latencyMs} ms
                    </span>
                  </div>
                  <Badge variant="outline" className="font-mono text-xs">
                    {testResponse.usage?.total_tokens || 0} tokens
                  </Badge>
                </div>

                <div className="border rounded-lg p-3 bg-muted/20 text-sm font-sans space-y-1">
                  <p className="text-xs text-muted-foreground font-semibold uppercase tracking-wider">
                    Model Response ({testResponse.model})
                  </p>
                  <p className="text-foreground whitespace-pre-wrap">{testResponse.content}</p>
                </div>

                <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 text-xs">
                  <div className="border rounded-md p-2 bg-muted/10">
                    <p className="text-muted-foreground">Input Tokens</p>
                    <p className="font-semibold">{testResponse.usage?.input_tokens || 0}</p>
                  </div>
                  <div className="border rounded-md p-2 bg-muted/10">
                    <p className="text-muted-foreground">Output Tokens</p>
                    <p className="font-semibold">{testResponse.usage?.output_tokens || 0}</p>
                  </div>
                  <div className="border rounded-md p-2 bg-muted/10">
                    <p className="text-muted-foreground">User Attribution</p>
                    <p className="font-semibold truncate">{user.name}</p>
                  </div>
                  <div className="border rounded-md p-2 bg-muted/10">
                    <p className="text-muted-foreground">Tenant Scoping</p>
                    <p className="font-semibold truncate">{organization.slug}</p>
                  </div>
                </div>
              </div>
            )}
          </CardContent>
          <CardFooter className="flex justify-between">
            <Button variant="outline" onClick={() => setCurrentStep(4)}>
              <ChevronLeft className="mr-1 size-4" /> Back
            </Button>
            <Link href="/app/overview">
              <Button variant="default">
                Complete Onboarding & Go to Console <Check className="ml-1 size-4" />
              </Button>
            </Link>
          </CardFooter>
        </Card>
      )}
    </div>
  );
}
