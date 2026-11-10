"use client";

import * as React from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { Search } from "lucide-react";

import { Input } from "@ai-governance/console-ui/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@ai-governance/console-ui/ui/select";

const ALL = "all";

interface ListFiltersProps {
  searchLabel: string;
  placeholder: string;
  /** Options for the status select; leave out for a search-only filter. */
  statuses?: { value: string; label: string }[];
  /** The query parameter the select writes; "status" unless said otherwise. */
  statusParam?: string;
  statusLabel?: string;
}

/**
 * Search and status filters for a platform list. Both live in the URL, so a
 * filtered list can be shared, and changing either goes back to page one.
 */
export function ListFilters({ searchLabel, placeholder, statuses = [], statusParam = "status", statusLabel: selectLabel = "Status" }: ListFiltersProps) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const [search, setSearch] = React.useState(searchParams.get("search") ?? "");
  const status = searchParams.get(statusParam) ?? ALL;
  // The animated Select mounts its items only when open, so the trigger is
  // given its label rather than reading it from the selected item.
  const statusLabel = statuses.find((option) => option.value === status)?.label ?? `Any ${selectLabel.toLowerCase()}`;

  const apply = (changes: Record<string, string>) => {
    const params = new URLSearchParams(searchParams.toString());
    for (const [key, value] of Object.entries(changes)) {
      if (value && value !== ALL) params.set(key, value);
      else params.delete(key);
    }
    params.delete("page");
    const query = params.toString();
    router.push(query ? `${pathname}?${query}` : pathname, { scroll: false });
  };

  return (
    <div className="flex flex-wrap items-center gap-2">
      <form
        role="search"
        className="relative min-w-0 flex-1 sm:max-w-sm"
        onSubmit={(event) => {
          event.preventDefault();
          apply({ search: search.trim() });
        }}
      >
        <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
        <Input
          type="search"
          name="search"
          autoComplete="off"
          spellCheck={false}
          aria-label={searchLabel}
          placeholder={placeholder}
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          className="pl-9"
        />
      </form>
      {statuses.length > 0 && (
      <Select value={status} onValueChange={(value) => apply({ [statusParam]: value })}>
        <SelectTrigger className="w-40" aria-label={selectLabel}>
          <SelectValue>{statusLabel}</SelectValue>
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            <SelectItem value={ALL}>Any {selectLabel.toLowerCase()}</SelectItem>
            {statuses.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
      )}
    </div>
  );
}
