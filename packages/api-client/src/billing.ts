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
  | "incomplete"
  | "unpaid";

/** Supported invoice statuses in Sentinel. */
export type InvoiceStatus = "open" | "void" | "paid";

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
  payment_provider?: string;
  provider_customer_id?: string | null;
  provider_subscription_id?: string | null;
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
  payment_provider?: string | null;
  payment_id?: string | null;
  paid_at?: string | null;
  lines: InvoiceLine[];
}

export interface CheckoutSessionParams {
  plan_code: string;
  payment_provider: "stripe" | "razorpay";
  customer_email?: string;
  success_url?: string;
  cancel_url?: string;
}

export interface CheckoutSessionResponse {
  session_id: string;
  payment_provider: "stripe" | "razorpay";
  checkout_url?: string;
  publishable_key?: string;
  order_id?: string;
  key_id?: string;
  amount_cents: number;
  currency: string;
  plan_code: string;
  metadata?: Record<string, unknown>;
}

export interface VerifyPaymentParams {
  payment_provider: "stripe" | "razorpay";
  order_id?: string;
  payment_id: string;
  signature: string;
  plan_code?: string;
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
  /** Update the organization's plan. */
  updateSubscription(planCode: string, options?: RequestOptions): Promise<Subscription>;
  /** Cancel the subscription at the end of the current billing period. */
  cancelSubscription(options?: RequestOptions): Promise<Subscription>;
  /** Create a payment checkout session with Stripe or Razorpay. */
  createCheckoutSession(params: CheckoutSessionParams, options?: RequestOptions): Promise<CheckoutSessionResponse>;
  /** Verify client-side checkout callback HMAC signature. */
  verifyPayment(params: VerifyPaymentParams, options?: RequestOptions): Promise<Subscription>;
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
    updateSubscription: (planCode, options) =>
      transport.request<Subscription>("/v1/billing/subscription", {
        ...options,
        method: "PUT",
        body: { plan_code: planCode },
      }),
    cancelSubscription: (options) =>
      transport.request<Subscription>("/v1/billing/subscription", {
        ...options,
        method: "DELETE",
      }),
    createCheckoutSession: (params, options) =>
      transport.request<CheckoutSessionResponse>("/v1/billing/checkout", {
        ...options,
        method: "POST",
        body: params,
      }),
    verifyPayment: (params, options) =>
      transport.request<Subscription>("/v1/billing/verify-payment", {
        ...options,
        method: "POST",
        body: params,
      }),
  };
}