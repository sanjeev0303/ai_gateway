import type { CategoryDatum } from "@/types";
import { formatCompact } from "@/lib/format";
import { chartColor } from "./utils";

interface BarChartProps {
  data: readonly CategoryDatum[];
  /** Minimum row height in pixels. */
  rowHeight?: number;
  label?: string;
  formatValue?: (value: number) => string;
  /** Render `secondary` as a second, lighter bar per row. */
  showSecondary?: boolean;
}

const percentOf = (value: number, max: number): string => `${max > 0 ? Math.max(0, (value / max) * 100) : 0}%`;

/**
 * Horizontal bars ranked by value, keeping long model and team names readable.
 * Plain HTML rather than SVG, so labels stay real text at any width: on a phone
 * the name sits above its bar instead of being clipped beside it.
 */
export function BarChart({
  data,
  rowHeight = 30,
  label = "Bar chart",
  formatValue = formatCompact,
  showSecondary = true,
}: BarChartProps) {
  if (data.length === 0) {
    return (
      <p className="py-12 text-center text-sm text-muted-foreground">
        No data to rank yet.
      </p>
    );
  }

  const primaryMax = data.reduce((acc, datum) => Math.max(acc, datum.value), 0);
  const secondaryMax = data.reduce((acc, datum) => Math.max(acc, datum.secondary ?? 0), 0);
  const hasSecondary = showSecondary && data.some((datum) => datum.secondary);

  return (
    <ul aria-label={label} className="flex flex-col gap-2.5 sm:gap-1">
      {data.map((datum) => (
        <li
          key={datum.label}
          style={{ minHeight: rowHeight }}
          className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-1 sm:grid-cols-[minmax(0,10rem)_minmax(0,1fr)_5rem]"
        >
          <span title={datum.label} className="truncate text-[0.8125rem] font-medium sm:text-right">
            {datum.label}
          </span>
          <span className="col-span-2 row-start-2 flex flex-col gap-0.5 overflow-hidden rounded-[3px] bg-muted sm:col-span-1 sm:col-start-2 sm:row-start-1">
            <span
              className="block h-2.5 rounded-[3px]"
              style={{ width: percentOf(datum.value, primaryMax), backgroundColor: chartColor(0) }}
            />
            {hasSecondary && (
              <span
                className="block h-1.5 rounded-[3px] opacity-60"
                style={{ width: percentOf(datum.secondary ?? 0, secondaryMax), backgroundColor: chartColor(1) }}
              />
            )}
          </span>
          <span className="text-right text-[0.8125rem] text-muted-foreground tabular-nums">
            {formatValue(datum.value)}
          </span>
        </li>
      ))}
    </ul>
  );
}
