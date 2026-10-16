"use server";

import { revalidatePath } from "next/cache";
import { ApiError } from "@ai-governance/api-client";
import { getApiClient } from "@/lib/session";

export interface SyncModelsResult {
  error?: string;
  syncedCount?: number;
  providerKind?: string;
}

export async function syncModelsAction(
  providerKind: string,
  apiKey: string
): Promise<SyncModelsResult> {
  if (!providerKind || !apiKey) {
    return { error: "Provider kind and API key are required to synchronize models." };
  }

  try {
    const client = await getApiClient();
    const res = await client.models.sync({
      provider_kind: providerKind,
      api_key: apiKey,
    });
    revalidatePath("/app/platform/model-registry");
    return {
      syncedCount: res.items.length,
      providerKind,
    };
  } catch (err) {
    const message = err instanceof ApiError ? err.message : "Failed to sync provider models.";
    return { error: message };
  }
}

export async function refreshModelsAction() {
  try {
    const client = await getApiClient();
    const res = await client.models.list();
    revalidatePath("/app/platform/model-registry");
    return { items: res.items };
  } catch (err) {
    const message = err instanceof ApiError ? err.message : "Failed to refresh model registry.";
    return { error: message, items: [] };
  }
}
