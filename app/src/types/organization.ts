import type { ID, ISODateString, StatusTone } from "./common";

export type PlanTier = "free" | "starter" | "growth" | "enterprise";

export type OrganizationRole = "owner" | "admin" | "member" | "viewer";

export type OrganizationStatus = "active" | "trialing" | "past_due" | "suspended";

export interface Organization {
  id: ID;
  name: string;
  slug: string;
  logoUrl?: string;
  plan: PlanTier;
  status: OrganizationStatus;
  createdAt: ISODateString;
  seatsUsed: number;
  seatsTotal: number;
  /** Billing contact, not necessarily a platform user. */
  billingEmail: string;
  region: string;
  /** Annual recurring revenue in whole USD. */
  mrr: number;
}

export interface OrganizationSummary {
  organization: Organization;
  activeUsers: number;
  activeProjects: number;
  totalSpendMtd: number;
  openAlerts: number;
  policyCompliance: number;
}

export const PLAN_LABELS: Record<PlanTier, string> = {
  free: "Free",
  starter: "Starter",
  growth: "Growth",
  enterprise: "Enterprise",
};

export const ORG_STATUS_TONE: Record<OrganizationStatus, StatusTone> = {
  active: "success",
  trialing: "info",
  past_due: "warning",
  suspended: "danger",
};
