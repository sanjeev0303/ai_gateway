import type { components } from "@ai-governance/contracts";
import type { Transport, RequestOptions, ListResponse } from "./client";

export type ModelAccessEntry = components["schemas"]["ModelAccessEntry"];
export type ModelAccessList = components["schemas"]["ModelAccessList"];
export type UpdateModelAccessRequest = components["schemas"]["UpdateModelAccessRequest"];
export type ModelAccessPreviewRequest = components["schemas"]["ModelAccessPreviewRequest"];
export type ModelAccessPreviewResponse = components["schemas"]["ModelAccessPreviewResponse"];

export type Budget = components["schemas"]["Budget"];
export type BudgetList = components["schemas"]["BudgetList"];
export type CreateBudgetRequest = components["schemas"]["CreateBudgetRequest"];
export type UpdateBudgetRequest = components["schemas"]["UpdateBudgetRequest"];

export interface ModelAccessModule {
  list(options?: RequestOptions): Promise<ListResponse<ModelAccessEntry>>;
  update(data: UpdateModelAccessRequest, options?: RequestOptions): Promise<{ status: string }>;
  preview(data: ModelAccessPreviewRequest, options?: RequestOptions): Promise<ModelAccessPreviewResponse>;
}

export interface BudgetsModule {
  list(options?: RequestOptions & { scope?: string; scope_id?: string; limit_type?: string; enabled?: boolean }): Promise<ListResponse<Budget>>;
  get(id: string, options?: RequestOptions): Promise<Budget>;
  create(data: CreateBudgetRequest, options?: RequestOptions): Promise<Budget>;
  update(id: string, data: UpdateBudgetRequest, options?: RequestOptions): Promise<Budget>;
  delete(id: string, options?: RequestOptions): Promise<void>;
}

export function createModelAccessModule(t: Transport): ModelAccessModule {
  return {
    list: (opts) =>
      t.request<ListResponse<ModelAccessEntry>>("/v1/model-access", opts),
    update: (data, opts) =>
      t.request<{ status: string }>("/v1/model-access", { ...opts, method: "PUT", body: data }),
    preview: (data, opts) =>
      t.requestAt<ModelAccessPreviewResponse>("/v1/model-access/preview", { ...opts, method: "POST", body: data }),
  };
}

export function createBudgetsModule(t: Transport): BudgetsModule {
  return {
    list: (opts) =>
      t.request<ListResponse<Budget>>("/v1/budgets", {
        ...opts,
        query: {
          scope: opts?.scope,
          scope_id: opts?.scope_id,
          limit_type: opts?.limit_type,
          enabled: opts?.enabled !== undefined ? String(opts.enabled) : undefined,
        },
      }),
    get: (id, opts) =>
      t.requestAt<Budget>(`/v1/budgets/${encodeURIComponent(id)}`, opts),
    create: (data, opts) =>
      t.request<Budget>("/v1/budgets", { ...opts, method: "POST", body: data }),
    update: (id, data, opts) =>
      t.requestAt<Budget>(`/v1/budgets/${encodeURIComponent(id)}`, { ...opts, method: "PATCH", body: data }),
    delete: (id, opts) =>
      t.requestAt<void>(`/v1/budgets/${encodeURIComponent(id)}`, { ...opts, method: "DELETE" }),
  };
}