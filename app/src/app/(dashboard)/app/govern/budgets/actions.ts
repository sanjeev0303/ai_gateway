"use server";

import { getApiClient } from "@/lib/session";
import type { CreateBudgetRequest, UpdateBudgetRequest } from "@ai-governance/api-client";

export async function fetchBudgetsAction(params: any) {
  const { budgets } = await getApiClient();
  const response = await budgets.list(params);
  return response.items || [];
}

export async function createBudgetAction(data: CreateBudgetRequest) {
  const { budgets } = await getApiClient();
  return budgets.create(data);
}

export async function updateBudgetAction(id: string, data: UpdateBudgetRequest) {
  const { budgets } = await getApiClient();
  return budgets.update(id, data);
}

export async function deleteBudgetAction(id: string) {
  const { budgets } = await getApiClient();
  return budgets.delete(id);
}
