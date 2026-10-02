import { costSummary } from "./cost";
import type {
  Organization,
  OrganizationSummary,
} from "@/types";

export const organization: Organization = {
  id: "org_acme",
  name: "Acme Corp",
  slug: "acme",
  plan: "enterprise",
  status: "active",
  createdAt: "2024-03-11T09:00:00Z",
  seatsUsed: 24,
  seatsTotal: 50,
  billingEmail: "ap@acme.example",
  region: "us-east-1",
  mrr: 4_250,
};

export const organizationSummary: OrganizationSummary = {
  organization,
  activeUsers: 21,
  activeProjects: 14,
  totalSpendMtd: costSummary.mtd,
  openAlerts: 5,
  policyCompliance: 96.4,
};
