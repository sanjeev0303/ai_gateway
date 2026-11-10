import type { components } from "@ai-governance/contracts";
import type { ListResponse, RequestOptions, Transport } from "./client";

export type Model = components["schemas"]["CatalogModel"];
/** Both fields are optional: provider_kind alone syncs that kind's active account, an empty body syncs every active account. */
export type SyncModelsInput = components["schemas"]["SyncCatalogModelsRequest"];

export interface ModelsModule {
  list(options?: RequestOptions): Promise<ListResponse<Model>>;
  /** Live sync against the provider; an API key, if given, is probed, never stored. */
  sync(input: SyncModelsInput, options?: RequestOptions): Promise<ListResponse<Model>>;
}

export function createModelsModule(transport: Transport): ModelsModule {
  return {
    list: (options) => transport.request<ListResponse<Model>>("/v1/models", options),
    sync: (input, options) =>
      transport.request<ListResponse<Model>>("/v1/models/sync", {
        ...options,
        method: "POST",
        body: input,
      }),
  };
}