"use client";

import type { ReactNode } from "react";

import type { ReadPermission } from "@/lib/control-api";
import {
  ApiCredentialsBody,
  GatewayCredentialsBody,
  InvitationsBody,
  MembersBody,
  ModelsBody,
  ProjectsBody,
  ProviderAccountsBody,
  ProviderHealthBody,
  ServiceAccountsBody,
  TeamsBody,
} from "@/lib/sections";
import {
  CostsBody,
  ConversationsBody,
  LiveActivityBody,
  UsageBody,
} from "@/lib/bodies-observe";
import {
  ModelsObserveBody,
  ProjectsObserveBody,
  TeamsObserveBody,
  UsersBody,
} from "@/lib/bodies-entities";
import { ProvidersObserveBody } from "@/lib/bodies-providers";
import {
  BudgetsLimitsBody,
  CredentialsBody,
  GuardrailsBody,
  ModelAccessBody,
  PoliciesBody,
} from "@/lib/bodies-govern";
import { AlertsBody, AuditLogsBody, SecurityEventsBody } from "@/lib/bodies-security";
import { ApiDocsBody, CliBody, QuickStartBody, SdksBody } from "@/lib/bodies-developer";
import { InvoicesBody, SubscriptionBody, UsageAndSpendBody } from "@/lib/bodies-billing";

export type Group =
  | "Observe"
  | "Govern"
  | "Security"
  | "Platform"
  | "Organization"
  | "Billing"
  | "Developer";

export interface SectionDef {
  readonly slug: string;
  readonly group: Group;
  readonly title: string;
  readonly blurb: string;
  readonly requires: ReadPermission;
  readonly Body: () => ReactNode;
}

export const SECTIONS: readonly SectionDef[] = [
  { slug: "live-activity", group: "Observe", title: "Live Activity", requires: "usage.read",
    blurb: "Streaming gateway activity and request inspector.", Body: LiveActivityBody },
  { slug: "usage", group: "Observe", title: "Usage", requires: "usage.read",
    blurb: "Filter usage by every request dimension.", Body: UsageBody },
  { slug: "costs", group: "Observe", title: "Costs", requires: "usage.read",
    blurb: "Derived spend and cost composition.", Body: CostsBody },
  { slug: "models", group: "Observe", title: "Models", requires: "providers.read",
    blurb: "Model usage, error rate, and latency.", Body: ModelsObserveBody },
  { slug: "providers", group: "Observe", title: "Providers", requires: "providers.read",
    blurb: "Provider health and consumption.", Body: ProvidersObserveBody },
  { slug: "users", group: "Observe", title: "Users", requires: "members.read",
    blurb: "Per-user activity and detail.", Body: UsersBody },
  { slug: "teams", group: "Observe", title: "Teams", requires: "teams.read",
    blurb: "Per-team consumption.", Body: TeamsObserveBody },
  { slug: "projects", group: "Observe", title: "Projects", requires: "projects.read",
    blurb: "Per-project consumption.", Body: ProjectsObserveBody },
  { slug: "conversations", group: "Observe", title: "Conversations", requires: "usage.read",
    blurb: "Conversation index and privacy mode.", Body: ConversationsBody },

  { slug: "policies", group: "Govern", title: "Policies", requires: "policies.read",
    blurb: "Visual policy builder and compiled preview.", Body: PoliciesBody },
  { slug: "model-access", group: "Govern", title: "Model Access", requires: "policies.read",
    blurb: "Who may call which model.", Body: ModelAccessBody },
  { slug: "budgets-limits", group: "Govern", title: "Budgets & Limits", requires: "policies.read",
    blurb: "Rate limits, token and spend budgets.", Body: BudgetsLimitsBody },
  { slug: "guardrails", group: "Govern", title: "Guardrails", requires: "policies.read",
    blurb: "Content controls per scope.", Body: GuardrailsBody },
  { slug: "credentials", group: "Govern", title: "Credentials", requires: "credentials.read",
    blurb: "Provider, gateway, and service account credentials.", Body: CredentialsBody },

  { slug: "audit-logs", group: "Security", title: "Audit Logs", requires: "audit.read",
    blurb: "Dense audit trail with filters.", Body: AuditLogsBody },
  { slug: "alerts", group: "Security", title: "Alerts", requires: "alerts.read",
    blurb: "Alert rules and lifecycle.", Body: AlertsBody },
  { slug: "security-events", group: "Security", title: "Security Events", requires: "alerts.read",
    blurb: "Policy denials and credential anomalies.", Body: SecurityEventsBody },

  { slug: "provider-health", group: "Platform", title: "Provider Health", requires: "providers.read",
    blurb: "Live provider connectivity checks.", Body: ProviderHealthBody },
  { slug: "provider-accounts", group: "Platform", title: "Provider Accounts", requires: "providers.read",
    blurb: "Provider connections and endpoints.", Body: ProviderAccountsBody },
  { slug: "service-accounts", group: "Platform", title: "Service Accounts", requires: "credentials.read",
    blurb: "Non-human principals.", Body: ServiceAccountsBody },
  { slug: "api-keys", group: "Platform", title: "API Keys", requires: "credentials.read",
    blurb: "Gateway API keys, shown once.", Body: ApiCredentialsBody },
  { slug: "gateway-credentials", group: "Platform", title: "Gateway Credentials", requires: "credentials.read",
    blurb: "Scoped gateway keys with rotate and revoke.", Body: GatewayCredentialsBody },
  { slug: "model-registry", group: "Platform", title: "Model Registry", requires: "providers.read",
    blurb: "Model catalog and sync.", Body: ModelsBody },

  { slug: "members", group: "Organization", title: "Members", requires: "members.read",
    blurb: "Members, roles, and removal.", Body: MembersBody },
  { slug: "invitations", group: "Organization", title: "Invitations", requires: "members.read",
    blurb: "Pending and accepted invitations.", Body: InvitationsBody },
  { slug: "org-teams", group: "Organization", title: "Teams", requires: "teams.read",
    blurb: "Create teams and manage membership.", Body: TeamsBody },
  { slug: "org-projects", group: "Organization", title: "Projects", requires: "projects.read",
    blurb: "Create projects and manage membership.", Body: ProjectsBody },

  { slug: "usage-spend", group: "Billing", title: "Usage & Spend", requires: "billing.read",
    blurb: "Platform plan consumption.", Body: UsageAndSpendBody },
  { slug: "subscription", group: "Billing", title: "Subscription", requires: "billing.read",
    blurb: "Plan, status, and period.", Body: SubscriptionBody },
  { slug: "invoices", group: "Billing", title: "Invoices", requires: "billing.read",
    blurb: "Issued invoices.", Body: InvoicesBody },

  { slug: "quick-start", group: "Developer", title: "Quick Start", requires: "credentials.read",
    blurb: "First request through the gateway.", Body: QuickStartBody },
  { slug: "sdks", group: "Developer", title: "SDKs", requires: "credentials.read",
    blurb: "Shared TypeScript packages.", Body: SdksBody },
  { slug: "cli", group: "Developer", title: "CLI", requires: "credentials.read",
    blurb: "Command line tooling.", Body: CliBody },
  { slug: "api-docs", group: "Developer", title: "API Docs", requires: "credentials.read",
    blurb: "OpenAPI and route reference.", Body: ApiDocsBody },
];

export const GROUPS: readonly Group[] = [
  "Observe",
  "Govern",
  "Security",
  "Platform",
  "Organization",
  "Billing",
  "Developer",
];

export function sectionBySlug(slug: string): SectionDef | undefined {
  return SECTIONS.find((section) => section.slug === slug);
}
