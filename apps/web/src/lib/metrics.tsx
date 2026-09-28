import type { ReactNode } from "react";

export const NO_DATA = "No usage recorded yet";

export const formatCount = (n: number): string => n.toLocaleString("en-US");

export const formatUsd = (n: number): string =>
  n > 0 && n < 0.01 ? `$${n.toFixed(4)}` : `$${n.toFixed(2)}`;

export const formatPercent = (n: number): string => `${(n * 100).toFixed(1)}%`;

export const formatMs = (n: number): string => `${Math.round(n).toLocaleString("en-US")} ms`;

export type MetricFormat = (n: number) => string;

interface KpiCardProps {
  readonly label: string;
  readonly current: number | null;
  readonly previous: number | null;
  readonly format: MetricFormat;
  readonly lowerIsBetter: boolean;
}

export function KpiCard({ label, current, previous, format, lowerIsBetter }: KpiCardProps) {
  return (
    <div className="kpi">
      <p className="kpi-label">{label}</p>
      {current === null ? (
        <p className="kpi-value kpi-value-none">{NO_DATA}</p>
      ) : (
        <p className="kpi-value">{format(current)}</p>
      )}
      <Trend current={current} previous={previous} format={format} lowerIsBetter={lowerIsBetter} />
    </div>
  );
}

interface TrendProps {
  readonly current: number | null;
  readonly previous: number | null;
  readonly format: MetricFormat;
  readonly lowerIsBetter?: boolean;
}

function Trend({ current, previous, format, lowerIsBetter }: TrendProps) {
  if (current === null) {
    return <p className="kpi-trend kpi-trend-none">No current period to compare</p>;
  }
  if (previous === null) {
    return <p className="kpi-trend kpi-trend-none">No previous period recorded</p>;
  }
  if (previous === 0) {
    return <p className="kpi-trend kpi-trend-none">Previous period was 0</p>;
  }
  const delta = (current - previous) / previous;
  const direction = delta > 0 ? "up" : delta < 0 ? "down" : "flat";
  const quality = delta === 0 || lowerIsBetter === undefined ? "flat" : (delta < 0) === lowerIsBetter ? "good" : "bad";
  const arrow = delta > 0 ? "▲" : delta < 0 ? "▼" : "▬";
  return (
    <p className="kpi-trend" data-dir={direction} data-quality={quality}>
      <span aria-hidden>{arrow}</span> {Math.abs(delta * 100).toFixed(1)}% vs previous{" "}
      {format(previous)}
    </p>
  );
}

interface UnavailablePanelProps {
  readonly what: string;
  readonly needs: readonly string[];
  readonly children?: ReactNode;
}

export function UnavailablePanel({ what, needs, children }: UnavailablePanelProps) {
  return (
    <div className="gap">
      <p className="gap-title">{what} is not available</p>
      <p className="gap-body">
        The control API exposes no endpoint that can answer this, so the dashboard reports nothing
        rather than estimating a figure. A blank here means unknown, not zero.
      </p>
      {needs.length > 0 && (
        <p className="gap-needs">
          Needs <span className="gap-endpoints">{needs.map((n) => <code key={n} className="mono">{n}</code>)}</span>
        </p>
      )}
      {children}
    </div>
  );
}
