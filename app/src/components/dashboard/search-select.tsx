"use client";

import * as React from "react";
import { Check, ChevronDown, Search } from "lucide-react";

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@ai-governance/console-ui/lib/utils";

export interface SearchSelectOption {
  id: string;
  name: string;
  detail?: string;
}

interface SearchSelectProps {
  id?: string;
  value: string;
  onChange: (value: string) => void;
  options: readonly SearchSelectOption[];
  placeholder: string;
  /** Names the control for assistive tech when no visible label points at it. */
  label?: string;
  loading?: boolean;
  /** Offers a first row that clears the value. */
  clearLabel?: string;
  disabled?: boolean;
  className?: string;
}

/** A dropdown you can type into, so nobody has to know an id to pick something. */
export function SearchSelect({
  id,
  value,
  onChange,
  options,
  placeholder,
  label,
  loading = false,
  clearLabel,
  disabled,
  className,
}: SearchSelectProps) {
  const [open, setOpen] = React.useState(false);
  const [query, setQuery] = React.useState("");
  const [cursor, setCursor] = React.useState(0);
  const listId = React.useId();

  const matches = React.useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) return options;
    return options.filter(
      (o) => o.name.toLowerCase().includes(needle) || o.detail?.toLowerCase().includes(needle),
    );
  }, [options, query]);

  // The clear row is an ordinary row, so the arrow keys can reach it too.
  const rows: readonly SearchSelectOption[] =
    clearLabel && !query ? [{ id: "", name: clearLabel }, ...matches] : matches;

  const selected = options.find((o) => o.id === value);

  const choose = (next: string) => {
    onChange(next);
    setOpen(false);
    setQuery("");
  };

  const moveCursor = (next: number) => {
    setCursor(next);
    document.getElementById(`${listId}-${next}`)?.scrollIntoView({ block: "nearest" });
  };

  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      moveCursor(Math.min(cursor + 1, rows.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      moveCursor(Math.max(cursor - 1, 0));
    } else if (e.key === "Enter") {
      e.preventDefault();
      const row = rows[cursor];
      if (row) choose(row.id);
    }
  };

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        setCursor(0);
        if (!next) setQuery("");
      }}
    >
      <PopoverTrigger asChild>
        <button
          id={id}
          type="button"
          disabled={disabled}
          aria-label={label}
          className={cn(
            "flex h-9 w-full min-w-0 items-center justify-between gap-2 rounded-[10px] border border-input bg-card pr-2.5 pl-3 text-left text-sm transition-[border-color,box-shadow] duration-150 outline-none hover:border-border-strong focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/25 disabled:cursor-not-allowed disabled:opacity-55 dark:bg-input/20",
            className,
          )}
        >
          <span className={cn("truncate", !selected && !value && "text-muted-foreground")}>
            {selected?.name ?? (value ? `${value.slice(0, 8)}…` : placeholder)}
          </span>
          <ChevronDown className="size-4 shrink-0 text-muted-foreground" aria-hidden />
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-(--radix-popover-trigger-width) min-w-60 gap-1.5 p-1.5" data-lenis-prevent>
        <div className="relative">
          <Search className="absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground" aria-hidden />
          <input
            autoFocus
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setCursor(0);
            }}
            onKeyDown={onKeyDown}
            placeholder="Type to filter…"
            role="combobox"
            aria-label={`Filter ${label ?? placeholder}`}
            aria-expanded={open}
            aria-controls={listId}
            aria-autocomplete="list"
            aria-activedescendant={rows[cursor] ? `${listId}-${cursor}` : undefined}
            className="h-8 w-full rounded-lg bg-muted/60 pr-2 pl-8 text-sm outline-none placeholder:text-muted-foreground focus-visible:ring-2 focus-visible:ring-ring/30"
          />
        </div>
        <ul
          id={listId}
          role="listbox"
          aria-label={label ?? placeholder}
          className="flex max-h-60 flex-col overflow-y-auto overscroll-contain empty:hidden"
        >
          {rows.map((row, index) => (
            // Focus stays in the input above, which handles the keyboard for this list.
            <li
              key={row.id || "clear"}
              id={`${listId}-${index}`}
              role="option"
              aria-selected={row.id === value}
              onClick={() => choose(row.id)}
              onMouseEnter={() => setCursor(index)}
              className={cn(
                "flex w-full cursor-pointer items-center justify-between gap-2 rounded-lg px-2 py-1.5 text-left text-sm",
                !row.id && "text-muted-foreground",
                index === cursor && "bg-accent",
              )}
            >
              <span className="flex min-w-0 flex-col">
                <span className="truncate">{row.name}</span>
                {row.detail && <span className="truncate text-xs text-muted-foreground">{row.detail}</span>}
              </span>
              {row.id === value && <Check className="size-3.5 shrink-0 text-copper" aria-hidden />}
            </li>
          ))}
        </ul>
        {loading && <p className="px-2 py-2 text-sm text-muted-foreground">Loading…</p>}
        {!loading && matches.length === 0 && (
          <p className="px-2 py-2 text-sm text-muted-foreground">Nothing matches that.</p>
        )}
      </PopoverContent>
    </Popover>
  );
}
