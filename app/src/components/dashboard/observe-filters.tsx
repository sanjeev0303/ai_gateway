"use client";

import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useEffect, useId, useState } from "react";
import { ChevronDown, SlidersHorizontal, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";
import type { Directory } from "@/lib/directory";
import { loadDirectory } from "./directory-actions";
import { SearchSelect } from "./search-select";

const PROVIDERS = [
  { value: "gemini", label: "Google Gemini" },
  { value: "mistral", label: "Mistral AI" },
  { value: "huggingface", label: "Hugging Face" },
  { value: "openai", label: "OpenAI" },
  { value: "anthropic", label: "Anthropic" },
  { value: "azure_openai", label: "Azure OpenAI" },
  { value: "custom", label: "Custom provider" },
];

const STATUSES = [
  { value: "succeeded", label: "Succeeded" },
  { value: "failed", label: "Failed" },
  { value: "cancelled", label: "Cancelled" },
];

type FilterName = "provider" | "status" | "model" | "team_id" | "project_id" | "user_id" | "service_account_id";

const MORE: { name: FilterName; label: string; list: keyof Directory }[] = [
  { name: "model", label: "Model", list: "models" },
  { name: "team_id", label: "Team", list: "teams" },
  { name: "project_id", label: "Project", list: "projects" },
  { name: "user_id", label: "User", list: "users" },
  { name: "service_account_id", label: "Service account", list: "serviceAccounts" },
];

interface ObserveFiltersProps {
  /** Filters this page's data cannot be narrowed by. */
  exclude?: readonly FilterName[];
}

export function ObserveFilters({ exclude = [] }: ObserveFiltersProps) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const baseId = useId();

  const more = MORE.filter((f) => !exclude.includes(f.name));
  const activeMore = more.filter((f) => searchParams.get(f.name));

  const [expanded, setExpanded] = useState(activeMore.length > 0);
  const [directory, setDirectory] = useState<Directory | null>(null);

  // Names are only needed once the panel opens or a filter is already set.
  const needsNames = expanded || activeMore.length > 0;
  useEffect(() => {
    if (!needsNames || directory) return;
    let live = true;
    loadDirectory()
      .then((result) => {
        if (live) setDirectory(result);
      })
      .catch(() => {
        // Pickers show an empty list; the chips fall back to a short id.
      });
    return () => {
      live = false;
    };
  }, [needsNames, directory]);

  const navigate = (params: URLSearchParams) => {
    const query = params.toString();
    router.push(query ? `${pathname}?${query}` : pathname, { scroll: false });
  };

  const setFilter = (name: string, value: string) => {
    const params = new URLSearchParams(searchParams.toString());
    if (value) params.set(name, value);
    else params.delete(name);
    navigate(params);
  };

  const clearAll = () => {
    const params = new URLSearchParams(searchParams.toString());
    for (const name of ["provider", "status", ...MORE.map((f) => f.name)]) params.delete(name);
    navigate(params);
  };

  const provider = searchParams.get("provider") ?? "";
  const status = searchParams.get("status") ?? "";

  const chips: { name: string; label: string; value: string }[] = [];
  if (provider) chips.push({ name: "provider", label: "Provider", value: PROVIDERS.find((p) => p.value === provider)?.label ?? provider });
  if (status) chips.push({ name: "status", label: "Status", value: STATUSES.find((s) => s.value === status)?.label ?? status });
  for (const filter of activeMore) {
    const id = searchParams.get(filter.name) ?? "";
    const entry = directory?.[filter.list].find((e) => e.id === id);
    chips.push({ name: filter.name, label: filter.label, value: entry?.name ?? (id.length > 24 ? `${id.slice(0, 8)}…` : id) });
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <Select value={provider || "all"} onValueChange={(v) => setFilter("provider", v === "all" ? "" : v)}>
          <SelectTrigger className="min-w-40 flex-1 sm:flex-none" aria-label="Provider">
            <SelectValue placeholder="All providers" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All providers</SelectItem>
            {PROVIDERS.map((p) => (
              <SelectItem key={p.value} value={p.value}>{p.label}</SelectItem>
            ))}
          </SelectContent>
        </Select>

        {!exclude.includes("status") && (
          <Select value={status || "all"} onValueChange={(v) => setFilter("status", v === "all" ? "" : v)}>
            <SelectTrigger className="min-w-36 flex-1 sm:flex-none" aria-label="Status">
              <SelectValue placeholder="All statuses" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All statuses</SelectItem>
              {STATUSES.map((s) => (
                <SelectItem key={s.value} value={s.value}>{s.label}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}

        <Button
          type="button"
          variant="outline"
          aria-expanded={expanded}
          aria-controls={`${baseId}-more`}
          onClick={() => setExpanded((open) => !open)}
        >
          <SlidersHorizontal className="size-4" aria-hidden />
          More filters
          {activeMore.length > 0 && (
            <span className="rounded-full bg-primary px-1.5 text-xs text-primary-foreground tabular-nums">
              {activeMore.length}
            </span>
          )}
          <ChevronDown className={cn("size-3.5 transition-transform duration-150", expanded && "rotate-180")} aria-hidden />
        </Button>
      </div>

      {expanded && (
        <div
          id={`${baseId}-more`}
          className="grid grid-cols-1 gap-3 rounded-2xl border border-border bg-card p-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5"
        >
          {more.map((filter) => (
            <div key={filter.name} className="flex min-w-0 flex-col gap-1.5">
              <Label htmlFor={`${baseId}-${filter.name}`} className="text-xs font-medium text-muted-foreground">
                {filter.label}
              </Label>
              <SearchSelect
                id={`${baseId}-${filter.name}`}
                value={searchParams.get(filter.name) ?? ""}
                onChange={(value) => setFilter(filter.name, value)}
                options={directory?.[filter.list] ?? []}
                loading={!directory}
                placeholder={`Any ${filter.label.toLowerCase()}`}
                clearLabel={`Any ${filter.label.toLowerCase()}`}
              />
            </div>
          ))}
        </div>
      )}

      {chips.length > 0 && (
        <ul aria-label="Active filters" className="flex flex-wrap items-center gap-1.5">
          {chips.map((chip) => (
            <li key={chip.name}>
              <button
                type="button"
                onClick={() => setFilter(chip.name, "")}
                aria-label={`Remove ${chip.label} filter: ${chip.value}`}
                className="group inline-flex h-7 items-center gap-1.5 rounded-full border border-border bg-accent/70 pr-2 pl-3 text-xs transition-colors duration-150 hover:border-border-strong focus-visible:ring-3 focus-visible:ring-ring/30 focus-visible:outline-none"
              >
                <span className="text-muted-foreground">{chip.label}</span>
                <span className="font-medium">{chip.value}</span>
                <X className="size-3 text-muted-foreground group-hover:text-foreground" aria-hidden />
              </button>
            </li>
          ))}
          {chips.length > 1 && (
            <li>
              <button
                type="button"
                onClick={clearAll}
                className="h-7 rounded-full px-2.5 text-xs font-medium text-primary underline-offset-4 hover:underline focus-visible:ring-3 focus-visible:ring-ring/30 focus-visible:outline-none"
              >
                Clear all
              </button>
            </li>
          )}
        </ul>
      )}
    </div>
  );
}
