export type DateRange = "24h" | "7d" | "30d" | "90d";

export const DATE_RANGES: readonly DateRange[] = ["24h", "7d", "30d", "90d"];

export const DEFAULT_RANGE: DateRange = "30d";

export function isDateRange(value: string | null): value is DateRange {
  return value !== null && (DATE_RANGES as readonly string[]).includes(value);
}

/** 30 days back from now. Usage summary and latency default to 24 hours, so pages
 * whose cards say "30d" pass this as `from`. */
export function thirtyDaysAgo(): Date {
  return new Date(Date.now() - 30 * 24 * 60 * 60 * 1000);
}
