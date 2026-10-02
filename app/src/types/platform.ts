import type { ID, ISODateString, StatusTone, TimeSeriesPoint } from "./common";

/** Ingestion and query health for the telemetry pipeline itself. */
export interface PipelineStatus {
  component: string;
  status: "operational" | "degraded" | "outage";
  /** Percentage of events landing within the freshness SLA. */
  freshnessSlaPercent: number;
  /** Median seconds between an upstream call and its appearance in the UI. */
  ingestLagP50Seconds: number;
  eventsPerSecond: number;
}

export interface PlatformHealth {
  status: "operational" | "degraded" | "partial_outage" | "major_outage";
  apiLatencyP50Ms: number;
  apiLatencyP95Ms: number;
  apiErrorRate: number;
  /** 0-100 composite used for the status badge. */
  overallScore: number;
  pipelines: PipelineStatus[];
  /** Rolling 30d API request volume. */
  requestVolume: TimeSeriesPoint[];
  updatedAt: ISODateString;
}

/** A platform-originated event, distinct from user activity. */
export interface PlatformEvent {
  id: ID;
  occurredAt: ISODateString;
  severity: "info" | "warning" | "error";
  source: string;
  message: string;
  /** Dashed path to the relevant dashboard page. */
  link: string;
}

export const PIPELINE_STATUS_TONE: Record<
  PipelineStatus["status"],
  StatusTone
> = {
  operational: "success",
  degraded: "warning",
  outage: "danger",
};
