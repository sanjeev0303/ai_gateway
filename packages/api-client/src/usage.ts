import type { ListResponse, Query, RequestOptions, Transport } from "./client";

/**
 * Usage payloads come from Go structs with no JSON tags, so these fields are
 * PascalCase on the wire. `CostMicros`/`TotalCostMicros` are micro-dollars.
 *
 * Latency and error-rate metrics are optional because they are absent, not
 * zero, when no sample exists: a non-streaming request has no
 * time-to-first-token sample, and an empty window has no samples at all.
 * Render a missing metric as "n/a" rather than 0.
 */

/** Fields shared by every entity breakdown; the Go struct embeds these, so they arrive flattened. */
export interface UsageTotals {
  Requests: number;
  InputTokens: number;
  OutputTokens: number;
  TotalTokens: number;
  CostMicros: number;
  Errors: number;
  AvgLatencyMs?: number;
  /** RFC3339. */
  LastActivityAt: string;
}

export interface UsageSummary {
  Requests: number;
  InputTokens: number;
  OutputTokens: number;
  CachedTokens: number;
  TotalTokens: number;
  TotalCostMicros: number;
  Errors: number;
  AvgLatencyMs?: number;
  /** RFC3339. */
  LastActivityAt: string;
}

export interface UsageTimeSeriesPoint {
  BucketStart: string;
  Requests: number;
  TotalTokens: number;
  CostMicros: number;
}

export interface UsageByProject extends UsageTotals {
  ProjectID: string;
  Name: string;
}

export interface UsageByModel {
  Model: string;
  Requests: number;
  TotalTokens: number;
  CostMicros: number;
}

export interface UsageByProvider {
  Provider: string;
  Requests: number;
  TotalTokens: number;
  CostMicros: number;
}

export interface UsageByUser extends UsageTotals {
  UserID: string;
  Email: string;
  DisplayName: string;
}

/** Team totals are not additive: one request counts toward every team its user belongs to. */
export interface UsageByTeam extends UsageTotals {
  TeamID: string;
  Name: string;
}

/**
 * A percentile summary computed in Postgres. `Provider`/`Model` are absent for
 * the single overall row. `TtftSamples` and `TotalSamples` are counted
 * separately, so a family is absent only when that specific family has none.
 */
export interface UsageLatency {
  Provider?: string;
  Model?: string;
  TtftSamples: number;
  TotalSamples: number;
  AvgTTFTMs?: number;
  P50TTFTMs?: number;
  P95TTFTMs?: number;
  P99TTFTMs?: number;
  AvgTotalMs?: number;
  P50TotalMs?: number;
  P95TotalMs?: number;
  P99TotalMs?: number;
}

export interface LlmRequest {
  ID: string;
  OrganizationID: string;
  ProjectID?: string;
  UserID?: string;
  APICredentialID?: string;
  RequestID: string;
  ProviderRequestID?: string;
  ProviderKind: string;
  Model: string;
  Status: string;
  ErrorCode?: string;
  PromptTokens: number;
  CompletionTokens: number;
  CachedInputTokens: number;
  TotalTokens: number;
  LatencyMs: number;
  CostMicros: number;
  CreatedAt: string;
  RequestStartedAt?: string;
  FirstTokenAt?: string;
  CompletedAt?: string;
  TimeToFirstTokenMs?: number;
  TotalLatencyMs?: number;
}

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
      transport.request<ListResponse<UsageByModel>>("/v1/usage/by-model", { ...options, query: range(query) }),
    byProvider: (query, options) =>
      transport.request<ListResponse<UsageByProvider>>("/v1/usage/by-provider", { ...options, query: range(query) }),
    byUser: (query, options) =>
      transport.request<ListResponse<UsageByUser>>("/v1/usage/by-user", { ...options, query: filtered(query) }),
    byTeam: (query, options) =>
      transport.request<ListResponse<UsageByTeam>>("/v1/usage/by-team", { ...options, query: filtered(query) }),
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