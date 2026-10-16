"use server";

import { revalidatePath } from "next/cache";
import { ApiError } from "@ai-governance/api-client";
import { getApiClient } from "@/lib/session";

export interface SubscriptionActionState {
  error?: string;
  success?: boolean;
}

export async function changeSubscriptionPlan(
  _prevState: SubscriptionActionState,
  formData: FormData
): Promise<SubscriptionActionState> {
  const planCode = String(formData.get("plan_code") ?? "").trim();
  if (!planCode) {
    return { error: "Please select a valid plan." };
  }

  try {
    const { billing } = await getApiClient();
    await billing.updateSubscription(planCode);
    revalidatePath("/app/billing/subscription");
    revalidatePath("/app/billing/invoices");
    return { success: true };
  } catch (err) {
    const message = err instanceof ApiError ? err.message : "Failed to change subscription plan.";
    return { error: message };
  }
}

export async function cancelSubscriptionAction(): Promise<SubscriptionActionState> {
  try {
    const { billing } = await getApiClient();
    await billing.cancelSubscription();
    revalidatePath("/app/billing/subscription");
    return { success: true };
  } catch (err) {
    const message = err instanceof ApiError ? err.message : "Failed to cancel subscription.";
    return { error: message };
  }
}
