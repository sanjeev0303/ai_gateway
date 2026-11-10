"use server";

import { revalidatePath } from "next/cache";
import { ApiError, type ApiClient } from "@ai-governance/api-client";

import { getApiClient } from "@/lib/session";

export interface AdminActionResult {
  error?: string;
}

/**
 * Every platform action goes through here: call the control API, which checks
 * the caller is a platform admin and audits the action, then refresh the
 * console so its figures and statuses are current.
 */
async function run<T>(act: (client: ApiClient) => Promise<T>): Promise<{ data?: T; error?: string }> {
  try {
    const data = await act(await getApiClient());
    revalidatePath("/", "layout");
    return { data };
  } catch (err) {
    return { error: err instanceof ApiError ? err.message : "The control API could not be reached. Try again." };
  }
}

export async function suspendOrganizationAction(id: string, reason: string): Promise<AdminActionResult> {
  const { error } = await run((c) => c.admin.suspendOrganization(id, reason));
  return { error };
}

export async function reactivateOrganizationAction(id: string, reason: string): Promise<AdminActionResult> {
  const { error } = await run((c) => c.admin.reactivateOrganization(id, reason));
  return { error };
}

export async function setOrganizationPlanAction(id: string, planCode: string, reason: string): Promise<AdminActionResult> {
  const { error } = await run((c) => c.admin.setOrganizationPlan(id, planCode, reason));
  return { error };
}

export async function revokeOrganizationKeysAction(id: string, reason: string): Promise<AdminActionResult & { revoked?: number }> {
  const { data, error } = await run((c) => c.admin.revokeOrganizationKeys(id, reason));
  return { error, revoked: data?.revoked };
}

export async function disableUserAction(id: string, reason: string): Promise<AdminActionResult> {
  const { error } = await run((c) => c.admin.disableUser(id, reason));
  return { error };
}

export async function enableUserAction(id: string, reason: string): Promise<AdminActionResult> {
  const { error } = await run((c) => c.admin.enableUser(id, reason));
  return { error };
}

export async function setModelActiveAction(id: string, active: boolean, reason: string): Promise<AdminActionResult> {
  const { error } = await run((c) => c.admin.setModelActive(id, active, reason));
  return { error };
}

export async function acknowledgeAlertAction(id: string): Promise<AdminActionResult> {
  const { error } = await run((c) => c.admin.acknowledgeAlert(id));
  return { error };
}
