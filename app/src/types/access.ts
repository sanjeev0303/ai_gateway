import type { ID, ISODateString, StatusTone } from "./common";
import type { PlanTier } from "./organization";

/** A rule granting a set of principals access to a set of models. */
export interface ModelAccessGrant {
  id: ID;
  name: string;
  modelIds: ID[];
  /** Ids of teams or projects granted access; empty means organization-wide. */
  principalIds: ID[];
  scope: "organization" | "team" | "project";
  status: "active" | "pending" | "expired";
  createdAt: ISODateString;
  expiresAt?: ISODateString;
  /** Requests admitted in the last 24h that this grant was required for. */
  requests24h: number;
}

/** A model a given principal is currently blocked from calling. */
export interface ModelDenial {
  id: ID;
  principalName: string;
  modelName: string;
  reason: string;
  blockedAt: ISODateString;
  requestsBlocked24h: number;
}

export const GRANT_STATUS_TONE: Record<
  ModelAccessGrant["status"],
  StatusTone
> = {
  active: "success",
  pending: "info",
  expired: "neutral",
};

/** Plan-gated model entitlements, rendered on the model-access page. */
export interface ModelEntitlement {
  tier: PlanTier;
  modelNames: string[];
  /** Whole USD monthly cap for the tier, or null when uncapped. */
  spendCap: number | null;
  requiresApproval: boolean;
}
