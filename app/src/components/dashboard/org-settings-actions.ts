"use server";

import { revalidatePath } from "next/cache";

import { ApiError } from "@ai-governance/api-client";

import { getApiClient } from "@/lib/session";

export interface OrganizationSettingsState {
  readonly error?: string;
  readonly saved?: boolean;
}

/**
 * Persist the organization name and slug.
 *
 * Those two fields are the only writable columns on the organization: the
 * control API registers `PATCH /v1/organization` (singular) and its request
 * schema carries nothing else, so the form deliberately edits no more than that.
 */
export async function updateOrganization(
  _state: OrganizationSettingsState,
  formData: FormData,
): Promise<OrganizationSettingsState> {
  const name = String(formData.get("name") ?? "").trim();
  const slug = String(formData.get("slug") ?? "").trim();

  if (!name) return { error: "Organization name is required." };
  if (!slug) return { error: "URL slug is required." };

  try {
    const { organization } = await getApiClient();
    await organization.update({ name, slug });
  } catch (error) {
    return {
      error:
        error instanceof ApiError
          ? error.message
          : "Could not reach the control API.",
    };
  }

  revalidatePath("/app/organization/settings");
  return { saved: true };
}