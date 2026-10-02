/**
 * Demo domain types.
 *
 * The one rule this layer enforces: **entities are pure dimensions**. A person,
 * team, project, model or provider carries identity and descriptive attributes
 * only — never a spend or request count. Every number the UI shows is derived
 * by aggregating `traffic`, so two pages reading the same number can never
 * disagree. See `./traffic.ts` for the generator and `./derive.ts` for the
 * selectors.
 */

import type { ID, ISODateString, StatusTone } from "@/types/common";

export type { ID, ISODateString, StatusTone };

/* ------------------------------------------------------------------ *
 * Date range
 * ------------------------------------------------------------------ */

export type DateRange = "24h" | "7d" | "30d" | "90d";

export const DATE_RANGES: readonly DateRange[] = ["24h", "7d", "30d", "90d"];

export const DATE_RANGE_LABEL: Record<DateRange, string> = {
  "24h": "Last 24 hours",
  "7d": "Last 7 days",
  "30d": "Last 30 days",
  "90d": "Last 90 days",
};

export const DATE_RANGE_SHORT: Record<DateRange, string> = {
  "24h": "24h",
  "7d": "7d",
  "30d": "30d",
  "90d": "90d",
};

/** Number of whole days each range covers. `24h` is a partial day. */
export const DATE_RANGE_DAYS: Record<DateRange, number> = {
  "24h": 1,
  "7d": 7,
  "30d": 30,
  "90d": 90,
};

/* ------------------------------------------------------------------ *
 * Organization
 * ------------------------------------------------------------------ */

export type PlanTier = "starter" | "growth" | "scale" | "enterprise";

export type OrganizationRole = "owner" | "admin" | "member" | "viewer";

export interface DemoOrganization {
  id: ID;
  name: string;
  slug: string;
  plan: PlanTier;
  status: "active" | "trialing" | "past_due";
  createdAt: ISODateString;
  seatsUsed: number;
  seatsTotal: number;
  billingEmail: string;
  region: string;
  /** Recurring platform fee in whole USD, before metered usage. */
  platformFeeMonthly: number;
  dataResidency: string;
  ssoEnforced: boolean;
}

export const PLAN_LABEL: Record<PlanTier, string> = {
  starter: "Starter",
  growth: "Growth",
  scale: "Scale",
  enterprise: "Enterprise",
};

/* ------------------------------------------------------------------ *
 * People, teams, projects
 * ------------------------------------------------------------------ */

export type PersonStatus = "active" | "invited" | "suspended";
export type PersonRole = OrganizationRole | "developer" | "designer" | "analyst";

export interface DemoPerson {
  id: ID;
  name: string;
  email: string;
  title: string;
  role: PersonRole;
  status: PersonStatus;
  teamId: ID;
  /** Ids of the projects this person can bill usage to. */
  projectIds: ID[];
  createdAt: ISODateString;
  lastActiveAt: ISODateString;
  /** Hue (0-359) used to derive the deterministic avatar colour. */
  avatarHue: number;
  hasPersonalToken: boolean;
  mfaEnrolled: boolean;
}

export interface DemoTeam {
  id: ID;
  name: string;
  slug: string;
  description: string;
  leadId: ID;
  memberIds: ID[];
  projectIds: ID[];
  createdAt: ISODateString;
  /** Whole USD per month, or null when the team is unbudgeted. */
  monthlyBudget: number | null;
}

export type ProjectEnvironment = "production" | "staging" | "development";
export type ProjectStatus = "active" | "paused" | "archived";

export interface DemoProject {
  id: ID;
  name: string;
  slug: string;
  description: string;
  environment: ProjectEnvironment;
  teamId: ID;
  ownerId: ID;
  status: ProjectStatus;
  createdAt: ISODateString;
  allowedModelIds: ID[];
  monthlyBudget: number | null;
  /** Git-style repository the workload runs from. */
  repository: string;
}

export const ENVIRONMENT_TONE: Record<ProjectEnvironment, StatusTone> = {
  production: "success",
  staging: "info",
  development: "neutral",
};

/* ------------------------------------------------------------------ *
 * Providers and models
 * ------------------------------------------------------------------ */

