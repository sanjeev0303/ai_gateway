import type { ID, ISODateString, StatusTone } from "./common";

export type PlanTier = "free" | "starter" | "growth" | "enterprise";

/** Mirrors the storable org roles in `backend/internal/domain/roles.go`. */
export type OrganizationRole = "owner" | "admin" | "member" | "auditor" | "billing_admin";

/** Mirrors the `organizations.status` CHECK constraint: active | suspended. */
export type OrganizationStatus = "active" | "suspended";

export interface Organization {
  id: ID;
  name: string;
  slug: string;
  status: OrganizationStatus;
  createdAt: ISODateString;
  updatedAt: ISODateString;
}

export interface OrganizationSummary {
  organization: Organization;
  activeUsers?: number;
  activeProjects?: number;
  totalSpendMtd?: number;
  openAlerts?: number;
  policyCompliance?: number;
}

export const PLAN_LABELS: Record<PlanTier, string> = {
  free: "Free",
  starter: "Starter",
  growth: "Growth",
  enterprise: "Enterprise",
};

export const ORG_STATUS_TONE: Record<OrganizationStatus, StatusTone> = {
  active: "success",
  suspended: "danger",
};

export const ORG_STATUS_LABELS: Record<OrganizationStatus, string> = {
  active: "Active",
  suspended: "Suspended",
};
