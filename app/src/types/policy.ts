import type { ID, ISODateString, StatusTone } from "./common";

export type PolicyEnforcement = "enforce" | "audit" | "disabled";

export type PolicyScope = "organization" | "team" | "project";

export type PolicyCategory =
  | "pii_redaction"
  | "prompt_injection"
  | "jailbreak"
  | "toxicity"
  | "data_residency"
  | "cost_cap"
  | "model_allowlist"
  | "retention";

export interface Policy {
  id: ID;
  name: string;
  description: string;
  category: PolicyCategory;
  enforcement: PolicyEnforcement;
  scope: PolicyScope;
  /** Ids of teams or projects this policy is bound to. */
  targetIds: ID[];
  enabled: boolean;
  createdAt: ISODateString;
  updatedAt: ISODateString;
  /** Rolling 24h count of requests this policy inspected. */
  evaluations24h: number;
  /** Rolling 24h count of requests this policy blocked. */
  blocks24h: number;
  /** Ids of the authors who last changed the policy. */
  createdById: ID;
}

export const POLICY_ENFORCEMENT_TONE: Record<PolicyEnforcement, StatusTone> = {
  enforce: "danger",
  audit: "info",
  disabled: "neutral",
};