export type ProviderStatus = "operational" | "degraded" | "outage" | "maintenance";

export interface DemoProvider {
  id: ID;
  /** Machine slug, e.g. "anthropic". */
  name: string;
  displayName: string;
  status: ProviderStatus;
  createdAt: ISODateString;
  /** Whole USD per month, or null when billed purely on usage. */
  platformFeeMonthly: number | null;
  region: string;
  docsUrl: string;
  /** Models are seeded per provider, not stored on the provider. */
  modelIds: ID[];
}

export const PROVIDER_TONE: Record<ProviderStatus, StatusTone> = {
  operational: "success",
  degraded: "warning",
  outage: "danger",
  maintenance: "info",
};

export type ModelModality = "text" | "vision" | "audio" | "embedding" | "multimodal";
export type ModelStatus = "ga" | "preview" | "deprecated";

export interface DemoModel {
  id: ID;
  /** Machine name, e.g. "claude-sonnet-4-5". */
  name: string;
  displayName: string;
  providerId: ID;
  modality: ModelModality;
  status: ModelStatus;
  contextWindowTokens: number;
  maxOutputTokens: number;
  /** Whole USD per one million tokens. */
  inputPerMTok: number;
  outputPerMTok: number;
  description: string;
  releasedAt: ISODateString;
  /** Typical time to first token in milliseconds, used to seed latency. */
  ttftMs: number;
  retiredOn?: ISODateString;
  /** Per-call USD for models billed per generation; overrides token pricing. */
  flatCostPerCall?: number;
}

export const MODEL_STATUS_TONE: Record<ModelStatus, StatusTone> = {
  ga: "success",
  preview: "info",
  deprecated: "warning",
};

/* ------------------------------------------------------------------ *
 * Traffic — the single source of truth for every number
 * ------------------------------------------------------------------ */

export type RequestKind =
  | "chat"
  | "completion"
  | "embedding"
  | "tool_call"
  | "vision"
  | "batch";

export type RequestStatus = "success" | "error" | "blocked";

export const REQUEST_KIND_LABEL: Record<RequestKind, string> = {
  chat: "Chat",
  completion: "Completion",
  embedding: "Embedding",
  tool_call: "Tool call",
  vision: "Vision",
  batch: "Batch",
};

export const REQUEST_STATUS_TONE: Record<RequestStatus, StatusTone> = {
  success: "success",
  error: "danger",
  blocked: "warning",
};

/** One recorded request. Only the most recent days are materialised. */
export interface DemoRequest {
  id: ID;
  occurredAt: ISODateString;
  /** UTC day bucket, "YYYY-MM-DD". */
  day: string;
  hour: number;
  personId: ID;
  teamId: ID;
  projectId: ID;
  modelId: ID;
  providerId: ID;
  environment: ProjectEnvironment;
  kind: RequestKind;
  status: RequestStatus;
  inputTokens: number;
  outputTokens: number;
  /** Full-precision USD. Formatting happens at the edge, never here. */
  cost: number;
  latencyMs: number;
  ttftMs: number;
  summary: string;
  /** Set when a policy or guardrail stopped the request. */
  blockedByPolicyId?: ID;
  blockedByGuardrailId?: ID;
}

/** Per-dimension slice of a day. Sums into its parent traffic cell. */
export interface TrafficSlice {
  requests: number;
  inputTokens: number;
  outputTokens: number;
  cost: number;
  errors: number;
  blocked: number;
}

/** One UTC hour inside a day cell. */
export interface HourBucket {
  requests: number;
  inputTokens: number;
  outputTokens: number;
  cost: number;
}

/**
 * A whole UTC day of traffic, pre-aggregated on every dimension the product
 * slices by. The 90-day history is stored as cells; the most recent days are
 * additionally expanded into `DemoRequest` records so tables and activity
 * feeds have real rows. Aggregations read cells only, which is why a 30-day
 * total and a 7-day total are always exactly consistent.
 */
