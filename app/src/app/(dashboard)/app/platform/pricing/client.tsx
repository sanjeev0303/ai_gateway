"use client";

import * as React from "react";
import { useState, useTransition, useMemo } from "react";
import {
  Coins,
  Plus,
  History,
  Calculator,
  Search,
  CheckCircle2,
  AlertCircle,
  Tag,
  ArrowRight,
  Sparkles,
  Zap,
  TrendingDown,
  Layers,
  Info,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { formatDate } from "@/lib/format";
import type { StatusTone } from "@/types";
import {
  createPricingModelAction,
  createPricingVersionAction,
  calculateCostAction,
} from "./actions";

export interface PricingVersionItem {
  id: string;
  pricing_model_id: string;
  version: number;
  input_unit_price_micros: number;
  output_unit_price_micros: number;
  cache_read_input_token_price_micros: number;
  effective_from: string;
  effective_to?: string | null;
  created_at: string;
}

export interface PricingModelRow {
  id: string;
  model_slug: string;
  provider_kind: string;
  display_name: string;
  currency: string;
  status: string;
  created_at: string;
  updated_at: string;
  latest_version?: PricingVersionItem;
}

const display = (kind: string) =>
  kind
    .split("_")
    .map((part) => (part ? part[0].toUpperCase() + part.slice(1) : part))
    .join(" ");

const getProviderBadge = (kind: string) => {
  const k = (kind || "").toLowerCase();
  switch (k) {
    case "openai":
      return {
        label: "OpenAI",
        className: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20",
        ruleTitle: "OpenAI Token Consumption Model",
        ruleDescription: "Standard 2-tier: non-cached input + 50% discount on cache read tokens + completion tokens.",
        cacheDiscount: "50% discount",
      };
    case "anthropic":
      return {
        label: "Anthropic",
        className: "bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border-indigo-500/20",
        ruleTitle: "Anthropic Prompt Caching Model",
        ruleDescription: "Prompt Caching: 90% discount on cache read (0.10x rate) • 5x output token rate multiplier.",
        cacheDiscount: "90% discount",
      };
    case "gemini":
      return {
        label: "Google Gemini",
        className: "bg-sky-500/10 text-sky-600 dark:text-sky-400 border-sky-500/20",
        ruleTitle: "Google Gemini Context-Tiered Model",
        ruleDescription: "Context-tiered: standard rate for ≤128k prompt tokens; 2x deep context rate for >128k prompt tokens; 75% cache read discount.",
        cacheDiscount: "75% discount",
      };
    case "huggingface":
      return {
        label: "Hugging Face",
        className: "bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20",
        ruleTitle: "Hugging Face Serverless Pass-Through",
        ruleDescription: "Pass-through serverless token pricing (Novita, Together, SambaNova, Fireworks) with zero proxy markup.",
        cacheDiscount: "Not applicable",
      };
    case "mistral":
    case "custom":
      return {
        label: "Mistral AI",
        className: "bg-orange-500/10 text-orange-600 dark:text-orange-400 border-orange-500/20",
        ruleTitle: "Mistral AI Token Pricing",
        ruleDescription: "Pay-as-you-go inference pricing across Mistral Small, Medium, Large, and Codestral.",
        cacheDiscount: "Standard",
      };
    default:
      return {
        label: display(kind),
        className: "bg-muted text-muted-foreground border-border",
        ruleTitle: `${display(kind)} Pricing Model`,
        ruleDescription: "Standard token rate pricing model.",
        cacheDiscount: "Standard",
      };
  }
};

const formatMicrosRate = (micros: number): string => {
  if (micros <= 0) return "$0.00 / 1M";
  if (micros < 0.1) return `$${micros.toFixed(4)} / 1M`;
  return `$${micros.toFixed(2)} / 1M`;
};

interface PricingClientProps {
  initialModels: PricingModelRow[];
}

export function PricingClient({ initialModels }: PricingClientProps) {
  const [models, setModels] = useState<PricingModelRow[]>(initialModels);
  const [search, setSearch] = useState("");
  const [providerFilter, setProviderFilter] = useState("all");

  const [createModelOpen, setCreateModelOpen] = useState(false);
  const [createVersionOpen, setCreateVersionOpen] = useState(false);
  const [calcOpen, setCalcOpen] = useState(false);
  const [selectedModel, setSelectedModel] = useState<PricingModelRow | null>(null);

  // Filtered rows
  const filteredRows = useMemo(() => {
    return models.filter((row) => {
      const matchesSearch =
        row.display_name.toLowerCase().includes(search.toLowerCase()) ||
        row.model_slug.toLowerCase().includes(search.toLowerCase());
      const matchesProvider =
        providerFilter === "all" ||
        row.provider_kind.toLowerCase() === providerFilter.toLowerCase();
      return matchesSearch && matchesProvider;
    });
  }, [models, search, providerFilter]);

  const handleOpenVersionDialog = (row: PricingModelRow) => {
    setSelectedModel(row);
    setCreateVersionOpen(true);
  };

  const handleOpenCalcDialog = (row?: PricingModelRow) => {
    setSelectedModel(row || (models.length > 0 ? models[0] : null));
    setCalcOpen(true);
  };

  return (
    <div className="space-y-4">
      {/* Search and Action Toolbar */}
      <div className="flex flex-col sm:flex-row items-center justify-between gap-3">
        <div className="flex flex-1 items-center gap-2 w-full sm:w-auto">
          <div className="relative flex-1 sm:max-w-xs">
            <Search className="absolute left-2.5 top-2.5 size-4 text-muted-foreground" />
            <Input
              placeholder="Search model or slug..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-8 text-sm h-9"
            />
          </div>
          <Select value={providerFilter} onValueChange={setProviderFilter}>
            <SelectTrigger className="w-35 h-9 text-xs">
              <SelectValue placeholder="All Providers" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Providers</SelectItem>
              <SelectItem value="openai">OpenAI</SelectItem>
              <SelectItem value="anthropic">Anthropic</SelectItem>
              <SelectItem value="gemini">Google Gemini</SelectItem>
              <SelectItem value="mistral">Mistral AI</SelectItem>
              <SelectItem value="huggingface">Hugging Face</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="flex items-center gap-2 w-full sm:w-auto justify-end">
          <Button
            variant="outline"
            size="sm"
            className="h-9 gap-1.5"
            onClick={() => handleOpenCalcDialog()}
          >
            <Calculator className="size-4" /> Cost Calculator
          </Button>
          <Button
            size="sm"
            className="h-9 gap-1.5"
            onClick={() => setCreateModelOpen(true)}
          >
            <Plus className="size-4" /> Add Pricing Model
          </Button>
        </div>
      </div>

      {/* Main Pricing Table */}
      <div className="rounded-md border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Model</TableHead>
              <TableHead>Active Version</TableHead>
              <TableHead>Input Tokens</TableHead>
              <TableHead>Output Tokens</TableHead>
              <TableHead>Cached Input</TableHead>
              <TableHead>Effective Date</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filteredRows.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} className="h-24 text-center text-muted-foreground">
                  No pricing models match your criteria. Add a pricing model to get started.
                </TableCell>
              </TableRow>
            ) : (
              filteredRows.map((row) => {
                const pMeta = getProviderBadge(row.provider_kind);
                return (
                  <TableRow key={row.id}>
                    <TableCell>
                      <div className="space-y-1">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="font-semibold text-foreground text-sm">{row.display_name}</span>
                          <Badge variant="outline" className={`text-[11px] font-semibold ${pMeta.className}`}>
                            {pMeta.label}
                          </Badge>
                          {row.provider_kind.toLowerCase() === "gemini" && (
                            <span className="text-[10px] bg-sky-500/10 text-sky-600 dark:text-sky-400 border border-sky-500/20 px-1 rounded font-mono">
                              2x &gt;128k
                            </span>
                          )}
                          {row.provider_kind.toLowerCase() === "anthropic" && (
                            <span className="text-[10px] bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border border-indigo-500/20 px-1 rounded font-mono">
                              90% cache
                            </span>
                          )}
                          {row.provider_kind.toLowerCase() === "openai" && (
                            <span className="text-[10px] bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 px-1 rounded font-mono">
                              50% cache
                            </span>
                          )}
                        </div>
                        <p className="font-mono text-xs text-muted-foreground">{row.model_slug}</p>
                      </div>
                    </TableCell>
                  <TableCell>
                    {row.latest_version ? (
                      <Badge variant="secondary" className="font-mono text-xs font-semibold">
                        v{row.latest_version.version}
                      </Badge>
                    ) : (
                      <span className="text-xs text-muted-foreground italic">No price sheet</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <div className="space-y-0.5">
                      <div className="font-medium text-xs text-foreground">
                        {row.latest_version
                          ? formatMicrosRate(row.latest_version.input_unit_price_micros)
                          : "—"}
                      </div>
                      {row.latest_version && (
                        <p className="font-mono text-xs text-muted-foreground">
                          {row.latest_version.input_unit_price_micros} µ$/token
                        </p>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="space-y-0.5">
                      <div className="font-medium text-xs text-foreground">
                        {row.latest_version
                          ? formatMicrosRate(row.latest_version.output_unit_price_micros)
                          : "—"}
                      </div>
                      {row.latest_version && (
                        <p className="font-mono text-xs text-muted-foreground">
                          {row.latest_version.output_unit_price_micros} µ$/token
                        </p>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    {!row.latest_version ? (
                      <span className="text-xs text-muted-foreground">—</span>
                    ) : row.latest_version.cache_read_input_token_price_micros > 0 ? (
                      <div className="space-y-0.5">
                        <span className="font-medium text-xs text-foreground">
                          {formatMicrosRate(row.latest_version.cache_read_input_token_price_micros)}
                        </span>
                        <p className="font-mono text-xs text-muted-foreground">
                          {row.latest_version.cache_read_input_token_price_micros} µ$/token
                        </p>
                      </div>
                    ) : (
                      <span className="text-xs text-muted-foreground italic">Not supported</span>
                    )}
                  </TableCell>
                  <TableCell>
                    {row.latest_version ? (
                      <span className="text-xs text-muted-foreground">
                        {formatDate(row.latest_version.effective_from)}
                      </span>
                    ) : (
                      <span className="text-xs text-muted-foreground">—</span>
                    )}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-1.5">
                      <Button
                        size="sm"
                        variant="ghost"
                        className="h-7 text-xs"
                        onClick={() => handleOpenCalcDialog(row)}
                      >
                        <Calculator className="size-3.5 mr-1" /> Calc
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        className="h-7 text-xs"
                        onClick={() => handleOpenVersionDialog(row)}
                      >
                        <History className="size-3.5 mr-1" /> Update
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
                );
              })
            )}
          </TableBody>
        </Table>
      </div>

      {/* Dialog: Create Pricing Model */}
      <CreatePricingModelDialog
        open={createModelOpen}
        onOpenChange={setCreateModelOpen}
        onSuccess={(newModel) => {
          setModels((prev) => [newModel, ...prev]);
        }}
      />

      {/* Dialog: Create Version */}
      {selectedModel && (
        <CreatePricingVersionDialog
          open={createVersionOpen}
          onOpenChange={setCreateVersionOpen}
          model={selectedModel}
          onSuccess={(newVersion) => {
            setModels((prev) =>
              prev.map((m) =>
                m.id === selectedModel.id ? { ...m, latest_version: newVersion } : m
              )
            );
          }}
        />
      )}

      {/* Dialog: Cost Calculator */}
      <CalculateCostDialog
        open={calcOpen}
        onOpenChange={setCalcOpen}
        models={models}
        initialModelSlug={selectedModel?.model_slug}
      />
    </div>
  );
}

// -------------------------------------------------------------
// Dialog: Create Pricing Model
// -------------------------------------------------------------
function CreatePricingModelDialog({
  open,
  onOpenChange,
  onSuccess,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: (model: PricingModelRow) => void;
}) {
  const [providerKind, setProviderKind] = useState("openai");
  const [modelSlug, setModelSlug] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [currency, setCurrency] = useState("USD");
  const [isPending, startTransition] = useTransition();

  const handleCreate = () => {
    if (!modelSlug.trim()) {
      toast.error("Model slug is required.");
      return;
    }
    startTransition(async () => {
      const res = await createPricingModelAction({
        model_slug: modelSlug.trim(),
        provider_kind: providerKind,
        display_name: displayName.trim() || modelSlug.trim(),
        currency: currency.toUpperCase(),
      });
      if (res.error) {
        toast.error(res.error);
      } else {
        toast.success(`Pricing model ${modelSlug} created successfully!`);
        onSuccess(res.data as PricingModelRow);
        onOpenChange(false);
        setModelSlug("");
        setDisplayName("");
      }
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Coins className="size-5 text-primary" /> Add New Pricing Model
          </DialogTitle>
          <DialogDescription>
            Register a model identifier to establish versioned token pricing and cost accounting rules.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="space-y-1.5">
            <Label htmlFor="create-provider">Provider</Label>
            <Select value={providerKind} onValueChange={setProviderKind}>
              <SelectTrigger id="create-provider">
                <SelectValue placeholder="Select provider" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="openai">OpenAI</SelectItem>
                <SelectItem value="anthropic">Anthropic</SelectItem>
                <SelectItem value="gemini">Google Gemini</SelectItem>
                <SelectItem value="mistral">Mistral AI</SelectItem>
                <SelectItem value="huggingface">Hugging Face</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="create-slug">Model Slug</Label>
            <Input
              id="create-slug"
              placeholder="e.g. gpt-4o, claude-3-7-sonnet-20250219"
              value={modelSlug}
              onChange={(e) => setModelSlug(e.target.value)}
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="create-name">Display Name</Label>
            <Input
              id="create-name"
              placeholder="e.g. GPT-4o Omni"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="create-currency">Currency</Label>
            <Input
              id="create-currency"
              value={currency}
              maxLength={3}
              onChange={(e) => setCurrency(e.target.value.toUpperCase())}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleCreate} disabled={isPending}>
            {isPending ? "Creating..." : "Create Model"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// -------------------------------------------------------------
// Dialog: Create Pricing Version
// -------------------------------------------------------------
function CreatePricingVersionDialog({
  open,
  onOpenChange,
  model,
  onSuccess,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  model: PricingModelRow;
  onSuccess: (version: PricingVersionItem) => void;
}) {
  const [inputMicros, setInputMicros] = useState("2.50");
  const [outputMicros, setOutputMicros] = useState("10.00");
  const [cachedMicros, setCachedMicros] = useState("0");
  const [isPending, startTransition] = useTransition();

  const supportsCached =
    model.provider_kind === "anthropic" ||
    model.provider_kind === "gemini" ||
    (model.provider_kind === "openai" && model.model_slug.includes("gpt-4o"));

  const handleCreate = () => {
    const inputVal = Math.round(parseFloat(inputMicros || "0") * 1);
    const outputVal = Math.round(parseFloat(outputMicros || "0") * 1);
    const cachedVal = Math.round(parseFloat(cachedMicros || "0") * 1);

    if (inputVal < 0 || outputVal < 0 || cachedVal < 0) {
      toast.error("Prices must be non-negative integer micro-dollars.");
      return;
    }

    startTransition(async () => {
      const res = await createPricingVersionAction(model.id, {
        input_unit_price_micros: inputVal,
        output_unit_price_micros: outputVal,
        cache_read_input_token_price_micros: supportsCached ? cachedVal : 0,
      });
      if (res.error) {
        toast.error(res.error);
      } else {
        toast.success(`New pricing version v${(res.data as PricingVersionItem).version} applied!`);
        onSuccess(res.data as PricingVersionItem);
        onOpenChange(false);
      }
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <History className="size-5 text-primary" /> Update Price Sheet — {model.display_name}
          </DialogTitle>
          <DialogDescription>
            Create an immutable new pricing version. Existing historical records will remain permanently bound to their recorded version.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="space-y-1.5">
            <Label htmlFor="version-input">Input Price (micro-dollars / token)</Label>
            <div className="flex items-center gap-2">
              <Input
                id="version-input"
                type="number"
                step="0.01"
                min="0"
                value={inputMicros}
                onChange={(e) => setInputMicros(e.target.value)}
              />
              <span className="text-xs text-muted-foreground whitespace-nowrap">
                = ${(parseFloat(inputMicros || "0")).toFixed(2)} / 1M
              </span>
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="version-output">Output Price (micro-dollars / token)</Label>
            <div className="flex items-center gap-2">
              <Input
                id="version-output"
                type="number"
                step="0.01"
                min="0"
                value={outputMicros}
                onChange={(e) => setOutputMicros(e.target.value)}
              />
              <span className="text-xs text-muted-foreground whitespace-nowrap">
                = ${(parseFloat(outputMicros || "0")).toFixed(2)} / 1M
              </span>
            </div>
          </div>

          {supportsCached && (
            <div className="space-y-1.5">
              <Label htmlFor="version-cached">Cached Input Price (micro-dollars / token)</Label>
              <div className="flex items-center gap-2">
                <Input
                  id="version-cached"
                  type="number"
                  step="0.01"
                  min="0"
                  value={cachedMicros}
                  onChange={(e) => setCachedMicros(e.target.value)}
                />
                <span className="text-xs text-muted-foreground whitespace-nowrap">
                  = ${(parseFloat(cachedMicros || "0")).toFixed(2)} / 1M
                </span>
              </div>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleCreate} disabled={isPending}>
            {isPending ? "Applying..." : "Append Price Version"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// -------------------------------------------------------------
// Dialog: Cost Calculator
// -------------------------------------------------------------
function CalculateCostDialog({
  open,
  onOpenChange,
  models,
  initialModelSlug,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  models: PricingModelRow[];
  initialModelSlug?: string;
}) {
  const [selectedSlug, setSelectedSlug] = useState(initialModelSlug || (models[0]?.model_slug ?? ""));
  const [promptTokens, setPromptTokens] = useState("1500");
  const [completionTokens, setCompletionTokens] = useState("800");
  const [cachedTokens, setCachedTokens] = useState("0");
  const [calculatedCost, setCalculatedCost] = useState<{
    costMicros: number;
    inputCostMicros: number;
    outputCostMicros: number;
    cachedInputCostMicros: number;
    savingsMicros: number;
    pricingModelFormula?: string;
    currency: string;
    version: number;
    promptTokens: number;
    completionTokens: number;
    cachedTokens: number;
  } | null>(null);
  const [isPending, startTransition] = useTransition();

  React.useEffect(() => {
    if (initialModelSlug) setSelectedSlug(initialModelSlug);
    else if (models.length > 0 && !selectedSlug) setSelectedSlug(models[0].model_slug);
  }, [initialModelSlug, models]);

  const currentModel = useMemo(() => {
    return models.find((m) => m.model_slug === selectedSlug) || models[0] || null;
  }, [models, selectedSlug]);

  const providerMeta = currentModel ? getProviderBadge(currentModel.provider_kind) : null;
  const isGemini = currentModel?.provider_kind.toLowerCase() === "gemini";
  const ptNum = parseInt(promptTokens || "0", 10);
  const isDeepContext = isGemini && ptNum > 128000;
  const isHuggingFace = currentModel?.provider_kind.toLowerCase() === "huggingface";

  const handleCalculate = () => {
    if (!selectedSlug) {
      toast.error("Please select a model.");
      return;
    }
    const pt = parseInt(promptTokens || "0", 10);
    const ct = parseInt(completionTokens || "0", 10);
    const cdt = isHuggingFace ? 0 : parseInt(cachedTokens || "0", 10);

    startTransition(async () => {
      const res = await calculateCostAction({
        model_slug: selectedSlug,
        prompt_tokens: pt,
        completion_tokens: ct,
        cached_input_tokens: cdt,
      });
      if (res.error) {
        toast.error(res.error);
        setCalculatedCost(null);
      } else {
        const d = res.data as {
          cost_micros: number;
          input_cost_micros?: number;
          output_cost_micros?: number;
          cached_input_cost_micros?: number;
          savings_micros?: number;
          pricing_model_formula?: string;
          currency: string;
          pricing_version: { version: number };
          prompt_tokens: number;
          completion_tokens: number;
          cached_input_tokens: number;
        };
        setCalculatedCost({
          costMicros: d.cost_micros,
          inputCostMicros: d.input_cost_micros ?? 0,
          outputCostMicros: d.output_cost_micros ?? 0,
          cachedInputCostMicros: d.cached_input_cost_micros ?? 0,
          savingsMicros: d.savings_micros ?? 0,
          pricingModelFormula: d.pricing_model_formula,
          currency: d.currency,
          version: d.pricing_version.version,
          promptTokens: d.prompt_tokens,
          completionTokens: d.completion_tokens,
          cachedTokens: d.cached_input_tokens,
        });
      }
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Calculator className="size-5 text-primary" /> Token Consumption Cost Calculator
          </DialogTitle>
          <DialogDescription>
            Compute accurate token costs across providers based on official pricing rules, cache discounts, and context tiers.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Model Selector & Provider Rules Card */}
          <div className="space-y-2">
            <Label htmlFor="calc-model">Model & Provider</Label>
            <Select value={selectedSlug} onValueChange={(val) => {
              setSelectedSlug(val);
              setCalculatedCost(null);
            }}>
              <SelectTrigger id="calc-model" className="h-10">
                <SelectValue placeholder="Select a model" />
              </SelectTrigger>
              <SelectContent>
                {models.map((m) => (
                  <SelectItem key={m.id} value={m.model_slug}>
                    {m.display_name} — {display(m.provider_kind)} ({m.model_slug})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            {providerMeta && currentModel && (
              <div className="rounded-lg border bg-muted/40 p-3 space-y-1.5 text-xs">
                <div className="flex items-center justify-between">
                  <span className="font-semibold text-foreground flex items-center gap-1.5">
                    <Sparkles className="size-3.5 text-primary" />
                    {providerMeta.ruleTitle}
                  </span>
                  <Badge variant="outline" className={`font-semibold text-[11px] ${providerMeta.className}`}>
                    {providerMeta.label}
                  </Badge>
                </div>
                <p className="text-muted-foreground leading-relaxed">
                  {providerMeta.ruleDescription}
                </p>
                {currentModel.latest_version && (
                  <div className="pt-1.5 flex flex-wrap items-center gap-x-4 gap-y-1 text-muted-foreground font-mono">
                    <span>Input: {formatMicrosRate(currentModel.latest_version.input_unit_price_micros)}</span>
                    <span>Output: {formatMicrosRate(currentModel.latest_version.output_unit_price_micros)}</span>
                    {currentModel.latest_version.cache_read_input_token_price_micros > 0 && (
                      <span className="text-emerald-600 dark:text-emerald-400">
                        Cache Read: {formatMicrosRate(currentModel.latest_version.cache_read_input_token_price_micros)} ({providerMeta.cacheDiscount})
                      </span>
                    )}
                  </div>
                )}
              </div>
            )}
          </div>

          {/* Prompt & Completion Token Inputs */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <div className="flex items-center justify-between">
                <Label htmlFor="calc-prompt">Prompt Tokens</Label>
                <div className="flex items-center gap-1">
                  {["1.5k", "32k", "150k"].map((t) => (
                    <button
                      key={t}
                      type="button"
                      onClick={() => setPromptTokens(t === "1.5k" ? "1500" : t === "32k" ? "32000" : "150000")}
                      className="text-[10px] px-1.5 py-0.5 rounded bg-muted hover:bg-muted/80 text-muted-foreground hover:text-foreground font-mono"
                    >
                      {t}
                    </button>
                  ))}
                </div>
              </div>
              <Input
                id="calc-prompt"
                type="number"
                min="0"
                value={promptTokens}
                onChange={(e) => setPromptTokens(e.target.value)}
              />
            </div>

            <div className="space-y-1.5">
              <div className="flex items-center justify-between">
                <Label htmlFor="calc-completion">Completion Tokens</Label>
                <div className="flex items-center gap-1">
                  {["500", "2k", "4k"].map((t) => (
                    <button
                      key={t}
                      type="button"
                      onClick={() => setCompletionTokens(t === "500" ? "500" : t === "2k" ? "2000" : "4000")}
                      className="text-[10px] px-1.5 py-0.5 rounded bg-muted hover:bg-muted/80 text-muted-foreground hover:text-foreground font-mono"
                    >
                      {t}
                    </button>
                  ))}
                </div>
              </div>
              <Input
                id="calc-completion"
                type="number"
                min="0"
                value={completionTokens}
                onChange={(e) => setCompletionTokens(e.target.value)}
              />
            </div>
          </div>

          {/* Cached Input Tokens Input */}
          {!isHuggingFace && (
            <div className="space-y-1.5">
              <div className="flex items-center justify-between">
                <Label htmlFor="calc-cached">Cached Input Tokens (Prompt Caching)</Label>
                <span className="text-xs text-muted-foreground">
                  {providerMeta?.cacheDiscount}
                </span>
              </div>
              <Input
                id="calc-cached"
                type="number"
                min="0"
                max={promptTokens}
                value={cachedTokens}
                onChange={(e) => setCachedTokens(e.target.value)}
                placeholder="0"
              />
            </div>
          )}

          {/* Google Gemini Deep Context Notification */}
          {isDeepContext && (
            <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 p-3 text-xs text-amber-800 dark:text-amber-200 flex items-start gap-2">
              <Zap className="size-4 mt-0.5 shrink-0 text-amber-500" />
              <div>
                <span className="font-semibold">Deep Context Tier Triggered (&gt;128k prompt tokens):</span>
                <p className="mt-0.5 text-amber-700 dark:text-amber-300">
                  Per Google Gemini pricing policy, prompts exceeding 128,000 tokens are billed at the deep-context rate ($2.50/1M input, $10.00/1M output for Pro).
                </p>
              </div>
            </div>
          )}

          {/* Results Card */}
          {calculatedCost && (
            <div className="rounded-xl border bg-card p-4 space-y-3.5 shadow-sm">
              <div className="flex items-center justify-between border-b pb-3">
                <div>
                  <span className="text-xs text-muted-foreground uppercase tracking-wider font-semibold">
                    Calculated Total Spend
                  </span>
                  <div className="flex items-baseline gap-2 mt-0.5">
                    <span className="text-3xl font-extrabold tracking-tight text-foreground">
                      ${(calculatedCost.costMicros / 1_000_000).toFixed(6)}
                    </span>
                    <span className="text-xs text-muted-foreground font-mono">
                      ({calculatedCost.costMicros.toLocaleString()} µ$)
                    </span>
                  </div>
                </div>
                <Badge variant="outline" className="font-mono text-xs">
                  Rate Sheet v{calculatedCost.version}
                </Badge>
              </div>

              {calculatedCost.pricingModelFormula && (
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground bg-muted/30 px-2.5 py-1.5 rounded font-mono">
                  <Info className="size-3.5 shrink-0" />
                  <span>{calculatedCost.pricingModelFormula}</span>
                </div>
              )}

              {/* Itemized Breakdown Grid */}
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-2.5 pt-1">
                <div className="rounded-lg border bg-muted/30 p-2.5 space-y-1">
                  <div className="text-[11px] text-muted-foreground uppercase tracking-wider font-semibold">
                    Prompt (Uncached)
                  </div>
                  <div className="text-sm font-bold text-foreground">
                    ${(calculatedCost.inputCostMicros / 1_000_000).toFixed(6)}
                  </div>
                  <div className="text-[10px] text-muted-foreground font-mono">
                    {Math.max(0, calculatedCost.promptTokens - calculatedCost.cachedTokens).toLocaleString()} tokens
                  </div>
                </div>

                <div className="rounded-lg border bg-muted/30 p-2.5 space-y-1">
                  <div className="text-[11px] text-muted-foreground uppercase tracking-wider font-semibold">
                    Cached Prompt Reads
                  </div>
                  <div className="text-sm font-bold text-foreground">
                    ${(calculatedCost.cachedInputCostMicros / 1_000_000).toFixed(6)}
                  </div>
                  <div className="text-[10px] text-muted-foreground font-mono">
                    {calculatedCost.cachedTokens.toLocaleString()} tokens
                  </div>
                  {calculatedCost.savingsMicros > 0 && (
                    <div className="text-[10px] text-emerald-600 dark:text-emerald-400 font-semibold flex items-center gap-0.5">
                      <TrendingDown className="size-3" />
                      Saved ${(calculatedCost.savingsMicros / 1_000_000).toFixed(6)}
                    </div>
                  )}
                </div>

                <div className="rounded-lg border bg-muted/30 p-2.5 space-y-1">
                  <div className="text-[11px] text-muted-foreground uppercase tracking-wider font-semibold">
                    Completion (Output)
                  </div>
                  <div className="text-sm font-bold text-foreground">
                    ${(calculatedCost.outputCostMicros / 1_000_000).toFixed(6)}
                  </div>
                  <div className="text-[10px] text-muted-foreground font-mono">
                    {calculatedCost.completionTokens.toLocaleString()} tokens
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>

        <DialogFooter className="gap-2 sm:gap-0">
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            Close
          </Button>
          <Button onClick={handleCalculate} disabled={isPending} className="gap-1.5">
            <Calculator className="size-4" />
            {isPending ? "Calculating..." : "Compute Token Cost"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
