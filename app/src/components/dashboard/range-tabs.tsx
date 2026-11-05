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
        className="liquid-glass inline-flex h-9 items-center rounded-full border p-[3px]"
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
                "h-full rounded-full px-3 text-xs font-medium tabular-nums transition-[color,background-color] duration-150",
                "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                isSelected
                  ? "bg-primary text-primary-foreground shadow-[0_1px_2px_rgba(20,26,51,0.18)]"
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
