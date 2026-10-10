import type { components } from "@ai-governance/contracts";
import type { Transport, RequestOptions, ListResponse } from "./client";

export type Policy = components["schemas"]["Policy"];
export type PolicyRule = components["schemas"]["PolicyRule"];
export type CreatePolicyRequest = components["schemas"]["CreatePolicyRequest"];
export type UpdatePolicyRequest = components["schemas"]["UpdatePolicyRequest"];
export type CreatePolicyRuleRequest = components["schemas"]["CreatePolicyRuleRequest"];
export type UpdatePolicyRuleRequest = components["schemas"]["UpdatePolicyRuleRequest"];

export interface PoliciesModule {
  list(options?: RequestOptions): Promise<ListResponse<Policy>>;
  get(id: string, options?: RequestOptions): Promise<Policy>;
  create(data: CreatePolicyRequest, options?: RequestOptions): Promise<Policy>;
  update(id: string, data: UpdatePolicyRequest, options?: RequestOptions): Promise<Policy>;
  delete(id: string, options?: RequestOptions): Promise<void>;

  listRules(policyId: string, options?: RequestOptions): Promise<ListResponse<PolicyRule>>;
  createRule(policyId: string, data: CreatePolicyRuleRequest, options?: RequestOptions): Promise<PolicyRule>;
  updateRule(policyId: string, ruleId: string, data: UpdatePolicyRuleRequest, options?: RequestOptions): Promise<PolicyRule>;
  deleteRule(policyId: string, ruleId: string, options?: RequestOptions): Promise<void>;
}

export function createPoliciesModule(t: Transport): PoliciesModule {
  return {
    list: (opts) =>
      t.request<ListResponse<Policy>>("/v1/policies", opts),
    get: (id, opts) =>
      t.requestAt<Policy>(`/v1/policies/${encodeURIComponent(id)}`, opts),
    create: (data, opts) =>
      t.request<Policy>("/v1/policies", { ...opts, method: "POST", body: data }),
    update: (id, data, opts) =>
      t.requestAt<Policy>(`/v1/policies/${encodeURIComponent(id)}`, { ...opts, method: "PATCH", body: data }),
    delete: (id, opts) =>
      t.requestAt<void>(`/v1/policies/${encodeURIComponent(id)}`, { ...opts, method: "DELETE" }),
    listRules: (policyId, opts) =>
      t.requestAt<ListResponse<PolicyRule>>(`/v1/policies/${encodeURIComponent(policyId)}/rules`, opts),
    createRule: (policyId, data, opts) =>
      t.requestAt<PolicyRule>(`/v1/policies/${encodeURIComponent(policyId)}/rules`, { ...opts, method: "POST", body: data }),
    updateRule: (policyId, ruleId, data, opts) =>
      t.requestAt<PolicyRule>(`/v1/policies/${encodeURIComponent(policyId)}/rules/${encodeURIComponent(ruleId)}`, { ...opts, method: "PATCH", body: data }),
    deleteRule: (policyId, ruleId, opts) =>
      t.requestAt<void>(`/v1/policies/${encodeURIComponent(policyId)}/rules/${encodeURIComponent(ruleId)}`, { ...opts, method: "DELETE" }),
  };
}
