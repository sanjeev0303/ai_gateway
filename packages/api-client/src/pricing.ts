import type { components } from "@ai-governance/contracts";
import type { RequestOptions, Transport } from "./client";

export type PricingVersion = components["schemas"]["PricingVersion"];
export type PricingModel = components["schemas"]["PricingModel"];
export type PricingModelDetail = components["schemas"]["PricingModelDetail"];
export type PricingModelListResponse = components["schemas"]["PricingModelListResponse"];
export type CreatePricingModelParams = components["schemas"]["CreatePricingModelRequest"];
export type CreatePricingVersionParams = components["schemas"]["CreatePricingVersionRequest"];
export type CalculateCostParams = components["schemas"]["CalculateCostRequest"];
export type CalculateCostResponse = components["schemas"]["CalculateCostResponse"];

export interface PricingModule {
  list(options?: RequestOptions): Promise<PricingModelListResponse>;
  get(idOrSlug: string, options?: RequestOptions): Promise<PricingModelDetail>;
  create(params: CreatePricingModelParams, options?: RequestOptions): Promise<PricingModel>;
  createVersion(idOrSlug: string, params: CreatePricingVersionParams, options?: RequestOptions): Promise<PricingVersion>;
  getVersion(idOrSlug: string, version: number, options?: RequestOptions): Promise<PricingVersion>;
  calculateCost(params: CalculateCostParams, options?: RequestOptions): Promise<CalculateCostResponse>;
}

export function createPricingModule(transport: Transport): PricingModule {
  return {
    list: (options) => transport.request<PricingModelListResponse>("/v1/pricing", options),
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