export interface TrafficCell {
  day: string;
  requests: number;
  inputTokens: number;
  outputTokens: number;
  cost: number;
  errors: number;
  blocked: number;
  p50LatencyMs: number;
  p95LatencyMs: number;
  /** 24 buckets, indexed by UTC hour. */
  hourly: HourBucket[];
  byPerson: Record<ID, TrafficSlice>;
  byProject: Record<ID, TrafficSlice>;
  byModel: Record<ID, TrafficSlice>;
  byProvider: Record<ID, TrafficSlice>;
  byTeam: Record<ID, TrafficSlice>;
  byKind: Record<RequestKind, TrafficSlice>;
  byEnvironment: Record<ProjectEnvironment, TrafficSlice>;
  byStatus: Record<RequestStatus, TrafficSlice>;
  /** Exact per-combination slices, so multi-dimension filters stay exact. */
  byCombo: Record<string, TrafficSlice>;
  /** True when individual request records exist for this day. */
  detailed: boolean;
}

/* ------------------------------------------------------------------ *
 * Aggregates — everything the UI reads
 * ------------------------------------------------------------------ */

export interface Totals {
  requests: number;
  inputTokens: number;
  outputTokens: number;
  totalTokens: number;
  cost: number;
  errors: number;
  blocked: number;
  p50LatencyMs: number;
  p95LatencyMs: number;
  errorRate: number;
  successRate: number;
}

export interface SeriesPoint {
  /** Day bucket for daily series, hour label for the 24h series. */
  label: string;
  requests: number;
  inputTokens: number;
  outputTokens: number;
  cost: number;
  errors: number;
}

export type Granularity = "hour" | "day";

export interface TrafficSeries {
  granularity: Granularity;
  range: DateRange;
  points: SeriesPoint[];
  totals: Totals;
}

export interface DimensionRollup {
  id: ID;
  name: string;
  /** Secondary label: a person's title, a model's provider, a project's env. */
  detail: string;
  requests: number;
  inputTokens: number;
  outputTokens: number;
  totalTokens: number;
  cost: number;
  errors: number;
  blocked: number;
  errorRate: number;
  p95LatencyMs: number;
  /** Fraction of the range's cost, 0-1. */
  share: number;
  /** Requests per day across the range, for pace comparisons. */
  perDay: number;
}

export interface Breakdown {
  byPerson: DimensionRollup[];
  byTeam: DimensionRollup[];
  byProject: DimensionRollup[];
  byModel: DimensionRollup[];
  byProvider: DimensionRollup[];
  byKind: DimensionRollup[];
  byEnvironment: DimensionRollup[];
}

/** The filter state every list and chart page shares. */
export interface TrafficFilters {
  personIds: ID[];
  teamIds: ID[];
  projectIds: ID[];
  modelIds: ID[];
  providerIds: ID[];
  kinds: RequestKind[];
  environments: ProjectEnvironment[];
  statuses: RequestStatus[];
  search: string;
}

export const EMPTY_FILTERS: TrafficFilters = {
  personIds: [],
  teamIds: [],
  projectIds: [],
  modelIds: [],
  providerIds: [],
  kinds: [],
  environments: [],
  statuses: [],
  search: "",
};

export function filtersAreEmpty(filters: TrafficFilters): boolean {
  return (
    filters.personIds.length === 0 &&
    filters.teamIds.length === 0 &&
    filters.projectIds.length === 0 &&
    filters.modelIds.length === 0 &&
    filters.providerIds.length === 0 &&
    filters.kinds.length === 0 &&
    filters.environments.length === 0 &&
    filters.statuses.length === 0 &&
    filters.search.trim() === ""
  );
}

export function countActiveFilters(filters: TrafficFilters): number {
  return (
    filters.personIds.length +
    filters.teamIds.length +
    filters.projectIds.length +
    filters.modelIds.length +
    filters.providerIds.length +
    filters.kinds.length +
    filters.environments.length +
    filters.statuses.length +
    (filters.search.trim() ? 1 : 0)
  );
}

/* ------------------------------------------------------------------ *
 * Conversations
 * ------------------------------------------------------------------ */

export type ConversationTurnRole = "user" | "assistant" | "tool" | "system";

export interface ConversationTurn {
  id: ID;
  role: ConversationTurnRole;
  content: string;
  createdAt: ISODateString;
  tokens: number;
  toolName?: string;
}

