import type { components } from "@ai-governance/contracts";
import type { ListResponse, RequestOptions, Transport } from "./client";
import type { UsageQuery, UsageSummary } from "./usage";

export type InvoiceStatus = components["schemas"]["Invoice"]["status"];
export type Plan = components["schemas"]["Plan"];
export type Subscription = components["schemas"]["Subscription"];
export type InvoiceLine = components["schemas"]["InvoiceLine"];
export type Invoice = components["schemas"]["Invoice"];
export type CheckoutSessionParams = components["schemas"]["BillingCheckoutRequest"];
export type CheckoutSessionResponse = components["schemas"]["BillingCheckoutSession"];
export type VerifyPaymentParams = components["schemas"]["BillingVerifyPaymentRequest"];

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
