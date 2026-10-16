"use server";

import { revalidatePath } from "next/cache";
import { ApiError } from "@ai-governance/api-client";
import { getApiClient } from "@/lib/session";

export interface ProbeResult {
  error?: string;
  data?: {
    provider_kind: string;
    status: string;
    status_reason?: string;
    last_error?: string;
    checked_at: string;
  };
}

export async function probeProviderHealthAction(
  providerKind: string,
  apiKey: string
): Promise<ProbeResult> {
  if (!providerKind || !apiKey) {
    return { error: "Provider kind and API key are required." };
  }

  try {
    const { providers } = await getApiClient();
    const result = await providers.check({
      provider_kind: providerKind,
      api_key: apiKey,
    });
    revalidatePath("/app/platform/provider-health");
    return {
      data: {
        provider_kind: result.provider_kind,
        status: result.status,
        status_reason: result.status_reason,
        last_error: result.last_error,
        checked_at: result.checked_at,
      },
    };
  } catch (err) {
    const message = err instanceof ApiError ? err.message : "Health check probe failed.";
    return { error: message };
  }
}

export async function refreshHealthAction() {
  try {
    const { providers } = await getApiClient();
    const res = await providers.health();
    revalidatePath("/app/platform/provider-health");
    return { items: res.items };
  } catch (err) {
    const message = err instanceof ApiError ? err.message : "Failed to refresh provider health.";
    return { error: message, items: [] };
  }
}
