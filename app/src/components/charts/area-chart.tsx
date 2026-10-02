import type { TimeSeriesPoint } from "@/types";
import { formatCompact } from "@/lib/format";
import {
  areaPath,
  labelStride,
  linearScale,
  niceMax,
  niceTicks,
  type Point,
} from "./utils";

export interface AreaSeries {
  label: string;
  /** CSS colour or a `var(--color-chart-n)` token. */
  color: string;
  data: readonly TimeSeriesPoint[];
}

interface AreaChartProps {
  series: readonly AreaSeries[];
  /** Stack each series on the previous one instead of overlapping them. */
  stacked?: boolean;
  height?: number;
  label?: string;
  formatValue?: (value: number) => string;
}

const WIDTH = 720;
const PADDING = { top: 16, right: 12, bottom: 28, left: 48 } as const;

/** One or more filled series over time; the shared shape for token splits. */
export function AreaChart({
  series,
  stacked = true,
  height = 260,
  label = "Area chart",
  formatValue = formatCompact,
}: AreaChartProps) {
  const length = series[0]?.data.length ?? 0;

  if (length === 0) {
    return (
      <p className="py-12 text-center text-sm text-muted-foreground">
        No data for this range.
      </p>
    );
  }

  const plotWidth = WIDTH - PADDING.left - PADDING.right;
  const plotHeight = height - PADDING.top - PADDING.bottom;
  const baseline = PADDING.top + plotHeight;

  const x = linearScale([0, Math.max(1, length - 1)], [PADDING.left, PADDING.left + plotWidth]);

  // Stacked series need the running total, otherwise the widest single band.
  const cumulative: number[][] = [];
  let running = new Array<number>(length).fill(0);
  let peak = 0;
  for (const entry of series) {
    running = running.map((value, index) => {
      const next = stacked ? value + (entry.data[index]?.value ?? 0) : Math.max(value, entry.data[index]?.value ?? 0);
      return next;
    });
    peak = Math.max(peak, ...running);
    cumulative.push([...running]);
  }

  const top = niceMax(peak);
  const y = linearScale([0, top], [baseline, PADDING.top]);
  const stride = labelStride(length);

  return (
    <svg
      viewBox={`0 0 ${WIDTH} ${height}`}
      className="h-auto w-full"
      role="img"
      aria-label={label}
    >
      <title>{label}</title>
      {niceTicks(top).map((tick) => (
        <g key={tick}>
          <line
            x1={PADDING.left}
            x2={PADDING.left + plotWidth}
            y1={y.map(tick)}
            y2={y.map(tick)}
            stroke="var(--border)"
            strokeDasharray="3 3"
          />
          <text
            x={PADDING.left - 8}
            y={y.map(tick)}
            textAnchor="end"
            dominantBaseline="middle"
            className="fill-muted-foreground text-[10px]"
          >
            {formatValue(tick)}
          </text>
        </g>
      ))}

      {series.map((entry, seriesIndex) => {
        const totals = cumulative[seriesIndex]!;
        const below = seriesIndex === 0 ? null : cumulative[seriesIndex - 1]!;
        const points: Point[] = entry.data.map((point, index) => ({
          x: x.map(index),
          y: y.map(stacked ? totals[index]! : point.value),
        }));
        const areaFloor = below
          ? below.map((value) => y.map(value))
          : new Array<number>(length).fill(baseline);

        // Each band closes against the top of the band beneath it.
        const band = points
          .map((point) => `${point.x.toFixed(2)},${point.y.toFixed(2)}`)
          .join(" L");
        const first = points[0]!;
        const last = points[points.length - 1]!;
        const d =
          points.length === 0
            ? ""
            : `M${band} L${last.x.toFixed(2)},${areaFloor[length - 1]!.toFixed(2)} L${first.x.toFixed(2)},${areaFloor[0]!.toFixed(2)} Z`;

        return (
          <g key={entry.label}>
            <path d={d} fill={entry.color} fillOpacity={0.75} />
            {!stacked && (
              <path
                d={areaPath(points, baseline)}
                fill={entry.color}
                fillOpacity={0.18}
              />
            )}
          </g>
        );
      })}

      {Array.from({ length: length }, (_, index) => index)
        .filter((index) => index % stride === 0)
        .map((index) => (
          <text
            key={index}
            x={x.map(index)}
            y={baseline + 18}
            textAnchor="middle"
            className="fill-muted-foreground text-[10px]"
          >
            {(series[0]!.data[index]?.date ?? "").slice(5)}
          </text>
        ))}
    </svg>
  );
}
