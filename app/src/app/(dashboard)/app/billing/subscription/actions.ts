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

export interface CheckoutActionState {
  error?: string;
  success?: boolean;
  checkoutUrl?: string;
  sessionId?: string;
  paymentProvider?: string;
  publishableKey?: string;
  orderId?: string;
  keyId?: string;
}

export async function createCheckoutSessionAction(
  planCode: string,
  paymentProvider: "stripe" | "razorpay"
): Promise<CheckoutActionState> {
  if (!planCode) {
    return { error: "Please select a valid plan." };
  }
  if (!paymentProvider) {
    return { error: "Please select a payment provider (Stripe or Razorpay)." };
  }

  try {
    const { billing } = await getApiClient();
    const session = await billing.createCheckoutSession({
      plan_code: planCode,
      payment_provider: paymentProvider,
      success_url: "/app/billing/subscription?success=true",
      cancel_url: "/app/billing/subscription?cancelled=true",
    });

    return {
      success: true,
      checkoutUrl: session.checkout_url,
      sessionId: session.session_id,
      paymentProvider: session.payment_provider,
      publishableKey: session.publishable_key,
      orderId: session.order_id,
      keyId: session.key_id,
    };
  } catch (err) {
    const message = err instanceof ApiError ? err.message : "Failed to initiate checkout session.";
    return { error: message };
  }
}

export interface VerifyPaymentActionState {
  error?: string;
  success?: boolean;
}

export async function verifyPaymentAction(params: {
  payment_provider: "stripe" | "razorpay";
  order_id?: string;
  payment_id: string;
  signature: string;
  plan_code?: string;
}): Promise<VerifyPaymentActionState> {
  try {
    const { billing } = await getApiClient();
    await billing.verifyPayment(params);
    revalidatePath("/app/billing/subscription");
    revalidatePath("/app/billing/invoices");
    return { success: true };
  } catch (err) {
    const message = err instanceof ApiError ? err.message : "Payment verification failed.";
    return { error: message };
  }
}
