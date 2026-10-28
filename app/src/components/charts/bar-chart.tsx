import type { CategoryDatum } from "@/types";
import { formatCompact } from "@/lib/format";
import { chartColor, linearScale } from "./utils";

interface BarChartProps {
  data: readonly CategoryDatum[];
  /** Row pitch in SVG units; total height derives from the row count. */
  rowHeight?: number;
  label?: string;
  formatValue?: (value: number) => string;
  /** Render `secondary` as a second, lighter bar per row. */
  showSecondary?: boolean;
}

const WIDTH = 720;
const LABEL_WIDTH = 150;
const VALUE_WIDTH = 76;
const PADDING = { top: 4, right: VALUE_WIDTH, left: LABEL_WIDTH } as const;

const truncate = (value: string, max = 22): string =>
  value.length > max ? `${value.slice(0, max - 1)}…` : value;

/**
 * Horizontal bars ranked by value. Horizontal keeps long model and team names
 * readable, which a vertical chart would have to rotate.
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

  const plotWidth = WIDTH - PADDING.left - PADDING.right;
  const height = data.length * rowHeight + PADDING.top * 2;
  const primaryMax = data.reduce((acc, datum) => Math.max(acc, datum.value), 0);
  const secondaryMax = data.reduce(
    (acc, datum) => Math.max(acc, datum.secondary ?? 0),
    0,
  );

  const primary = linearScale([0, primaryMax || 1], [0, plotWidth]);
  const secondary = linearScale([0, secondaryMax || 1], [0, plotWidth]);
  const hasSecondary = showSecondary && data.some((datum) => datum.secondary);

  return (
    <svg
      viewBox={`0 0 ${WIDTH} ${height}`}
      className="h-auto w-full"
      role="img"
      aria-label={label}
    >
      <title>{label}</title>
      {data.map((datum, index) => {
        const rowTop = PADDING.top + index * rowHeight;
        const barHeight = hasSecondary ? 8 : 12;
        return (
          <g key={datum.label}>
            <text
              x={PADDING.left - 10}
              y={rowTop + rowHeight / 2}
              textAnchor="end"
              dominantBaseline="middle"
              className="fill-foreground text-xs font-medium"
            >
              {truncate(datum.label)}
            </text>

            <rect
              x={PADDING.left}
              y={rowTop + (rowHeight - barHeight * (hasSecondary ? 2 : 1) - 2) / 2}
              width={plotWidth}
              height={barHeight * (hasSecondary ? 2 : 1) + 2}
              rx={3}
              className="fill-muted"
            />

            <rect
              x={PADDING.left}
              y={rowTop + (rowHeight - barHeight * (hasSecondary ? 2 : 1) - 2) / 2}
              width={primary.map(datum.value)}
              height={barHeight}
              rx={3}
              fill={chartColor(0)}
            />

            {hasSecondary && datum.secondary !== undefined && (
              <rect
                x={PADDING.left}
                y={rowTop + rowHeight / 2 + 1}
                width={secondary.map(datum.secondary)}
                height={barHeight}
                rx={3}
                fill={chartColor(1)}
                fillOpacity={0.6}
              />
            )}

            <text
              x={PADDING.left + plotWidth + 8}
              y={rowTop + rowHeight / 2}
              dominantBaseline="middle"
              className="fill-muted-foreground text-xs font-mono"
            >
              {formatValue(datum.value)}
            </text>
          </g>
        );
      })}
    </svg>
  );
}
