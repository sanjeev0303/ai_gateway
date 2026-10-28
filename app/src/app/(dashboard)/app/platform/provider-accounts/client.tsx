"use client";

import * as React from "react";
import { Plus, Eye, EyeOff, ShieldCheck, CheckCircle2, AlertTriangle, XCircle, Sparkles } from "lucide-react";
import { toast } from "sonner";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "@/components/ui/card";
import { createProviderAccount } from "./actions";

interface ProviderOption {
  kind: string;
  name: string;
  badge: string;
  models: string;
  capabilities: string[];
  placeholderKey: string;
  defaultBaseUrl?: string;
}

const PROVIDER_CATALOG: ProviderOption[] = [
  {
    kind: "anthropic",
    name: "Anthropic",
    badge: "Tier 1",
    models: "Claude 3.7 Sonnet, Claude 3.5 Haiku, Opus",
    capabilities: ["Thinking / Reasoning", "Streaming", "Tool Use", "Prompt Caching"],
    placeholderKey: "sk-ant-...",
  },
  {
    kind: "openai",
    name: "OpenAI",
    badge: "Tier 1",
    models: "GPT-4o, o1, o3-mini, GPT-4o-mini",
    capabilities: ["Reasoning Effort", "Structured Outputs", "Vision", "Streaming"],
    placeholderKey: "sk-proj-...",
  },
  {
    kind: "gemini",
    name: "Google Gemini",
    badge: "Tier 1",
    models: "Gemini 2.5 Pro, Flash, Thinking, Flash-Lite",
    capabilities: ["Multimodal", "System Instructions", "Thinking Mode", "Tool Calling"],
    placeholderKey: "AIzaSy...",
  },
  {
    kind: "huggingface",
    name: "Hugging Face",
    badge: "Tier 1",
    models: "DeepSeek-R1, Qwen 2.5, Llama 3.3, Mistral",
    capabilities: ["Serverless Inference", "DeepSeek Reasoning", "Custom Endpoints"],
    placeholderKey: "hf_...",
    defaultBaseUrl: "https://api-inference.huggingface.co",
  },
  {
    kind: "azure_openai",
    name: "Azure OpenAI",
    badge: "Enterprise",
    models: "Dedicated deployment endpoints",
    capabilities: ["Private VNet", "Azure Compliance", "SLA Guarantees"],
    placeholderKey: "az-...",
  },
  {
    kind: "custom",
    name: "Custom Provider",
    badge: "OpenAI-Compatible",
    models: "vLLM, Ollama, LocalAI, TGI",
    capabilities: ["Self-Hosted", "Zero Outbound Data", "Standard OpenAI API"],
    placeholderKey: "Bearer token or key...",
  },
];

