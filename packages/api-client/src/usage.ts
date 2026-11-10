import type { components } from "@ai-governance/contracts";
import type { ListResponse, Query, RequestOptions, Transport } from "./client";

type Schemas = components["schemas"];

/**
 * Usage payloads come from Go structs with no JSON tags, so these fields are
 * PascalCase on the wire. `CostMicros`/`TotalCostMicros` are micro-dollars.
 *
 * Latency metrics are null, not zero, when no sample exists: a non-streaming
 * request has no time-to-first-token sample, and an empty window has no
 * samples at all. Render a null metric as "n/a" rather than 0.
 */
export type UsageTotals = Schemas["UsageTotals"];
export type UsageSummary = Schemas["UsageSummary"];
export type UsageTimeSeriesPoint = Schemas["UsageTimeSeriesPoint"];
export type UsageByProject = Schemas["UsageByProject"];
export type UsageByModel = Schemas["UsageByModel"];
export type UsageByProvider = Schemas["UsageByProvider"];
export type UsageByUser = Schemas["UsageByUser"];
/** Team totals are not additive: one request counts toward every team its user belongs to. */
export type UsageByTeam = Schemas["UsageByTeam"];
export type UsageByServiceAccount = Schemas["UsageByServiceAccount"];
export type UsageLatency = Schemas["UsageLatency"];
export type LlmRequest = Schemas["LlmRequest"];

export interface UsageQuery {
  /** RFC3339; a `Date` is serialised as ISO 8601. */
  from?: string | Date;
  to?: string | Date;
  project_id?: string;
  user_id?: string;
  team_id?: string;
  provider?: string;
  model?: string;
  /** One of `succeeded`, `failed`, `cancelled`. */
  status?: string;
  service_account_id?: string;
}

export interface UsageRequestQuery extends UsageQuery {
  limit?: number;
  offset?: number;
}

export interface UsageLatencyQuery extends UsageQuery {
  /** Split the percentiles per provider and model instead of one overall row. */
  group_by?: "provider_model";
}

export interface UsageModule {
  /** The only usage endpoints that return a bare object instead of a list. */
  summary(query?: UsageQuery, options?: RequestOptions): Promise<UsageSummary>;
  timeseries(query?: UsageQuery, options?: RequestOptions): Promise<ListResponse<UsageTimeSeriesPoint>>;
  byProject(query?: UsageQuery, options?: RequestOptions): Promise<ListResponse<UsageByProject>>;
  byModel(query?: UsageQuery, options?: RequestOptions): Promise<ListResponse<UsageByModel>>;
  byProvider(query?: UsageQuery, options?: RequestOptions): Promise<ListResponse<UsageByProvider>>;
  byUser(query?: UsageQuery, options?: RequestOptions): Promise<ListResponse<UsageByUser>>;
  byTeam(query?: UsageQuery, options?: RequestOptions): Promise<ListResponse<UsageByTeam>>;
  byServiceAccount(query?: UsageQuery, options?: RequestOptions): Promise<ListResponse<UsageByServiceAccount>>;
  latency(query?: UsageLatencyQuery, options?: RequestOptions): Promise<ListResponse<UsageLatency>>;
  recent(options?: RequestOptions): Promise<ListResponse<LlmRequest>>;
  requests(query?: UsageRequestQuery, options?: RequestOptions): Promise<ListResponse<LlmRequest>>;
}

function range(query: UsageQuery | undefined): Query {
  return { from: query?.from, to: query?.to };
}

function filtered(query: UsageQuery | undefined): Query {
  return {
    ...range(query),
    project_id: query?.project_id,
    user_id: query?.user_id,
    team_id: query?.team_id,
    provider: query?.provider,
    model: query?.model,
    status: query?.status,
    service_account_id: query?.service_account_id,
  };
}

export function createUsageModule(transport: Transport): UsageModule {
  return {
    summary: (query, options) =>
      transport.request<UsageSummary>("/v1/usage/summary", { ...options, query: filtered(query) }),
    timeseries: (query, options) =>
      transport.request<ListResponse<UsageTimeSeriesPoint>>("/v1/usage/timeseries", {
        ...options,
        query: { ...range(query), project_id: query?.project_id },
      }),
    byProject: (query, options) =>
      transport.request<ListResponse<UsageByProject>>("/v1/usage/by-project", {
        ...options,
        query: filtered(query),
      }),
    byModel: (query, options) =>
      transport.request<ListResponse<UsageByModel>>("/v1/usage/by-model", { ...options, query: filtered(query) }),
    byProvider: (query, options) =>
      transport.request<ListResponse<UsageByProvider>>("/v1/usage/by-provider", { ...options, query: filtered(query) }),
    byUser: (query, options) =>
      transport.request<ListResponse<UsageByUser>>("/v1/usage/by-user", { ...options, query: filtered(query) }),
    byTeam: (query, options) =>
      transport.request<ListResponse<UsageByTeam>>("/v1/usage/by-team", { ...options, query: filtered(query) }),
    byServiceAccount: (query, options) =>
      transport.request<ListResponse<UsageByServiceAccount>>("/v1/usage/by-service-account", { ...options, query: filtered(query) }),
    latency: (query, options) =>
      transport.request<ListResponse<UsageLatency>>("/v1/usage/latency", {
        ...options,
        query: { ...filtered(query), group_by: query?.group_by },
      }),
    recent: (options) => transport.request<ListResponse<LlmRequest>>("/v1/usage/recent", options),
    requests: (query, options) =>
      transport.request<ListResponse<LlmRequest>>("/v1/usage/requests", {
        ...options,
        query: { ...filtered(query), limit: query?.limit, offset: query?.offset },
      }),
  };
}