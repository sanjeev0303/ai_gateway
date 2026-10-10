"use server";

import { revalidatePath } from "next/cache";
import { ApiError } from "@ai-governance/api-client";
import { getApiClient } from "@/lib/session";

export interface ActionState {
  error?: string;
  success?: boolean;
}

export interface CreateCredentialState extends ActionState {
  secret?: string;
}

export async function createGatewayCredential(
  _state: CreateCredentialState,
  formData: FormData
): Promise<CreateCredentialState> {
  const name = String(formData.get("name") ?? "").trim();
  const subjectType = String(formData.get("subject_type") ?? "").trim();
  const subjectId = String(formData.get("subject_id") ?? "").trim();
  const teamId = String(formData.get("team_id") ?? "").trim();
  const projectId = String(formData.get("project_id") ?? "").trim();
  const scopesStr = String(formData.get("scopes") ?? "").trim();
  const expiryDaysStr = String(formData.get("expiry_days") ?? "").trim();

  if (!name) return { error: "Name is required." };
  if (!subjectType) return { error: "Subject type is required." };
  if (!subjectId) return { error: "Subject ID is required." };

  const scopes = scopesStr ? scopesStr.split(",").map((s) => s.trim()) : [];
  
  let expires_at: string | undefined;
  if (expiryDaysStr && !isNaN(Number(expiryDaysStr))) {
    const days = Number(expiryDaysStr);
    const date = new Date();
    date.setDate(date.getDate() + days);
    expires_at = date.toISOString();
  }

  try {
    const { credentials } = await getApiClient();
    const result = await credentials.createGateway({
      name,
      subject_type: subjectType,
      subject_id: subjectId,
      team_id: teamId || undefined,
      project_id: projectId || undefined,
      scopes,
      expires_at,
    });
    
    revalidatePath("/app/platform/gateway-credentials");
    // Return the one-time secret
    return { success: true, secret: result.key_prefix };
  } catch (error) {
    return {
      error: error instanceof ApiError ? error.message : "Failed to create credential.",
    };
  }
}

export async function revokeGatewayCredential(
  _state: ActionState,
  formData: FormData
): Promise<ActionState> {
  const id = String(formData.get("id") ?? "");
  if (!id) return { error: "Missing credential ID." };

  try {
    const { credentials } = await getApiClient();
    await credentials.removeGateway(id);
    revalidatePath("/app/platform/gateway-credentials");
    return { success: true };
  } catch (error) {
    return {
      error: error instanceof ApiError ? error.message : "Failed to revoke credential.",
    };
  }
}

export async function rotateGatewayCredential(
  _state: CreateCredentialState,
  formData: FormData
): Promise<CreateCredentialState> {
  const id = String(formData.get("id") ?? "");
  if (!id) return { error: "Missing credential ID." };

  try {
    const { credentials } = await getApiClient();
    const result = await credentials.rotateGateway(id, {});
    revalidatePath("/app/platform/gateway-credentials");
    return { success: true, secret: result.key_prefix };
  } catch (error) {
    return {
      error: error instanceof ApiError ? error.message : "Failed to rotate credential.",
    };
  }
}

export async function expireGatewayCredential(
  _state: ActionState,
  formData: FormData
): Promise<ActionState> {
  const id = String(formData.get("id") ?? "");
  if (!id) return { error: "Missing credential ID." };

  try {
    const { credentials } = await getApiClient();
    await credentials.expireGateway(id);
    revalidatePath("/app/platform/gateway-credentials");
    return { success: true };
  } catch (error) {
    return {
      error: error instanceof ApiError ? error.message : "Failed to expire credential.",
    };
  }
}

export async function restoreGatewayCredential(
  _state: ActionState,
  formData: FormData
): Promise<ActionState> {
  const id = String(formData.get("id") ?? "");
  if (!id) return { error: "Missing credential ID." };

  try {
    const { credentials } = await getApiClient();
    await credentials.restoreGateway(id);
    revalidatePath("/app/platform/gateway-credentials");
    return { success: true };
  } catch (error) {
    return {
      error: error instanceof ApiError ? error.message : "Failed to restore credential.",
    };
  }
}
