import type { ListResponse, RequestOptions, Transport } from "./client";
import type { UsageQuery, UsageSummary } from "./usage";

/**
 * Billing payloads are the snake_case DTOs declared in
 * `backend/api/openapi/openapi.yaml`, mirrored here so a consumer of this
 * package does not need `@ai-governance/contracts` at the type level.
 *
 * These endpoints report *local* billing state only. No payment processor is
 * integrated, so an invoice is never reported as paid, and nothing in these
 * types references a charge, a payment, or a downloadable PDF. Do not add such
 * fields here unless a processor is genuinely integrated.
 */

/** Matches the `status` CHECK on `plans`. */
export type PlanStatus = "active" | "inactive";

/** Mirrors the stored `subscriptions.status` CHECK. */
export type SubscriptionStatus =
  | "active"
  | "trialing"
  | "canceled"
  | "past_due"
  | "incomplete";

/**
 * The only invoice statuses this service can store. There is deliberately no
 * `paid`: with no payment processor integrated, no invoice here implies that
 * money moved.
 */
export type InvoiceStatus = "open" | "void";

export interface Plan {
  id: string;
  code: string;
  name: string;
  /** Empty string when unset; no price is stored or derived. */
  description: string;
  status: PlanStatus;
  /** RFC3339. */
  created_at: string;
  /** RFC3339. */
  updated_at: string;
}

export interface Subscription {
  plan_code: string;
  plan_name: string;
  status: SubscriptionStatus;
  /** RFC3339, or null when the stored column is NULL. */
  current_period_start: string | null;
  /** RFC3339, or null when the stored column is NULL. */
  current_period_end: string | null;
  cancel_at_period_end: boolean;
}

export interface InvoiceLine {
  id: string;
  description: string;
  quantity: number;
  /** Per-unit price in millionths of the currency's major unit. */
  unit_micros: number;
  /** Line total in millionths of the currency's major unit. */
  amount_micros: number;
}

export interface Invoice {
  id: string;
  /** Null when the invoice is not tied to a subscription row. */
  subscription_id: string | null;
  /** Human-readable invoice reference. */
  number: string;
  status: InvoiceStatus;
  currency: string;
  /** Invoice total in millionths of the currency's major unit. */
  total_micros: number;
  /** RFC3339. */
  period_start: string;
  /** RFC3339. */
  period_end: string;
  /** RFC3339. */
  issued_at: string;
  lines: InvoiceLine[];
}

export interface BillingModule {
  /** The global plan catalog, identical for every organization. */
  listPlans(options?: RequestOptions): Promise<ListResponse<Plan>>;
  /**
   * The caller's organization plan and billing cycle.
   *
   * Rejects with a not-found error when the organization has no subscription,
   * rather than resolving to an empty object.
   */
  getSubscription(options?: RequestOptions): Promise<Subscription>;
  /**
   * Recorded request and cost totals for a window. Shares the body shape of
   * `usage.summary`; every value is measured from recorded requests.
   */
  getUsage(query?: UsageQuery, options?: RequestOptions): Promise<UsageSummary>;
  /** Invoices newest issued first, each with its itemisation lines. */
  listInvoices(options?: RequestOptions): Promise<ListResponse<Invoice>>;
}

export function createBillingModule(transport: Transport): BillingModule {
  return {
    listPlans: (options) => transport.request<ListResponse<Plan>>("/v1/billing/plans", options),
    getSubscription: (options) => transport.request<Subscription>("/v1/billing/subscription", options),
    getUsage: (query, options) =>
      transport.request<UsageSummary>("/v1/billing/usage", {
        ...options,
        query: { from: query?.from, to: query?.to },
      }),
    listInvoices: (options) => transport.request<ListResponse<Invoice>>("/v1/billing/invoices", options),
  };
}