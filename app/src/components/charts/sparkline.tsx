import type { TimeSeriesPoint } from "@/types";
import {
  areaPath,
  linearScale,
  linePath,
  maxOf,
  minOf,
  type Point,
} from "./utils";

interface SparklineProps {
  data: readonly TimeSeriesPoint[];
  /** Inherits the surrounding text colour when omitted. */
  color?: string;
  width?: number;
  height?: number;
  label?: string;
  /** Draw the soft fill under the line. */
  filled?: boolean;
}

/**
 * Wordless trend line for stat tiles. Sized in pixels rather than scaled to its
 * container, so it never distorts the dashboard grid.
 */
export function Sparkline({
  data,
  color = "currentColor",
  width = 120,
  height = 32,
  label = "Trend",
  filled = true,
}: SparklineProps) {
  if (data.length < 2) {
    return <div style={{ width, height }} aria-hidden />;
  }

  const low = minOf(data);
  const high = maxOf(data);

  const x = linearScale([0, data.length - 1], [1, width - 1]);
  const y = linearScale([low, high], [height - 2, 2]);
  const points: Point[] = data.map((point, index) => ({
    x: x.map(index),
    y: y.map(point.value),
  }));

  return (
    <svg
      width={width}
      height={height}
      viewBox={`0 0 ${width} ${height}`}
      role="img"
      aria-label={label}
    >
      <title>{label}</title>
      {filled && (
        <path d={areaPath(points, height)} fill={color} fillOpacity={0.12} />
      )}
      <path
        d={linePath(points)}
        fill="none"
        stroke={color}
        strokeWidth={1.5}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}
