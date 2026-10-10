import type { ID, ISODateString, StatusTone } from "./common";

export type BudgetPeriod = "daily" | "weekly" | "monthly";

export type BudgetScope = "organization" | "team" | "project" | "gateway_credential" | "service_account";

/** Action taken once a budget's limit is reached. */
export type BudgetAction = "alert" | "throttle" | "hard_stop";

export interface Budget {
  id: ID;
  name: string;
  scope: BudgetScope;
  targetId: ID;
  targetName: string;
  /** Whole USD. */
  limit: number;
  period: BudgetPeriod;
  /** Whole USD consumed in the current period. */
  spent: number;
  action: BudgetAction;
  enabled: boolean;
  /** Whole USD at which an alert fires, before the hard limit. */
  alertThreshold: number;
  createdAt: ISODateString;
}

export const BUDGET_ACTION_TONE: Record<BudgetAction, StatusTone> = {
  alert: "info",
  throttle: "warning",
  hard_stop: "danger",
};

/** Fraction of the limit consumed, clamped to 0-1 for progress bars. */
export function budgetUtilization(budget: Budget): number {
  if (budget.limit <= 0) return 0;
  return Math.min(1, budget.spent / budget.limit);
}
