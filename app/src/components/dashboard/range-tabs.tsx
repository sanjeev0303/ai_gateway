"use client";

import { useDemoStore } from "@/lib/demo/store";
import { DATE_RANGES } from "@/lib/demo/types";
import type { DateRange } from "@/lib/demo/types";
import { cn } from "@/lib/utils";

const RANGE_LABEL: Record<DateRange, string> = {
  "24h": "24h",
  "7d": "7d",
  "30d": "30d",
  "90d": "90d",
};

const ALL_LABEL = "all";

/**
 * Range selector shared by every analytics surface. Reads and writes the store,
 * so changing it here is changing it everywhere.
 */
export function RangeTabs({ className }: { className?: string }) {
  const { range, setRange, isFiltered, reset, activeFilterCount } = useDemoStore();

  return (
    <div className={cn("flex flex-wrap items-center gap-2", className)}>
      <div
        role="group"
        aria-label="Date range"
        className="inline-flex items-center rounded-lg border border-border bg-card p-0.5"
      >
        {DATE_RANGES.map((value) => {
          const selected = value === range;
          return (
            <button
              key={value}
              type="button"
              aria-pressed={selected}
              onClick={() => setRange(value)}
              className={cn(
                "rounded-md px-2.5 py-1 text-xs font-medium transition-colors",
                "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                selected
                  ? "bg-foreground text-background"
                  : "text-muted-foreground hover:text-foreground",
              )}
            >
              {RANGE_LABEL[value]}
            </button>
          );
        })}
      </div>

      {isFiltered && (
        <button
          type="button"
          onClick={reset}
          className="inline-flex items-center gap-1 rounded-md border border-border px-2.5 py-1 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          Clear filters
          {activeFilterCount > 0 && (
            <span className="rounded-full bg-muted px-1.5 py-0.5 text-[10px] tabular-nums">
              {activeFilterCount}
            </span>
          )}
        </button>
      )}
    </div>
  );
}

export { ALL_LABEL };
