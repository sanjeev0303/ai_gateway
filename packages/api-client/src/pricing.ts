import type { RequestOptions, Transport } from "./client";

export interface PricingVersion {
  id: string;
  pricing_model_id: string;
  version: number;
  input_unit_price_micros: number;
  output_unit_price_micros: number;
  cache_read_input_token_price_micros: number;
  effective_from: string;
  effective_to?: string | null;
  created_at: string;
}

export interface PricingModel {
  id: string;
  model_slug: string;
  provider_kind: "openai" | "anthropic" | "gemini" | "azure_openai" | "custom" | string;
  display_name: string;
  currency: string;
  status: "active" | "inactive";
  created_at: string;
  updated_at: string;
  latest_version?: PricingVersion;
}

export interface PricingModelDetail {
  model: PricingModel;
  versions: PricingVersion[];
}

export interface CreatePricingModelParams {
  model_slug: string;
  provider_kind: string;
  display_name?: string;
  currency: string;
}

export interface CreatePricingVersionParams {
  input_unit_price_micros: number;
  output_unit_price_micros: number;
  cache_read_input_token_price_micros?: number;
  effective_from?: string;
  effective_to?: string;
}

export interface CalculateCostParams {
  model_slug: string;
  provider_kind?: string;
  prompt_tokens: number;
  completion_tokens: number;
  cached_input_tokens?: number;
  version?: number;
  effective_at?: string;
}

export interface CalculateCostResponse {
  cost_micros: number;
  input_cost_micros?: number;
  output_cost_micros?: number;
  cached_input_cost_micros?: number;
  savings_micros?: number;
  pricing_model_formula?: string;
  currency: string;
  pricing_version: PricingVersion;
  prompt_tokens: number;
  completion_tokens: number;
  cached_input_tokens: number;
}

export interface PricingModule {
  list(options?: RequestOptions): Promise<{ data: PricingModel[] }>;
  get(idOrSlug: string, options?: RequestOptions): Promise<PricingModelDetail>;
  create(params: CreatePricingModelParams, options?: RequestOptions): Promise<PricingModel>;
  createVersion(idOrSlug: string, params: CreatePricingVersionParams, options?: RequestOptions): Promise<PricingVersion>;
  getVersion(idOrSlug: string, version: number, options?: RequestOptions): Promise<PricingVersion>;
  calculateCost(params: CalculateCostParams, options?: RequestOptions): Promise<CalculateCostResponse>;
}

export function createPricingModule(transport: Transport): PricingModule {
  return {
    list: (options) => transport.request<{ data: PricingModel[] }>("/v1/pricing", options),
    get: (idOrSlug, options) => transport.request<PricingModelDetail>(`/v1/pricing/${idOrSlug}`, options),
    create: (params, options) =>
      transport.request<PricingModel>("/v1/pricing", {
        ...options,
        method: "POST",
        body: params,
      }),
    createVersion: (idOrSlug, params, options) =>
      transport.request<PricingVersion>(`/v1/pricing/${idOrSlug}/versions`, {
        ...options,
        method: "POST",
        body: params,
      }),
    getVersion: (idOrSlug, version, options) =>
      transport.request<PricingVersion>(`/v1/pricing/${idOrSlug}/versions/${version}`, options),
    calculateCost: (params, options) =>
      transport.request<CalculateCostResponse>("/v1/pricing/calculate-cost", {
        ...options,
        method: "POST",
        body: params,
      }),
  };
}