export interface DemoConversation {
  id: ID;
  title: string;
  personId: ID;
  projectId: ID;
  modelId: ID;
  startedAt: ISODateString;
  endedAt?: ISODateString;
  turnCount: number;
  inputTokens: number;
  outputTokens: number;
  cost: number;
  flagged: boolean;
  flagReason?: string;
  turns: ConversationTurn[];
  /** Ids of the request records this conversation was built from. */
  requestIds: ID[];
}

/* ------------------------------------------------------------------ *
 * Governance
 * ------------------------------------------------------------------ */

export type PolicyCategory =
  | "pii_redaction"
  | "prompt_injection"
  | "jailbreak"
  | "toxicity"
  | "data_residency"
  | "cost_cap"
  | "model_allowlist"
  | "retention";

export type PolicyEnforcement = "enforce" | "audit" | "disabled";
export type PolicyScope = "organization" | "team" | "project";

export const POLICY_CATEGORY_LABEL: Record<PolicyCategory, string> = {
  pii_redaction: "PII redaction",
  prompt_injection: "Prompt injection",
  jailbreak: "Jailbreak",
  toxicity: "Toxicity",
  data_residency: "Data residency",
  cost_cap: "Cost cap",
  model_allowlist: "Model allowlist",
  retention: "Retention",
};

export const POLICY_ENFORCEMENT_TONE: Record<PolicyEnforcement, StatusTone> = {
  enforce: "danger",
  audit: "info",
  disabled: "neutral",
};

export interface DemoPolicy {
  id: ID;
  name: string;
  description: string;
  category: PolicyCategory;
  enforcement: PolicyEnforcement;
  scope: PolicyScope;
  /** Team or project ids this policy is bound to; empty means org-wide. */
  targetIds: ID[];
  enabled: boolean;
  createdAt: ISODateString;
  updatedAt: ISODateString;
  createdById: ID;
  /** Fraction of matching requests stopped, 0-1. */
  blockRate: number;
}

export type GuardrailLayer = "input" | "output";
export type GuardrailAction = "block" | "mask" | "warn" | "log";

export const GUARDRAIL_ACTION_TONE: Record<GuardrailAction, StatusTone> = {
  block: "danger",
  mask: "warning",
  warn: "info",
  log: "neutral",
};

export interface DemoGuardrail {
  id: ID;
  name: string;
  description: string;
  layer: GuardrailLayer;
  action: GuardrailAction;
  enabled: boolean;
  /** Detector sensitivity, 0-1. Higher is stricter. */
  threshold: number;
  /** Categories this detector recognises. */
  detectors: string[];
  updatedAt: ISODateString;
  projectIds: ID[];
}

export type BudgetPeriod = "daily" | "weekly" | "monthly";
export type BudgetScope = "organization" | "team" | "project";
export type BudgetAction = "alert" | "throttle" | "hard_stop";

export const BUDGET_ACTION_TONE: Record<BudgetAction, StatusTone> = {
  alert: "info",
  throttle: "warning",
  hard_stop: "danger",
};

export interface DemoBudget {
  id: ID;
  name: string;
  scope: BudgetScope;
  /** null when the scope is the organization. */
  targetId: ID | null;
  limit: number;
  period: BudgetPeriod;
  action: BudgetAction;
  enabled: boolean;
  /** Fraction of the limit at which an alert fires, 0-1. */
  alertThreshold: number;
  createdAt: ISODateString;
}

/* ------------------------------------------------------------------ *
 * Security
 * ------------------------------------------------------------------ */

export type CredentialKind = "api_key" | "service_account" | "oauth" | "webhook";
export type CredentialStatus = "healthy" | "expiring" | "expired" | "revoked";

export const CREDENTIAL_STATUS_TONE: Record<CredentialStatus, StatusTone> = {
  healthy: "success",
  expiring: "warning",
  expired: "danger",
  revoked: "neutral",
};

export interface DemoCredential {
  id: ID;
  name: string;
  kind: CredentialKind;
  providerId: ID;
  providerName: string;
  status: CredentialStatus;
  /** Last four characters only. The secret is shown once, at creation. */
  maskedValue: string;
  createdAt: ISODateString;
  expiresAt?: ISODateString;
  lastUsedAt: ISODateString;
  /** Project ids holding a copy of this credential. */
  projectIds: ID[];
  scopes: string[];
}

