import type { CategoryDatum } from "@/types";
import { formatCompact } from "@/lib/format";
import { chartColor } from "./utils";

interface DonutChartProps {
  data: readonly CategoryDatum[];
  size?: number;
  thickness?: number;
  /** Small caption under the total in the hole. */
  centerLabel?: string;
  formatValue?: (value: number) => string;
}

/**
 * Donut built from dashed circle strokes rather than arc paths — one element
 * per segment, and no trigonometry in the render.
 */
export function DonutChart({
  data,
  size = 200,
  thickness = 26,
  centerLabel,
  formatValue = formatCompact,
}: DonutChartProps) {
  const total = data.reduce((acc, datum) => acc + datum.value, 0);

  if (total <= 0) {
    return (
      <p className="py-12 text-center text-sm text-muted-foreground">
        No usage recorded yet.
      </p>
    );
  }

  const radius = (size - thickness) / 2;
  const circumference = 2 * Math.PI * radius;

  const segments = data.reduce<{ dash: number; offset: number }[]>(
    (acc, datum) => {
      const previous = acc[acc.length - 1];
      const offset = previous ? previous.offset + previous.dash : 0;
      acc.push({ dash: (datum.value / total) * circumference, offset });
      return acc;
    },
    [],
  );

  return (
    <div className="flex flex-wrap items-center gap-6">
      <svg
        viewBox={`0 0 ${size} ${size}`}
        style={{ width: size, height: size }}
        role="img"
        aria-label="Usage breakdown"
        className="shrink-0"
      >
        <title>Usage breakdown</title>
        {/* Start every segment at 12 o'clock instead of 3 o'clock. */}
        <g transform={`rotate(-90 ${size / 2} ${size / 2})`}>
          {data.map((datum, index) => {
            const { dash, offset } = segments[index];
            return (
              <circle
                key={datum.label}
                cx={size / 2}
                cy={size / 2}
                r={radius}
                fill="none"
                strokeWidth={thickness}
                stroke={chartColor(index)}
                strokeDasharray={`${dash} ${circumference - dash}`}
                strokeDashoffset={-offset}
              />
            );
          })}
        </g>
        <text
          x={size / 2}
          y={size / 2 - 4}
          textAnchor="middle"
          className="fill-foreground text-lg font-semibold"
        >
          {formatValue(total)}
        </text>
        {centerLabel && (
          <text
            x={size / 2}
            y={size / 2 + 16}
            textAnchor="middle"
            className="fill-muted-foreground text-xs font-medium"
          >
            {centerLabel}
          </text>
        )}
      </svg>

      <ul className="min-w-40 flex-1 space-y-2">
        {data.map((datum, index) => (
          <li
            key={datum.label}
            className="flex items-center gap-2 text-sm"
          >
            <span
              aria-hidden
              className="size-2.5 shrink-0 rounded-full"
              style={{ backgroundColor: chartColor(index) }}
            />
            <span className="min-w-0 flex-1 truncate text-muted-foreground">
              {datum.label}
            </span>
            <span className="shrink-0 tabular-nums font-medium">
              {formatValue(datum.value)}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}
