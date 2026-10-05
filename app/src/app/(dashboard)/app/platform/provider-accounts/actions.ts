"use server";

import { revalidatePath } from "next/cache";
import { ApiError } from "@ai-governance/api-client";
import { getApiClient } from "@/lib/session";

export interface ActionState {
  error?: string;
  success?: boolean;
}

export async function createProviderAccount(
  _state: ActionState,
  formData: FormData
): Promise<ActionState> {
  const providerKind = String(formData.get("provider_kind") ?? "").trim();
  const name = String(formData.get("name") ?? "").trim();
  const baseUrl = String(formData.get("base_url") ?? "").trim();

  if (!providerKind) return { error: "Provider kind is required." };
  if (!name) return { error: "Name is required." };

  try {
    const { providers } = await getApiClient();
    await providers.createAccount({
      provider_kind: providerKind,
      name,
      base_url: baseUrl || undefined,
    });
    
    revalidatePath("/app/platform/provider-accounts");
    return { success: true };
  } catch (error) {
    return {
      error: error instanceof ApiError ? error.message : "Failed to create provider account.",
    };
  }
}
