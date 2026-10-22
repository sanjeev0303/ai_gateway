"use server";

import { revalidatePath } from "next/cache";
import { ApiError } from "@ai-governance/api-client";
import { getApiClient } from "@/lib/session";
import type {
  CreatePricingModelParams,
  CreatePricingVersionParams,
  CalculateCostParams,
} from "@ai-governance/api-client";

export interface ActionResult<T = unknown> {
  error?: string;
  data?: T;
}

export async function createPricingModelAction(
  params: CreatePricingModelParams
): Promise<ActionResult> {
  try {
    const client = await getApiClient();
    const model = await client.pricing.create(params);
    revalidatePath("/app/platform/pricing");
    return { data: model };
  } catch (err) {
    const message = err instanceof ApiError ? err.message : "Failed to create pricing model.";
    return { error: message };
  }
}

export async function createPricingVersionAction(
  idOrSlug: string,
  params: CreatePricingVersionParams
): Promise<ActionResult> {
  try {
    const client = await getApiClient();
    const version = await client.pricing.createVersion(idOrSlug, params);
    revalidatePath("/app/platform/pricing");
    return { data: version };
  } catch (err) {
    const message = err instanceof ApiError ? err.message : "Failed to create pricing version.";
    return { error: message };
  }
}

export async function calculateCostAction(
  params: CalculateCostParams
): Promise<ActionResult> {
  try {
    const client = await getApiClient();
    const result = await client.pricing.calculateCost(params);
    return { data: result };
  } catch (err) {
    const message = err instanceof ApiError ? err.message : "Failed to calculate token cost.";
    return { error: message };
  }
}
