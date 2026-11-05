"use server";

import { revalidatePath } from "next/cache";
import { getApiClient } from "@/lib/session";

/** Deletes a whole conversation, its text included. Owners and admins only. */
export async function deleteConversationAction(id: string): Promise<{ success: boolean; error?: string }> {
  try {
    const { conversations } = await getApiClient();
    await conversations.remove(id);
    revalidatePath("/app/observe/conversations");
    return { success: true };
  } catch (err: unknown) {
    console.error("deleteConversationAction error:", err);
    return { success: false, error: err instanceof Error ? err.message : "Failed to delete the conversation" };
  }
}
