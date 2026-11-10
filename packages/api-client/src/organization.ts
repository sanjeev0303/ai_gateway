import type { components } from "@ai-governance/contracts";
import type { RequestOptions, Transport } from "./client";

export type Organization = components["schemas"]["Organization"];
export type OrganizationUpdateInput = components["schemas"]["UpdateOrganizationRequest"];

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
