"use server";

import { revalidatePath } from "next/cache";
import { ApiError } from "@ai-governance/api-client";
import { getApiClient } from "@/lib/session";

export interface ActionState {
  error?: string;
  success?: boolean;
}

export async function createServiceAccount(
  _state: ActionState,
  formData: FormData
): Promise<ActionState> {
  const name = String(formData.get("name") ?? "").trim();
  const description = String(formData.get("description") ?? "").trim();

  if (!name) return { error: "Name is required." };

  try {
    const { serviceAccounts } = await getApiClient();
    await serviceAccounts.create({
      name,
      description: description || "",
    });
    
    revalidatePath("/app/platform/service-accounts");
    return { success: true };
  } catch (error) {
    return {
      error: error instanceof ApiError ? error.message : "Failed to create service account.",
    };
  }
}

export async function removeServiceAccount(
  _state: ActionState,
  formData: FormData
): Promise<ActionState> {
  const id = String(formData.get("id") ?? "");
  if (!id) return { error: "Missing service account ID." };

  try {
    const { serviceAccounts } = await getApiClient();
    await serviceAccounts.remove(id);
    revalidatePath("/app/platform/service-accounts");
    return { success: true };
  } catch (error) {
    return {
      error: error instanceof ApiError ? error.message : "Failed to remove service account.",
    };
  }
}