export type AuditAction =
  | "create"
  | "update"
  | "delete"
  | "login"
  | "logout"
  | "export"
  | "policy_change"
  | "permission_change"
  | "credential_use";

export const AUDIT_ACTION_TONE: Record<AuditAction, StatusTone> = {
  create: "success",
  update: "info",
  delete: "danger",
  login: "neutral",
  logout: "neutral",
  export: "warning",
  policy_change: "warning",
  permission_change: "warning",
  credential_use: "info",
};

export type AuditActorType = "user" | "api_key" | "system";

export interface DemoAuditEvent {
  id: ID;
  occurredAt: ISODateString;
  actorType: AuditActorType;
  actorId: ID;
  actorName: string;
  actorEmail?: string;
  ipAddress: string;
  ipCountry: string;
  userAgent: string;
  action: AuditAction;
  /** Dotted resource path, e.g. "policy/pol_3f9a". */
  resource: string;
  details: string;
  mfaVerified: boolean;
}

export type AlertSeverity = "critical" | "high" | "medium" | "low";
export type AlertState = "open" | "acknowledged" | "resolved";

export const ALERT_SEVERITY_TONE: Record<AlertSeverity, StatusTone> = {
  critical: "danger",
  high: "danger",
  medium: "warning",
  low: "info",
};

export type AlertCategory =
  | "cost_anomaly"
  | "policy_violation"
  | "provider_outage"
  | "error_rate"
  | "credential_expiry"
  | "budget_exceeded"
  | "unusual_access";

export const ALERT_CATEGORY_LABEL: Record<AlertCategory, string> = {
  cost_anomaly: "Cost anomaly",
  policy_violation: "Policy violation",
  provider_outage: "Provider outage",
  error_rate: "Error rate",
  credential_expiry: "Credential expiry",
  budget_exceeded: "Budget exceeded",
  unusual_access: "Unusual access",
};

export interface DemoAlert {
  id: ID;
  occurredAt: ISODateString;
  severity: AlertSeverity;
  state: AlertState;
  category: AlertCategory;
  title: string;
  description: string;
  /** Route into the dashboard where the cause is visible. */
  link: string;
  projectId?: ID;
  personId?: ID;
  providerId?: ID;
  policyId?: ID;
  notifiedUserIds: ID[];
  acknowledgedAt?: ISODateString;
  acknowledgedById?: ID;
  resolvedAt?: ISODateString;
}

export type SecurityEventKind =
  | "prompt_injection"
  | "pii_detected"
  | "credential_leak"
  | "unusual_volume"
  | "privilege_change"
  | "impossible_travel";

export const SECURITY_EVENT_TONE: Record<SecurityEventKind, StatusTone> = {
  prompt_injection: "danger",
  pii_detected: "warning",
  credential_leak: "danger",
  unusual_volume: "warning",
  privilege_change: "info",
  impossible_travel: "info",
};

export interface DemoSecurityEvent {
  id: ID;
  occurredAt: ISODateString;
  kind: SecurityEventKind;
  severity: AlertSeverity;
  personId: ID;
  personName: string;
  projectId: ID;
  projectName: string;
  ipAddress: string;
  ipCountry: string;
  summary: string;
  /** Number of requests the event covers. */
  affectedRequests: number;
  action: GuardrailAction;
  resolved: boolean;
}

/* ------------------------------------------------------------------ *
 * Platform
 * ------------------------------------------------------------------ */

export interface DemoProviderAccount {
  id: ID;
  providerId: ID;
  providerName: string;
  label: string;
  environment: ProjectEnvironment;
  status: "connected" | "degraded" | "disconnected";
  maskedKey: string;
  connectedAt: ISODateString;
  lastRotatedAt: ISODateString;
  monthlyFee: number;
  contractId: string;
}

export type ServiceAccountStatus = "active" | "idle" | "revoked";

