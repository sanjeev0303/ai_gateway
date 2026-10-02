import { NOW } from "./clock";
import { costSummary } from "./cost";
import { organization } from "./organization";
import { projects } from "./projects";
import { teams } from "./teams";
import type { Budget } from "@/types";

const daysAgo = (n: number) =>
  new Date(NOW.getTime() - n * 86_400_000).toISOString();

/**
 * Team and project budgets are derived from the fixtures so `targetName`
 * always matches the entity the id points at.
 */
const teamBudgets: readonly {
  slug: string;
  limit: number;
  action: Budget["action"];
  alertThreshold: number;
  enabled: boolean;
}[] = [
  { slug: "engineering", limit: 4_000, action: "throttle", alertThreshold: 3_200, enabled: true },
  { slug: "product", limit: 2_500, action: "throttle", alertThreshold: 2_250, enabled: true },
  { slug: "research", limit: 3_200, action: "throttle", alertThreshold: 2_880, enabled: true },
  { slug: "security", limit: 1_800, action: "hard_stop", alertThreshold: 1_620, enabled: true },
  { slug: "finance", limit: 1_200, action: "alert", alertThreshold: 1_080, enabled: true },
  { slug: "compliance", limit: 1_500, action: "alert", alertThreshold: 1_350, enabled: false },
];

const projectBudgets: readonly {
  slug: string;
  limit: number;
  action: Budget["action"];
  alertThreshold: number;
  enabled: boolean;
}[] = [
  { slug: "customer-portal", limit: 1_500, action: "throttle", alertThreshold: 1_350, enabled: true },
  { slug: "internal-tools", limit: 2_000, action: "hard_stop", alertThreshold: 1_800, enabled: true },
];

export const budgets: Budget[] = [
  {
    id: "bud_1",
    name: "Organization monthly spend",
    scope: "organization",
    targetId: organization.id,
    targetName: organization.name,
    limit: 20_000,
    period: "monthly",
    spent: costSummary.mtd,
    action: "alert",
    enabled: true,
    alertThreshold: 18_000,
    createdAt: daysAgo(180),
  },
  ...teamBudgets.map((entry, index) => {
    const team = teams.find((candidate) => candidate.slug === entry.slug);
    if (!team) {
      throw new Error(`budget references unknown team: ${entry.slug}`);
    }
    return {
      id: `bud_${index + 2}`,
      name: `${team.name} monthly spend`,
      scope: "team" as const,
      targetId: team.id,
      targetName: team.name,
      limit: entry.limit,
      period: "monthly" as const,
      spent: team.spendMtd,
      action: entry.action,
      enabled: entry.enabled,
      alertThreshold: entry.alertThreshold,
      createdAt: daysAgo(150 - index * 8),
    };
  }),
  ...projectBudgets.map((entry, index) => {
    const project = projects.find(
      (candidate) => candidate.slug === entry.slug,
    );
    if (!project) {
      throw new Error(`budget references unknown project: ${entry.slug}`);
    }
    return {
      id: `bud_${index + 8}`,
      name: `${project.name} monthly spend`,
      scope: "project" as const,
      targetId: project.id,
      targetName: project.name,
      limit: entry.limit,
      period: "monthly" as const,
      spent: project.spendMtd,
      action: entry.action,
      enabled: entry.enabled,
      alertThreshold: entry.alertThreshold,
      createdAt: daysAgo(120 - index * 14),
    };
  }),
];
