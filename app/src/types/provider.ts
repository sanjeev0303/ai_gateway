import type { ID, ISODateString, StatusTone } from "./common";

export type ProviderStatus = "healthy" | "degraded" | "outage" | "maintenance";

/** Uptime measured over the trailing 30 days, 0-100. */
export type UptimePercent = number;

export interface Provider {
  id: ID;
  /** Machine slug, e.g. "anthropic". */
  name: string;
  displayName: string;
  status: ProviderStatus;
  uptime30d: UptimePercent;
  /** Median first-token latency in milliseconds. */
  latencyP50Ms: number;
  /** 95th percentile time to first token in milliseconds. */
  latencyP95Ms: number;
  models: ID[];
  createdAt: ISODateString;
  lastIncidentAt?: ISODateString;
  /** Monthly platform fee in whole USD. */
  monthlyFee: number;
}

export const PROVIDER_STATUS_TONE: Record<ProviderStatus, StatusTone> = {
  healthy: "success",
  degraded: "warning",
  outage: "danger",
  maintenance: "info",
};
