import type { ListResponse, RequestOptions, Transport } from "./client";

export interface ServiceAccount {
  id: string;
  organization_id: string;
  name: string;
  description: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface ServiceAccountsModule {
  create(input: { name: string; description: string }, options?: RequestOptions): Promise<ServiceAccount>;
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