import type { RequestOptions, Transport } from "./client";

export interface ActivityEvent {
  request_id: string;
  organization_id: string;
  user_id?: string;
  user_name?: string;
  service_account?: string;
  project_id?: string;
  project_name?: string;
  team_id?: string;
  provider: string;
  model: string;
  status: string;
  started_at: string;
  current_state: string;
}

/** Activity responses wrap the payload in `data`, not `items`. */
export interface ActivityData {
  data: ActivityEvent[];
}

export interface ActivityCount {
  count: number;
}

export interface ActivityModule {
  events(options?: RequestOptions): Promise<ActivityData>;
  activeUsers(options?: RequestOptions): Promise<ActivityCount>;
  activeRequests(options?: RequestOptions): Promise<ActivityCount>;
  /**
   * Server-sent events. Returns the raw `Response` so the caller can consume
   * `body` as a stream; the server opens with an `event: ping` frame.
   * Errors are not parsed here, so check `response.ok` before reading.
   */
  stream(options?: RequestOptions): Promise<Response>;
}

export function createActivityModule(transport: Transport): ActivityModule {
  return {
    events: (options) => transport.request<ActivityData>("/v1/activity", options),
    activeUsers: (options) => transport.request<ActivityCount>("/v1/activity/active-users", options),
    activeRequests: (options) =>
      transport.request<ActivityCount>("/v1/activity/active-requests", options),
    stream: (options) => transport.fetchRaw("/v1/activity/stream", options),
  };
}