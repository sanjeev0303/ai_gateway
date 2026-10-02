import type { TimeSeriesPoint } from "@/types";
import { formatCompact } from "@/lib/format";
import {
  labelStride,
  linePath,
  linearScale,
  maxOf,
  niceMax,
  niceTicks,
  type Point,
} from "./utils";

interface LineChartProps {
  data: readonly TimeSeriesPoint[];
  /** CSS colour or a `var(--color-chart-n)` token. */
  color?: string;
  /** Viewport height in SVG units; the width scales with the container. */
  height?: number;
  label?: string;
  formatValue?: (value: number) => string;
}

const WIDTH = 720;
const PADDING = { top: 16, right: 12, bottom: 28, left: 48 } as const;

/** Single-series line over time, with round gridlines and thinned x labels. */
export function LineChart({
  data,
  color = "var(--color-chart-1)",
  height = 240,
  label = "Line chart",
  formatValue = formatCompact,
}: LineChartProps) {
  if (data.length === 0) {
    return (
      <p className="py-12 text-center text-sm text-muted-foreground">
        No data for this range.
      </p>
    );
  }

  const top = niceMax(maxOf(data));
  const plotWidth = WIDTH - PADDING.left - PADDING.right;
  const plotHeight = height - PADDING.top - PADDING.bottom;
  const baseline = PADDING.top + plotHeight;

  const x = linearScale([0, Math.max(1, data.length - 1)], [PADDING.left, PADDING.left + plotWidth]);
  const y = linearScale([0, top], [baseline, PADDING.top]);
  const points: Point[] = data.map((point, index) => ({
    x: x.map(index),
    y: y.map(point.value),
  }));

  const stride = labelStride(data.length);
  const ticks = niceTicks(top);

  return (
    <svg
      viewBox={`0 0 ${WIDTH} ${height}`}
      className="h-auto w-full"
      role="img"
      aria-label={label}
    >
      <title>{label}</title>
      {ticks.map((tick) => (
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

      <path
        d={linePath(points)}
        fill="none"
        stroke={color}
        strokeWidth={2}
        strokeLinecap="round"
        strokeLinejoin="round"
      />

      {points.map((point, index) =>
        index % stride === 0 ? (
          <text
            key={data[index]!.date}
            x={point.x}
            y={baseline + 18}
            textAnchor="middle"
            className="fill-muted-foreground text-[10px]"
          >
            {data[index]!.date.slice(5)}
          </text>
        ) : null,
      )}
    </svg>
  );
}
