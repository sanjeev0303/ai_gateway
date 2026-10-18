import type { ListResponse, RequestOptions, Transport } from "./client";
import type { ProviderProbeInput } from "./providers";

export interface Model {
  id: string;
  provider_kind: string;
  model_slug: string;
  display_name: string;
  capabilities?: string[];
  context_limit?: number;
  supports_streaming?: boolean;
  supports_tools?: boolean;
  supports_reasoning?: boolean;
  status?: string;
  pricing_version?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface ModelsModule {
  list(options?: RequestOptions): Promise<ListResponse<Model>>;
  /** Live sync against the provider; the API key is probed, never stored. */
  sync(input: ProviderProbeInput, options?: RequestOptions): Promise<ListResponse<Model>>;
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