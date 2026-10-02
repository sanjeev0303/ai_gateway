import { NOW } from "./clock";
import type { Team } from "@/types";

const daysAgo = (n: number) =>
  new Date(NOW.getTime() - n * 86_400_000).toISOString();

export const teams: Team[] = [
  {
    id: "tm_platform",
    name: "Engineering",
    slug: "engineering",
    description:
      "Owns the gateway runtime, provider routing, and internal developer tooling.",
    memberIds: ["usr_1", "usr_2", "usr_3", "usr_4", "usr_5"],
    leadId: "usr_1",
    projectIds: ["prj_1", "prj_2", "prj_3"],
    createdAt: daysAgo(198),
    monthlyBudget: 4_000,
    spendMtd: 2_860,
  },
  {
    id: "tm_support",
    name: "Product",
    slug: "product",
    description:
      "Builds and operates the support copilot handling first-line ticket triage.",
    memberIds: ["usr_6", "usr_7", "usr_8", "usr_9", "usr_10", "usr_11"],
    leadId: "usr_6",
    projectIds: ["prj_4", "prj_5"],
    createdAt: daysAgo(171),
    monthlyBudget: 2_500,
    spendMtd: 2_340,
  },
  {
    id: "tm_data",
    name: "Research",
    slug: "research",
    description:
      "Analytics enablement: natural-language querying over the warehouse.",
    memberIds: ["usr_12", "usr_13", "usr_14", "usr_15"],
    leadId: "usr_12",
    projectIds: ["prj_6", "prj_7", "prj_8"],
    createdAt: daysAgo(154),
    monthlyBudget: 3_200,
    spendMtd: 2_910,
  },
  {
    id: "tm_legal",
    name: "Security",
    slug: "security",
    description:
      "Contract review and regulatory reporting; subject to residency controls.",
    memberIds: ["usr_16", "usr_17", "usr_18"],
    leadId: "usr_16",
    projectIds: ["prj_9", "prj_10"],
    createdAt: daysAgo(142),
    monthlyBudget: 1_800,
    spendMtd: 1_062,
  },
  {
    id: "tm_marketing",
    name: "Finance",
    slug: "finance",
    description:
      "Campaign content generation and audience segmentation workflows.",
    memberIds: ["usr_19", "usr_20", "usr_21"],
    leadId: "usr_19",
    projectIds: ["prj_11", "prj_12"],
    createdAt: daysAgo(120),
    monthlyBudget: 1_200,
    spendMtd: 1_190,
  },
  {
    id: "tm_security",
    name: "Compliance",
    slug: "compliance",
    description:
      "Runs guardrail policy, red-team suites, and incident response.",
    memberIds: ["usr_22", "usr_23", "usr_24"],
    leadId: "usr_22",
    projectIds: ["prj_13", "prj_14"],
    createdAt: daysAgo(96),
    monthlyBudget: 1_500,
    spendMtd: 640,
  },
];
