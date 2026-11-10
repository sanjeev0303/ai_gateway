import type { components } from "@ai-governance/contracts";
import type { ListResponse, RequestOptions, Transport } from "./client";

export type GuardrailTargetType = components["schemas"]["Guardrail"]["target_type"];
export type GuardrailDetector = components["schemas"]["Guardrail"]["detector"];
export type GuardrailAction = components["schemas"]["Guardrail"]["action"];

export type Guardrail = components["schemas"]["Guardrail"];
export type CreateGuardrailInput = components["schemas"]["CreateGuardrailRequest"];
export type UpdateGuardrailInput = components["schemas"]["UpdateGuardrailRequest"];
export type GuardrailEvent = components["schemas"]["GuardrailEvent"];
export type TestGuardrailInput = components["schemas"]["TestGuardrailRequest"];
export type TestGuardrailResponse = components["schemas"]["TestGuardrailResponse"];

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
