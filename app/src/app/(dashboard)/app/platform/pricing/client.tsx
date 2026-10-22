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

const formatMicrosRate = (micros: number): string => {
  if (micros <= 0) return "$0.00 / 1M";
  const perMillion = (micros / 1_000_000) * 1_000_000;
  return `$${perMillion.toFixed(2)} / 1M`;
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
            <SelectTrigger className="w-[140px] h-9 text-xs">
              <SelectValue placeholder="All Providers" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Providers</SelectItem>
              <SelectItem value="openai">OpenAI</SelectItem>
              <SelectItem value="anthropic">Anthropic</SelectItem>
              <SelectItem value="gemini">Google Gemini</SelectItem>
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
              filteredRows.map((row) => (
                <TableRow key={row.id}>
                  <TableCell>
                    <div className="space-y-0.5">
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-foreground">{row.display_name}</span>
                        <Badge variant="outline" className="text-[10px] uppercase font-semibold">
                          {display(row.provider_kind)}
                        </Badge>
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
                        <p className="font-mono text-[10px] text-muted-foreground">
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
                        <p className="font-mono text-[10px] text-muted-foreground">
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
                        <p className="font-mono text-[10px] text-muted-foreground">
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
              ))
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
    currency: string;
    version: number;
  } | null>(null);
  const [isPending, startTransition] = useTransition();

  React.useEffect(() => {
    if (initialModelSlug) setSelectedSlug(initialModelSlug);
    else if (models.length > 0 && !selectedSlug) setSelectedSlug(models[0].model_slug);
  }, [initialModelSlug, models]);

  const handleCalculate = () => {
    if (!selectedSlug) {
      toast.error("Please select a model.");
      return;
    }
    const pt = parseInt(promptTokens || "0", 10);
    const ct = parseInt(completionTokens || "0", 10);
    const cdt = parseInt(cachedTokens || "0", 10);

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
          currency: string;
          pricing_version: { version: number };
        };
        setCalculatedCost({
          costMicros: d.cost_micros,
          currency: d.currency,
          version: d.pricing_version.version,
        });
      }
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Calculator className="size-5 text-primary" /> Token Cost Calculator
          </DialogTitle>
          <DialogDescription>
            Simulate exact token spend for any model using verified organization pricing rates.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="space-y-1.5">
            <Label htmlFor="calc-model">Model</Label>
            <Select value={selectedSlug} onValueChange={setSelectedSlug}>
              <SelectTrigger id="calc-model">
                <SelectValue placeholder="Select a model" />
              </SelectTrigger>
              <SelectContent>
                {models.map((m) => (
                  <SelectItem key={m.id} value={m.model_slug}>
                    {m.display_name} ({display(m.provider_kind)})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="calc-prompt">Prompt Tokens</Label>
              <Input
                id="calc-prompt"
                type="number"
                min="0"
                value={promptTokens}
                onChange={(e) => setPromptTokens(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="calc-completion">Completion Tokens</Label>
              <Input
                id="calc-completion"
                type="number"
                min="0"
                value={completionTokens}
                onChange={(e) => setCompletionTokens(e.target.value)}
              />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="calc-cached">Cached Input Tokens</Label>
            <Input
              id="calc-cached"
              type="number"
              min="0"
              value={cachedTokens}
              onChange={(e) => setCachedTokens(e.target.value)}
            />
          </div>

          {calculatedCost && (
            <div className="rounded-lg border bg-muted/40 p-4 space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-xs text-muted-foreground uppercase tracking-wider font-semibold">
                  Estimated Total Cost
                </span>
                <Badge variant="outline" className="font-mono text-xs">
                  Rate Sheet v{calculatedCost.version}
                </Badge>
              </div>
              <div className="flex items-baseline gap-2">
                <span className="text-2xl font-bold tracking-tight text-foreground">
                  ${(calculatedCost.costMicros / 1_000_000).toFixed(6)}
                </span>
                <span className="text-xs text-muted-foreground">
                  ({calculatedCost.costMicros.toLocaleString()} µ$)
                </span>
              </div>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            Close
          </Button>
          <Button onClick={handleCalculate} disabled={isPending}>
            {isPending ? "Calculating..." : "Compute Cost"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
