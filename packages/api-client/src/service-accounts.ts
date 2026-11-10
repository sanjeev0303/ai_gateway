import type { components } from "@ai-governance/contracts";
import type { ListResponse, RequestOptions, Transport } from "./client";

export type ServiceAccount = components["schemas"]["ServiceAccount"];

export interface ServiceAccountsModule {
  create(input: components["schemas"]["CreateServiceAccountRequest"], options?: RequestOptions): Promise<ServiceAccount>;
  list(options?: RequestOptions): Promise<ListResponse<ServiceAccount>>;
  remove(serviceAccountId: string, options?: RequestOptions): Promise<void>;
}

export function createServiceAccountsModule(transport: Transport): ServiceAccountsModule {
  return {
    create: (input, options) =>
      transport.request<ServiceAccount>("/v1/service-accounts", { ...options, method: "POST", body: input }),
    list: (options) => transport.request<ListResponse<ServiceAccount>>("/v1/service-accounts", options),
    remove: (serviceAccountId, options) =>
      transport.requestAt<void>(`/v1/service-accounts/${encodeURIComponent(serviceAccountId)}`, {
        ...options,
        method: "DELETE",
      }),
  };
}