import type { CategoryDatum, TimeSeriesPoint } from "./common";
import type { ModelModality } from "./model";

export interface CostForecast {
  /** Whole USD projected at period end. */
  projected: number;
  /** Whole USD actually consumed so far this period. */
  actual: number;
  budget: number | null;
  /** Projected overspend in whole USD; 0 when within budget. */
  projectedOverspend: number;
}

export interface CostSummary {
  mtd: number;
  lastMonth: number;
  /** Fractional change, e.g. 0.12 for +12%. */
  changePct: number;
  forecast: CostForecast;
  /** Whole USD per million tokens, blended across all traffic. */
  blendedCostPerMTok: number;
}

export interface CostSeries {
  dailySpend: TimeSeriesPoint[];
  byModel: CategoryDatum[];
  byProvider: CategoryDatum[];
  byTeam: CategoryDatum[];
  byProject: CategoryDatum[];
  byEnvironment: CategoryDatum[];
}

export interface CostByModality extends CategoryDatum {
  modality: ModelModality;
}