export function CreateProviderAccountButton() {
  const router = useRouter();
  const [open, setOpen] = React.useState(false);
  const [selectedProvider, setSelectedProvider] = React.useState<ProviderOption>(PROVIDER_CATALOG[0]);
  const [name, setName] = React.useState("");
  const [baseUrl, setBaseUrl] = React.useState("");
  const [apiKey, setApiKey] = React.useState("");
  const [showSecret, setShowSecret] = React.useState(false);
  const [isSubmitting, setIsSubmitting] = React.useState(false);

  // Result state
  const [resultStatus, setResultStatus] = React.useState<string | null>(null);

  const handleSelectProvider = (prov: ProviderOption) => {
    setSelectedProvider(prov);
    if (!name || PROVIDER_CATALOG.some((p) => name === `${p.name} Production`)) {
      setName(`${prov.name} Production`);
    }
    if (prov.defaultBaseUrl) {
      setBaseUrl(prov.defaultBaseUrl);
    } else {
      setBaseUrl("");
    }
  };

  const handleOpen = () => {
    setSelectedProvider(PROVIDER_CATALOG[0]);
    setName("Anthropic Production");
    setBaseUrl("");
    setApiKey("");
    setShowSecret(false);
    setResultStatus(null);
    setOpen(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      toast.error("Please enter a display name for the account");
      return;
    }

    setIsSubmitting(true);
    const fd = new FormData();
    fd.append("provider_kind", selectedProvider.kind);
    fd.append("name", name.trim());
    if (baseUrl.trim()) fd.append("base_url", baseUrl.trim());
    if (apiKey.trim()) fd.append("api_key", apiKey.trim());

    const res = await createProviderAccount({}, fd);
    setIsSubmitting(false);

    if (res.error) {
      toast.error(res.error);
    } else {
      setResultStatus(res.validationStatus || "connected");
      if (res.validationStatus === "connected") {
        toast.success(`Connected ${selectedProvider.name} successfully! Models discovered.`);
      } else if (res.validationStatus === "credential_invalid") {
        toast.error(`Connected ${selectedProvider.name}, but upstream rejected the credential.`);
      } else if (res.validationStatus === "provider_unreachable") {
        toast.warning(`Account created, but upstream endpoint was unreachable.`);
      } else {
        toast.success(`Account created successfully.`);
      }
      router.refresh();
      if (res.accountId) {
        setTimeout(() => {
          setOpen(false);
          router.push(`/app/platform/provider-accounts/${res.accountId}`);
        }, 1200);
      }
    }
  };

  return (
    <>
      <Button onClick={handleOpen} size="sm" className="gap-1.5 shadow-sm">
        <Plus className="h-4 w-4" />
        Connect Provider
      </Button>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-xl">
              <Sparkles className="size-5 text-primary" /> Connect AI Provider
            </DialogTitle>
            <DialogDescription>
              Select a foundation model provider to connect your organization's API credentials.
            </DialogDescription>
          </DialogHeader>

          {resultStatus ? (
            <div className="py-6 space-y-4 text-center">
              {resultStatus === "connected" && (
                <div className="space-y-2">
                  <CheckCircle2 className="size-12 text-emerald-500 mx-auto" />
                  <div className="text-lg font-semibold text-emerald-600">Provider Connected & Verified</div>
                  <p className="text-xs text-muted-foreground">
                    Credentials validated with upstream API. Models have been automatically synchronized to the registry.
                  </p>
                </div>
              )}
              {resultStatus === "credential_invalid" && (
                <div className="space-y-2">
                  <XCircle className="size-12 text-destructive mx-auto" />
                  <div className="text-lg font-semibold text-destructive">Credential Invalid</div>
                  <p className="text-xs text-muted-foreground">
                    The account was created, but the upstream provider rejected authentication. Please rotate the API key.
                  </p>
                </div>
              )}
              {resultStatus === "provider_unreachable" && (
                <div className="space-y-2">
                  <AlertTriangle className="size-12 text-amber-500 mx-auto" />
                  <div className="text-lg font-semibold text-amber-600">Provider Unreachable</div>
                  <p className="text-xs text-muted-foreground">
                    Upstream endpoint timed out or failed DNS resolution. Routing will degrade gracefully.
                  </p>
                </div>
              )}
              {resultStatus === "discovery_unavailable" && (
                <div className="space-y-2">
                  <AlertTriangle className="size-12 text-amber-500 mx-auto" />
                  <div className="text-lg font-semibold text-amber-600">Model Discovery Unavailable</div>
                  <p className="text-xs text-muted-foreground">
                    Credential verified, but provider model catalog query was temporarily unavailable.
                  </p>
                </div>
              )}
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4 py-2">
              <div className="space-y-2">
                <Label className="text-xs font-semibold text-muted-foreground uppercase">
                  1. Select Upstream Provider
                </Label>
                <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                  {PROVIDER_CATALOG.map((prov) => {
                    const isSelected = selectedProvider.kind === prov.kind;
                    return (
                      <div
                        key={prov.kind}
                        onClick={() => handleSelectProvider(prov)}
                        className={`cursor-pointer rounded-lg border p-3 transition-all text-left flex flex-col justify-between ${
                          isSelected
                            ? "border-primary bg-accent/30 shadow-xs"
                            : "border-border hover:border-primary/40 bg-card"
                        }`}
                      >
                        <div className="flex items-center justify-between">
                          <span className="font-semibold text-sm">{prov.name}</span>
                          <Badge variant="outline" className="text-xs px-1.5 py-0.5">
                            {prov.badge}
                          </Badge>
                        </div>
                        <div className="text-xs text-muted-foreground mt-1 line-clamp-1">
                          {prov.models}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>

              <div className="space-y-3 pt-2 border-t">
                <Label className="text-xs font-semibold text-muted-foreground uppercase">
                  2. Connection & Credential Details
                </Label>

                <div className="grid gap-3 sm:grid-cols-2">
                  <div className="space-y-1.5">
                    <Label htmlFor="account-name">Display Name</Label>
                    <Input
                      id="account-name"
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      placeholder="e.g. Production Claude Account"
                      required
                    />
                  </div>

                  <div className="space-y-1.5">
                    <Label htmlFor="base-url">
                      Custom Base URL <span className="text-xs text-muted-foreground">(Optional)</span>
                    </Label>
                    <Input
                      id="base-url"
                      value={baseUrl}
                      onChange={(e) => setBaseUrl(e.target.value)}
                      placeholder={selectedProvider.defaultBaseUrl || "Provider default"}
                    />
                  </div>
                </div>

                <div className="space-y-1.5">
                  <div className="flex items-center justify-between">
                    <Label htmlFor="api-key">API Key / Secret Token</Label>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => setShowSecret(!showSecret)}
                      className="h-6 text-xs text-muted-foreground px-2"
                    >
                      {showSecret ? <EyeOff className="size-3 mr-1" /> : <Eye className="size-3 mr-1" />}
                      {showSecret ? "Hide" : "Show"}
                    </Button>
                  </div>
                  <Input
                    id="api-key"
                    type={showSecret ? "text" : "password"}
                    value={apiKey}
                    onChange={(e) => setApiKey(e.target.value)}
                    placeholder={selectedProvider.placeholderKey}
                    className="font-mono text-xs"
                  />
                  <div className="flex items-center gap-1.5 text-xs text-muted-foreground mt-1">
                    <ShieldCheck className="size-3.5 text-emerald-500 shrink-0" />
                    <span>
                      Encrypted with AES-256 before persistence. Never logged or echoed in plaintext.
                    </span>
                  </div>
                </div>
              </div>

              <DialogFooter className="gap-2 sm:gap-0 pt-2 border-t">
                <Button type="button" variant="ghost" onClick={() => setOpen(false)} disabled={isSubmitting}>
                  Cancel
                </Button>
                <Button type="submit" disabled={isSubmitting}>
                  {isSubmitting ? "Connecting & Validating…" : `Connect ${selectedProvider.name}`}
                </Button>
              </DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
}