export interface DemoServiceAccount {
  id: ID;
  name: string;
  /** e.g. "svc_ci@acme.example". */
  email: string;
  description: string;
  status: ServiceAccountStatus;
  createdAt: ISODateString;
  lastUsedAt: ISODateString;
  scopes: string[];
  projectIds: ID[];
  rotatedAt: ISODateString;
}

export type GatewayCredentialEnvironment = "production" | "staging" | "development";

export interface DemoGatewayCredential {
  id: ID;
  name: string;
  /** Masked as `gk_live_••••1234`; the secret is shown once at creation. */
  maskedKey: string;
  environment: GatewayCredentialEnvironment;
  scopes: string[];
  createdAt: ISODateString;
  lastUsedAt: ISODateString;
  expiresAt?: ISODateString;
  status: "active" | "revoked" | "expired";
  createdById: ID;
}

export type IncidentState = "investigating" | "identified" | "monitoring" | "resolved";

export interface DemoIncident {
  id: ID;
  providerId: ID;
  providerName: string;
  title: string;
  state: IncidentState;
  severity: AlertSeverity;
  startedAt: ISODateString;
  resolvedAt?: ISODateString;
  impact: string;
  updates: { at: ISODateString; state: IncidentState; body: string }[];
}

/** 30 points of rolling uptime, oldest first, 0-100. */
export interface ProviderUptimePoint {
  day: string;
  uptime: number;
  latencyP50Ms: number;
  errorRate: number;
}

export interface PipelineStatus {
  component: string;
  status: "operational" | "degraded" | "outage";
  freshnessSlaPercent: number;
  ingestLagP50Seconds: number;
  eventsPerSecond: number;
}

export const PIPELINE_TONE: Record<PipelineStatus["status"], StatusTone> = {
  operational: "success",
  degraded: "warning",
  outage: "danger",
};

/* ------------------------------------------------------------------ *
 * Model access
 * ------------------------------------------------------------------ */

export type GrantStatus = "active" | "pending" | "expired";

export const GRANT_TONE: Record<GrantStatus, StatusTone> = {
  active: "success",
  pending: "info",
  expired: "neutral",
};

export interface DemoModelGrant {
  id: ID;
  name: string;
  modelIds: ID[];
  /** Team or project ids granted access; empty means organization-wide. */
  principalIds: ID[];
  scope: "organization" | "team" | "project";
  status: GrantStatus;
  createdAt: ISODateString;
  expiresAt?: ISODateString;
}

export interface DemoPlanEntitlement {
  tier: string;
  modelIds: ID[];
  /** Monthly ceiling in USD; null means uncapped. */
  spendCap: number | null;
  requiresApproval: boolean;
}

/* ------------------------------------------------------------------ *
 * Billing
 * ------------------------------------------------------------------ */

export type InvoiceStatus = "paid" | "open" | "past_due" | "void";

export const INVOICE_TONE: Record<InvoiceStatus, StatusTone> = {
  paid: "success",
  open: "info",
  past_due: "warning",
  void: "neutral",
};

export interface InvoiceLine {
  id: ID;
  description: string;
  quantity: number;
  unitPrice: number;
  amount: number;
}

export interface DemoInvoice {
  id: ID;
  number: string;
  issuedAt: ISODateString;
  dueAt: ISODateString;
  status: InvoiceStatus;
  subtotal: number;
  tax: number;
  total: number;
  lines: InvoiceLine[];
}

export interface DemoPaymentMethod {
  id: ID;
  brand: "visa" | "mastercard" | "amex" | "ach";
  last4: string;
  expiresMonth: number;
  expiresYear: number;
  isDefault: boolean;
}

export interface DemoSubscription {
  plan: PlanTier;
  planName: string;
  /** Whole USD charged on the current recurring cycle. */
  monthlySubscription: number;
  /** Whole USD of metered usage already billed this cycle. */
  meteredUsage: number;
  /** Whole USD of provider spend included in the plan. */
  includedSpend: number;
  hardCap: number | null;
  overageRate: number;
  renewsAt: ISODateString;
  startedAt: ISODateString;
  seats: number;
  seatPrice: number;
  paymentMethod: DemoPaymentMethod;
  /** Order-to-change-upgrade on the plan ladder. */
  tierIndex: number;
}
