"use server";

import { revalidatePath } from "next/cache";
import { ApiError } from "@ai-governance/api-client";
import { getApiClient } from "@/lib/session";

export interface ActionState {
  error?: string;
  success?: boolean;
  accountId?: string;
  validationStatus?: "connected" | "credential_invalid" | "provider_unreachable" | "discovery_unavailable";
}

export async function createProviderAccount(
  _state: ActionState,
  formData: FormData
): Promise<ActionState> {
  let providerKind = String(formData.get("provider_kind") ?? "").trim().toLowerCase();
  const name = String(formData.get("name") ?? "").trim();
  const baseUrl = String(formData.get("base_url") ?? "").trim();
  const apiKey = String(formData.get("api_key") ?? "").trim();

  if (providerKind === "google") {
    providerKind = "gemini";
  }

  if (!providerKind) return { error: "Provider kind is required." };
  if (!name) return { error: "Display name is required." };

  try {
    const { providers } = await getApiClient();
    const account = await providers.createAccount({
      provider_kind: providerKind,
      name,
      base_url: baseUrl || undefined,
    });

    let validationStatus: ActionState["validationStatus"] = "connected";

    if (apiKey) {
      await providers.createCredential(account.id, {
        name: `${name} Key`,
        kind: "api_key",
        secret: apiKey,
      });

      try {
        const valRes = await providers.validateAccount(account.id);
        if (!valRes.valid) {
          validationStatus = "credential_invalid";
        }
      } catch {
        validationStatus = "provider_unreachable";
      }

      try {
        await providers.syncAccountModels(account.id);
      } catch {
        if (validationStatus === "connected") {
          validationStatus = "discovery_unavailable";
        }
      }
    }
    
    revalidatePath("/app/platform/provider-accounts");
    revalidatePath("/app/platform/model-registry");
    return { success: true, accountId: account.id, validationStatus };
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

export async function testAccountConnectionAction(accountId: string): Promise<{ valid: boolean; error?: string }> {
  try {
    const { providers } = await getApiClient();
    const res = await providers.validateAccount(accountId);
    return { valid: res.valid };
  } catch (err: any) {
    return { valid: false, error: err?.message || "Connection failed" };
  }
}

export async function syncAccountModelsAction(accountId: string): Promise<{ success: boolean; total?: number; error?: string }> {
  try {
    const { providers } = await getApiClient();
    const res = await providers.syncAccountModels(accountId);
    revalidatePath("/app/platform/model-registry");
    return { success: true, total: res.total };
  } catch (err: any) {
    return { success: false, error: err?.message || "Model sync failed" };
  }
}
