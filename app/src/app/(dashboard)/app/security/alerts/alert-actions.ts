"use server";

import { getApiClient } from "@/lib/session";
import { revalidatePath } from "next/cache";
import type { AlertSeverity, AlertStatus, AlertType } from "@ai-governance/api-client";

export async function createAlert(data: { name: string; alert_type: AlertType; severity: AlertSeverity; configuration: Record<string, unknown> }) {
  const { alerts } = await getApiClient();
  await alerts.create(data);
  revalidatePath("/app/security/alerts");
}

export async function updateAlert(id: string, data: { name: string; severity: AlertSeverity; status: AlertStatus; configuration: Record<string, unknown> }) {
  const { alerts } = await getApiClient();
  await alerts.update(id, data);
  revalidatePath("/app/security/alerts");
}

export async function deleteAlert(id: string) {
  const { alerts } = await getApiClient();
  await alerts.delete(id);
  revalidatePath("/app/security/alerts");
}

export async function testAlert(id: string) {
  const { alerts } = await getApiClient();
  return await alerts.test(id);
}
