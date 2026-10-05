import type { RequestOptions, Transport } from "./client";

export interface OrganizationPlan {
  code: string;
  name: string;
}

export interface OrganizationBilling {
  status: "active" | "trialing";
  /** RFC3339 timestamp, or an empty string when the stored column is NULL. */
  current_period_start: string;
  /** RFC3339 timestamp, or an empty string when the stored column is NULL. */
  current_period_end: string;
  cancel_at_period_end: boolean;
}

export interface Organization {
  id: string;
  name: string;
  slug: string;
  status: "active" | "suspended";
  created_at: string;
  updated_at: string;
  plan: OrganizationPlan | null;
  billing: OrganizationBilling | null;
}

export interface OrganizationUpdateInput {
  name?: string;
  slug?: string;
}

export interface OrganizationModule {
  get(options?: RequestOptions): Promise<Organization>;
  update(input: OrganizationUpdateInput, options?: RequestOptions): Promise<Organization>;
}

export function createOrganizationModule(transport: Transport): OrganizationModule {
  return {
    // GET /v1/organizations/me is an alias of GET /v1/organization; both are
    // registered by the control API.
    get: (options) => transport.requestAt<Organization>("/v1/organizations/me", options),
    // PATCH is registered on the singular path only, so /v1/organizations/me
    // would 404 here.
    update: (input, options) =>
      transport.requestAt<Organization>("/v1/organization", {
        ...options,
        method: "PATCH",
        body: input,
      }),
  };
}
