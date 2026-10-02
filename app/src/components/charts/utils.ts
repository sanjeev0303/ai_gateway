/** Scale and path helpers shared by every chart. No dependencies. */

export interface LinearScale {
  map: (value: number) => number;
  domain: readonly [number, number];
  range: readonly [number, number];
}

export function linearScale(
  domain: readonly [number, number],
  range: readonly [number, number],
): LinearScale {
  const [d0, d1] = domain;
  const [r0, r1] = range;
  const span = d1 - d0;
  return {
    // A flat series has no meaningful position, so centre it instead of dividing by a zero span.
    map: (value) =>
      span === 0 ? (r0 + r1) / 2 : r0 + ((value - d0) / span) * (r1 - r0),
    domain,
    range,
  };
}

export interface Point {
  x: number;
  y: number;
}

/** Round tick values (1/2/5 x 10^n) covering [0, max] for gridlines and axes. */
export function niceTicks(max: number, count = 4): number[] {
  if (!Number.isFinite(max) || max <= 0) return [0];
  const rawStep = max / count;
  const magnitude = 10 ** Math.floor(Math.log10(rawStep));
  const normalized = rawStep / magnitude;
  const step =
    (normalized <= 1 ? 1 : normalized <= 2 ? 2 : normalized <= 5 ? 5 : 10) *
    magnitude;
  const ticks: number[] = [];
  // The epsilon keeps the final tick from being dropped by float error.
  for (let value = 0; value <= max + step * 1e-3; value += step) {
    ticks.push(Math.round(value * 1_000) / 1_000);
  }
  return ticks;
}

/** Extend a domain to the next round tick so the top series is not clipped. */
export function niceMax(max: number, count = 4): number {
  const ticks = niceTicks(max, count);
  return ticks.length > 0 ? ticks[ticks.length - 1]! : max;
}

export function linePath(points: readonly Point[]): string {
  return points
    .map(
      (point, index) =>
        `${index === 0 ? "M" : "L"}${point.x.toFixed(2)},${point.y.toFixed(2)}`,
    )
    .join(" ");
}

/** Closes a line path down to the baseline to make a fillable area. */
export function areaPath(points: readonly Point[], baselineY: number): string {
  if (points.length === 0) return "";
  const first = points[0]!;
  const last = points[points.length - 1]!;
  return `${linePath(points)} L${last.x.toFixed(2)},${baselineY.toFixed(2)} L${first.x.toFixed(2)},${baselineY.toFixed(2)} Z`;
}

/** Thin out x-axis labels so they never overlap. */
export function labelStride(count: number, maxLabels = 6): number {
  return Math.max(1, Math.ceil(count / maxLabels));
}

export function maxOf(points: readonly { value: number }[]): number {
  return points.reduce((acc, point) => Math.max(acc, point.value), 0);
}

export function minOf(points: readonly { value: number }[]): number {
  return points.reduce(
    (acc, point) => Math.min(acc, point.value),
    Number.POSITIVE_INFINITY,
  );
}

/** Default palette, cycling if a chart needs more series than tokens exist. */
export const CHART_COLORS = [
  "var(--color-chart-1)",
  "var(--color-chart-2)",
  "var(--color-chart-3)",
  "var(--color-chart-4)",
  "var(--color-chart-5)",
] as const;

export function chartColor(index: number): string {
  return CHART_COLORS[index % CHART_COLORS.length]!;
}
