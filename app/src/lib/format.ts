/** Formatters and small pure helpers shared by every page. */

/**
 * Compact USD for dashboard tiles: $1.2k, $3.4M.
 * Whole dollars only — sub-cent precision is noise at this scale.
 */
export function formatUsd(value: number): string {
  const abs = Math.abs(value);
  if (abs >= 1_000_000) return `$${(value / 1_000_000).toFixed(1)}M`;
  if (abs >= 1_000) return `$${(value / 1_000).toFixed(1)}k`;
  return `$${Math.round(value)}`;
}

/** Full USD with thousands separators, for invoices and detail rows. */
export function formatUsdExact(value: number): string {
  return value.toLocaleString("en-US", {
    style: "currency",
    currency: "USD",
    maximumFractionDigits: 2,
  });
}

export const MICROS_PER_USD = 1_000_000;

export function microsToUsd(micros: number): number {
  return micros / MICROS_PER_USD;
}

/** Token counts stay readable past a million: 1.2M tokens. */
export function formatCompact(value: number): string {
  if (Math.abs(value) >= 1_000_000)
    return `${(value / 1_000_000).toFixed(1)}M`;
  if (Math.abs(value) >= 1_000) return `${(value / 1_000).toFixed(1)}k`;
  return String(Math.round(value));
}

export function formatNumber(value: number): string {
  return value.toLocaleString("en-US");
}

export function formatPercent(value: number, digits = 1): string {
  return `${(value * 100).toFixed(digits)}%`;
}

/** Latency is always milliseconds, or seconds past 1000ms. */
export function formatMs(value: number): string {
  if (value >= 1000) return `${(value / 1000).toFixed(2)}s`;
  return `${Math.round(value)}ms`;
}

export function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    timeZone: "UTC",
  });
}

export function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    timeZone: "UTC",
  });
}

/** "3m ago" style relative time, for live activity and alerts. */
export function formatRelative(iso: string, now: Date = new Date()): string {
  const seconds = Math.round((now.getTime() - new Date(iso).getTime()) / 1000);
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.round(hours / 24)}d ago`;
}

/** Delta already expressed as a fraction, e.g. 0.12 -> "+12.0%". */
export function formatDelta(fraction: number): string {
  const sign = fraction > 0 ? "+" : "";
  return `${sign}${(fraction * 100).toFixed(1)}%`;
}

/** Initials for avatar fallbacks, capped at two characters. */
export function initials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .join("");
}

export function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}
