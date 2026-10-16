import type { ListResponse, RequestOptions, Transport } from "./client";

export type GuardrailTargetType = "global" | "team" | "project" | "gateway_credential";
export type GuardrailDetector = "pii" | "secrets" | "unsafe_keywords" | "request_constraints";
export type GuardrailAction = "allow" | "warn" | "block" | "redact";

export interface Guardrail {
  id: string;
  name: string;
  description: string;
  target_type: GuardrailTargetType;
  target_id?: string | null;
  detector: GuardrailDetector;
  action: GuardrailAction;
  configuration: Record<string, unknown>;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateGuardrailInput {
  name: string;
  description?: string;
  target_type: GuardrailTargetType;
  target_id?: string | null;
  detector: GuardrailDetector;
  action: GuardrailAction;
  configuration: Record<string, unknown>;
  enabled?: boolean;
}

export interface UpdateGuardrailInput {
  name: string;
  description?: string;
  action: GuardrailAction;
  configuration: Record<string, unknown>;
  enabled: boolean;
}

export interface GuardrailEvent {
  id: string;
  guardrail_id: string;
  request_id: string;
  direction: "input" | "output";
  detector: string;
  action: string;
  matched_category: string;
  timestamp: string;
}

export interface TestGuardrailInput {
  payload: string;
  token_count?: number;
}

export interface TestGuardrailResponse {
  triggered: boolean;
  action?: string;
  matched_category?: string;
}

export interface GuardrailsModule {
  list(options?: RequestOptions): Promise<ListResponse<Guardrail>>;
  create(input: CreateGuardrailInput, options?: RequestOptions): Promise<Guardrail>;
  get(id: string, options?: RequestOptions): Promise<Guardrail>;
  update(id: string, input: UpdateGuardrailInput, options?: RequestOptions): Promise<Guardrail>;
  delete(id: string, options?: RequestOptions): Promise<void>;
  listEvents(options?: RequestOptions): Promise<ListResponse<GuardrailEvent>>;
  test(id: string, input: TestGuardrailInput, options?: RequestOptions): Promise<TestGuardrailResponse>;
}

export function createGuardrailsModule(transport: Transport): GuardrailsModule {
  return {
    list(options) {
      return transport.request<ListResponse<Guardrail>>("/v1/guardrails", {
        method: "GET",
        ...options,
      });
    },
    create(input, options) {
      return transport.request<Guardrail>("/v1/guardrails", {
        method: "POST",
        body: input,
        ...options,
      });
    },
    get(id, options) {
      return transport.request<Guardrail>(`/v1/guardrails/${encodeURIComponent(id)}`, {
        method: "GET",
        ...options,
      });
    },
    update(id, input, options) {
      return transport.request<Guardrail>(`/v1/guardrails/${encodeURIComponent(id)}`, {
        method: "PATCH",
        body: input,
        ...options,
      });
    },
    delete(id, options) {
      return transport.request<void>(`/v1/guardrails/${encodeURIComponent(id)}`, {
        method: "DELETE",
        ...options,
      });
    },
    listEvents(options) {
      return transport.request<ListResponse<GuardrailEvent>>("/v1/guardrails/events", {
        method: "GET",
        ...options,
      });
    },
    test(id, input, options) {
      return transport.request<TestGuardrailResponse>(`/v1/guardrails/${encodeURIComponent(id)}/test`, {
        method: "POST",
        body: input,
        ...options,
      });
    },
  };
}
