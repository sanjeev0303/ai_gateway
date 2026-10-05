export type DateRange = "24h" | "7d" | "30d" | "90d";

export const DATE_RANGES: readonly DateRange[] = ["24h", "7d", "30d", "90d"];

export const DEFAULT_RANGE: DateRange = "30d";

export function isDateRange(value: string | null): value is DateRange {
  return value !== null && (DATE_RANGES as readonly string[]).includes(value);
}
