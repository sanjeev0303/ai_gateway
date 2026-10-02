import type { CategoryDatum, ID, TimeSeriesPoint } from "./common";

export type UsageGranularity = "hour" | "day" | "week" | "month";

export interface UsageTotals {
  totalCalls: number;
  inputTokens: number;
  outputTokens: number;
  totalTokens: number;
  /** Whole USD. */
  cost: number;
  p50LatencyMs: number;
  p95LatencyMs: number;
  errorRate: number;
}

export interface UsageSeries {
  granularity: UsageGranularity;
  totals: UsageTotals;
  calls: TimeSeriesPoint[];
  tokens: TimeSeriesPoint[];
  cost: TimeSeriesPoint[];
  /** Token split by direction, for the stacked area chart. */
  inputTokens: TimeSeriesPoint[];
  outputTokens: TimeSeriesPoint[];
}

export interface UsageByModel extends CategoryDatum {
  modelId: ID;
  cost: number;
  calls: number;
}

export interface UsageByProvider extends CategoryDatum {
  providerId: ID;
  cost: number;
  calls: number;
}

export interface UsageBreakdown {
  byModel: UsageByModel[];
  byProvider: UsageByProvider[];
  byProject: CategoryDatum[];
  byTeam: CategoryDatum[];
  byUser: CategoryDatum[];
}
