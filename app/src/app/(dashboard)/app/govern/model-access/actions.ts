"use server";

import { getApiClient } from "@/lib/session";
import type { UpdateModelAccessRequest, ModelAccessPreviewRequest } from "@ai-governance/api-client";

export async function fetchModelAccessAction() {
  const { modelAccess } = await getApiClient();
  const response = await modelAccess.list();
  return response.items || [];
}

export async function updateModelAccessAction(updateData: UpdateModelAccessRequest) {
  const { modelAccess } = await getApiClient();
  await modelAccess.update(updateData);
}

export async function previewModelAccessAction(data: ModelAccessPreviewRequest) {
  const { modelAccess } = await getApiClient();
  return modelAccess.preview(data);
}
