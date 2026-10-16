"use server";

import { revalidatePath } from "next/cache";
import { getApiClient } from "@/lib/session";
import type { CreatedCredential } from "@ai-governance/api-client";

export async function createControlCredentialAction(data: {
  name: string;
  scope?: string;
}): Promise<{ success: boolean; data?: CreatedCredential; error?: string }> {
  try {
    const { credentials } = await getApiClient();
    const res = await credentials.create({
      name: data.name,
      scope: data.scope || "control_plane",
    });
    revalidatePath("/app/govern/credentials");
    return { success: true, data: res };
  } catch (err: unknown) {
    console.error("createControlCredentialAction error:", err);
    return {
      success: false,
      error: err instanceof Error ? err.message : "Failed to issue credential",
    };
  }
}

export async function revokeControlCredentialAction(
  id: string
): Promise<{ success: boolean; error?: string }> {
  try {
    const { credentials } = await getApiClient();
    await credentials.remove(id);
    revalidatePath("/app/govern/credentials");
    return { success: true };
  } catch (err: unknown) {
    console.error("revokeControlCredentialAction error:", err);
    return {
      success: false,
      error: err instanceof Error ? err.message : "Failed to revoke credential",
    };
  }
}
