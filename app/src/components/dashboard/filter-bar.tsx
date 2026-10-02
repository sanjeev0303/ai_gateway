"use client";

import * as React from "react";
import { Search, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { totalsFor } from "@/lib/demo/derive";
import { models, providers } from "@/lib/demo/entities";
import { useDemoStore } from "@/lib/demo/store";
import { EMPTY_FILTERS } from "@/lib/demo/types";
import type { ProjectEnvironment } from "@/lib/demo/types";

const ALL = "all";

const ENVIRONMENT_OPTIONS: readonly ProjectEnvironment[] = [
  "production",
  "staging",
  "development",
];

const isEnvironment = (value: string): value is ProjectEnvironment =>
  ENVIRONMENT_OPTIONS.includes(value as ProjectEnvironment);

/**
 * Shared control row for analytics and list pages. Reads and writes the global
 * demo store, so a model picked here still narrows Overview, Cost, and Traffic.
 */
export function FilterBar() {
  const { range, filters, setSearch, setFilterValues, reset, isFiltered, totals } =
    useDemoStore();

  const unfiltered = React.useMemo(
    () => totalsFor(range, EMPTY_FILTERS).requests,
    [range],
  );

  return (
    <div className="flex flex-wrap items-center gap-2">
      <div className="relative min-w-48 flex-1 sm:max-w-72">
        <Search
          className="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
          aria-hidden
        />
        <Input
          type="search"
          value={filters.search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder="Filter by name"
          aria-label="Filter"
          className="pl-8"
        />
      </div>

      <Select
        value={filters.modelIds[0] ?? ALL}
        onValueChange={(value) =>
          setFilterValues("modelIds", value === ALL ? [] : [value])
        }
      >
        <SelectTrigger className="w-44" aria-label="Model">
          <SelectValue placeholder="All models" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL}>All models</SelectItem>
          {models.map((model) => (
            <SelectItem key={model.id} value={model.id}>
              {model.displayName}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <Select
        value={filters.providerIds[0] ?? ALL}
        onValueChange={(value) =>
          setFilterValues("providerIds", value === ALL ? [] : [value])
        }
      >
        <SelectTrigger className="w-40" aria-label="Provider">
          <SelectValue placeholder="All providers" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL}>All providers</SelectItem>
          {providers.map((provider) => (
            <SelectItem key={provider.id} value={provider.id}>
              {provider.displayName}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <Select
        value={filters.environments[0] ?? ALL}
        onValueChange={(value) => {
          if (value === ALL) setFilterValues("environments", []);
          else if (isEnvironment(value)) setFilterValues("environments", [value]);
        }}
      >
        <SelectTrigger className="w-36" aria-label="Environment">
          <SelectValue placeholder="All environments" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL}>All environments</SelectItem>
          {ENVIRONMENT_OPTIONS.map((option) => (
            <SelectItem key={option} value={option}>
              {option[0]!.toUpperCase() + option.slice(1)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      {isFiltered && (
        <Button variant="ghost" onClick={reset} className="text-muted-foreground">
          <X className="size-4" aria-hidden /> Reset
        </Button>
      )}

      <span className="ml-auto text-sm tabular-nums text-muted-foreground">
        {totals.requests.toLocaleString("en-US")} of{" "}
        {unfiltered.toLocaleString("en-US")}
      </span>
    </div>
  );
}
