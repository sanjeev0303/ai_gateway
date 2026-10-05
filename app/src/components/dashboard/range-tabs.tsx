"use client";

import { usePathname, useRouter, useSearchParams } from "next/navigation";

import { cn } from "@/lib/utils";

import { type DateRange, DATE_RANGES, DEFAULT_RANGE, isDateRange } from "@/lib/date-range";

const RANGE_LABEL: Record<DateRange, string> = {
  "24h": "24h",
  "7d": "7d",
  "30d": "30d",
  "90d": "90d",
};

/**
 * Date-range selector for analytics surfaces. The choice is held in the `range`
 * query parameter so the server component re-queries the backend for that
 * window, rather than filtering an already-fetched dataset in the browser.
 */
export function RangeTabs({ className }: { className?: string }) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const current = searchParams.get("range");
  const selected: DateRange = isDateRange(current) ? current : DEFAULT_RANGE;

  const select = (value: DateRange) => {
    const params = new URLSearchParams(searchParams.toString());
    // The default range stays implicit, keeping shared links clean.
    if (value === DEFAULT_RANGE) params.delete("range");
    else params.set("range", value);
    const query = params.toString();
    router.push(query ? `${pathname}?${query}` : pathname, { scroll: false });
  };

  return (
    <div className={cn("flex flex-wrap items-center gap-2", className)}>
      <div
        role="group"
        aria-label="Date range"
        className="inline-flex items-center rounded-lg border border-border bg-card p-0.5"
      >
        {DATE_RANGES.map((value) => {
          const isSelected = value === selected;
          return (
            <button
              key={value}
              type="button"
              aria-pressed={isSelected}
              onClick={() => select(value)}
              className={cn(
                "rounded-md px-2.5 py-1 text-xs font-medium transition-colors",
                "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                isSelected
                  ? "bg-foreground text-background"
                  : "text-muted-foreground hover:text-foreground",
              )}
            >
              {RANGE_LABEL[value]}
            </button>
          );
        })}
      </div>
    </div>
  );
}
