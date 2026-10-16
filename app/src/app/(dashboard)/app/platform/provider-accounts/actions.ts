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
  let providerKind = String(formData.get("provider_kind") ?? "").trim().toLowerCase();
  const name = String(formData.get("name") ?? "").trim();
  const baseUrl = String(formData.get("base_url") ?? "").trim();

  if (providerKind === "google") {
    providerKind = "gemini";
  }

  if (!providerKind) return { error: "Provider kind is required." };
  if (!name) return { error: "Display name is required." };

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
    let msg = error instanceof ApiError ? error.message : "Failed to create provider account.";
    if (msg.includes("repositories: invalid") || msg === "invalid") {
      msg = "Invalid provider account parameters. Please verify provider and name.";
    } else if (msg.includes("repositories: conflict") || msg.includes("conflict")) {
      msg = "A provider account with this display name already exists.";
    }
    return {
      error: msg,
    };
  }
}
