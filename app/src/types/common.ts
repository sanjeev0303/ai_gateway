/** Shared primitives used across every domain type. */

export type ID = string;

/** ISO-8601 timestamp string, e.g. "2026-09-29T14:03:00Z". */
export type ISODateString = string;

export type TrendDirection = "up" | "down" | "flat";

export type StatusTone = "success" | "warning" | "danger" | "info" | "neutral";

/** A single point in a time series, used by every chart in the product. */
export interface TimeSeriesPoint {
  /** Bucket label, e.g. "2026-09-29" for daily or "14:00" for hourly. */
  date: string;
  value: number;
}

/** A labelled numeric value, used for categorical bar charts. */
export interface CategoryDatum {
  label: string;
  value: number;
  /** Optional secondary series for stacked/grouped bars. */
  secondary?: number;
}

export interface Paginated<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}
