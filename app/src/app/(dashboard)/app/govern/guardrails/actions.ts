"use server";

import { revalidatePath } from "next/cache";
import { getApiClient } from "@/lib/session";
import type {
  CreateGuardrailInput,
  Guardrail,
  GuardrailEvent,
  TestGuardrailResponse,
  UpdateGuardrailInput,
} from "@ai-governance/api-client";

export async function fetchGuardrailsAction(): Promise<{ items: Guardrail[] }> {
  try {
    const { guardrails } = await getApiClient();
    const res = await guardrails.list();
    return { items: res.items ?? [] };
  } catch (err) {
    console.error("fetchGuardrailsAction error:", err);
    return { items: [] };
  }
}

export async function fetchGuardrailEventsAction(): Promise<{ items: GuardrailEvent[] }> {
  try {
    const { guardrails } = await getApiClient();
    const res = await guardrails.listEvents();
    return { items: res.items ?? [] };
  } catch (err) {
    console.error("fetchGuardrailEventsAction error:", err);
    return { items: [] };
  }
}

export async function createGuardrailAction(input: CreateGuardrailInput): Promise<{ success: boolean; data?: Guardrail; error?: string }> {
  try {
    const { guardrails } = await getApiClient();
    const data = await guardrails.create(input);
    revalidatePath("/app/govern/guardrails");
    return { success: true, data };
  } catch (err: unknown) {
    console.error("createGuardrailAction error:", err);
    const msg = err instanceof Error ? err.message : "Failed to create guardrail";
    return { success: false, error: msg };
  }
}

export async function updateGuardrailAction(id: string, input: UpdateGuardrailInput): Promise<{ success: boolean; data?: Guardrail; error?: string }> {
  try {
    const { guardrails } = await getApiClient();
    const data = await guardrails.update(id, input);
    revalidatePath("/app/govern/guardrails");
    return { success: true, data };
  } catch (err: unknown) {
    console.error("updateGuardrailAction error:", err);
    const msg = err instanceof Error ? err.message : "Failed to update guardrail";
    return { success: false, error: msg };
  }
}

export async function deleteGuardrailAction(id: string): Promise<{ success: boolean; error?: string }> {
  try {
    const { guardrails } = await getApiClient();
    await guardrails.delete(id);
    revalidatePath("/app/govern/guardrails");
    return { success: true };
  } catch (err: unknown) {
    console.error("deleteGuardrailAction error:", err);
    const msg = err instanceof Error ? err.message : "Failed to delete guardrail";
    return { success: false, error: msg };
  }
}

export async function testGuardrailAction(id: string, payload: string, tokenCount: number = 0): Promise<{ success: boolean; data?: TestGuardrailResponse; error?: string }> {
  try {
    const { guardrails } = await getApiClient();
    const data = await guardrails.test(id, { payload, token_count: tokenCount });
    return { success: true, data };
  } catch (err: unknown) {
    console.error("testGuardrailAction error:", err);
    const msg = err instanceof Error ? err.message : "Failed to test guardrail";
    return { success: false, error: msg };
  }
}
