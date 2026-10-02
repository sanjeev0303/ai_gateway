import { NOW } from "./clock";
import { projects } from "./projects";
import { teams } from "./teams";
import type { Policy } from "@/types";

const daysAgo = (n: number) =>
  new Date(NOW.getTime() - n * 86_400_000).toISOString();

const resolveIds = (
  kind: "team" | "project",
  entities: readonly { id: string; slug: string }[],
  slugs: readonly string[],
) =>
  slugs.map((slug) => {
    const entity = entities.find((candidate) => candidate.slug === slug);
    if (!entity) {
      throw new Error(`${kind} reference has unknown slug: ${slug}`);
    }
    return entity.id;
  });

const teamIds = (...slugs: string[]) => resolveIds("team", teams, slugs);

const projectIds = (...slugs: string[]) =>
  resolveIds("project", projects, slugs);

export const policies: Policy[] = [
  {
    id: "pol_1",
    name: "PII redaction on input",
    description:
      "Strips emails, phone numbers, national identifiers and card numbers from prompts before they reach any provider.",
    category: "pii_redaction",
    enforcement: "enforce",
    scope: "organization",
    targetIds: [],
    enabled: true,
    createdAt: daysAgo(184),
    updatedAt: daysAgo(9),
    evaluations24h: 412_806,
    blocks24h: 0,
    createdById: "usr_22",
  },
  {
    id: "pol_2",
    name: "PII redaction on output",
    description:
      "Removes personal data from model responses before they are returned to an application.",
    category: "pii_redaction",
    enforcement: "enforce",
    scope: "organization",
    targetIds: [],
    enabled: true,
    createdAt: daysAgo(184),
    updatedAt: daysAgo(9),
    evaluations24h: 408_112,
    blocks24h: 1_284,
    createdById: "usr_22",
  },
  {
    id: "pol_3",
    name: "Prompt injection detection",
    description:
      "Detects instructions embedded in retrieved documents that attempt to override system prompts.",
    category: "prompt_injection",
    enforcement: "enforce",
    scope: "organization",
    targetIds: [],
    enabled: true,
    createdAt: daysAgo(178),
    updatedAt: daysAgo(21),
    evaluations24h: 402_940,
    blocks24h: 3_918,
    createdById: "usr_1",
  },
  {
    id: "pol_4",
    name: "Jailbreak pattern blocklist",
    description:
      "Blocks known role-play and instruction-override escape patterns before dispatch.",
    category: "jailbreak",
    enforcement: "enforce",
    scope: "organization",
    targetIds: [],
    enabled: true,
    createdAt: daysAgo(176),
    updatedAt: daysAgo(4),
    evaluations24h: 402_940,
    blocks24h: 612,
    createdById: "usr_22",
  },
  {
    id: "pol_5",
    name: "Toxicity classifier",
    description:
      "Scores responses for toxic content. Audit-only today while thresholds are calibrated.",
    category: "toxicity",
    enforcement: "audit",
    scope: "organization",
    targetIds: [],
    enabled: true,
    createdAt: daysAgo(150),
    updatedAt: daysAgo(38),
    evaluations24h: 398_004,
    blocks24h: 0,
    createdById: "usr_24",
  },
  {
    id: "pol_6",
    name: "EU data residency",
    description:
      "Confines inference for EU-resident personal data to providers processing inside eu-central-1.",
    category: "data_residency",
    enforcement: "enforce",
    scope: "organization",
    targetIds: [],
    enabled: true,
    createdAt: daysAgo(140),
    updatedAt: daysAgo(2),
    evaluations24h: 96_410,
    blocks24h: 74,
    createdById: "usr_16",
  },
  {
    id: "pol_7",
    name: "Per-project spend cap",
    description:
      "Hard-stops a project once its monthly budget is exhausted, rather than allowing overrun.",
    category: "cost_cap",
    enforcement: "enforce",
    scope: "project",
    targetIds: projectIds("r-and-d", "segment-builder"),
    enabled: true,
    createdAt: daysAgo(132),
    updatedAt: daysAgo(17),
    evaluations24h: 0,
    blocks24h: 41,
    createdById: "usr_2",
  },
  {
    id: "pol_8",
    name: "Legal model allowlist",
    description:
      "Restricts the Legal team to Claude Opus and Sonnet, pending the vendor review cycle.",
    category: "model_allowlist",
    enforcement: "enforce",
    scope: "team",
    targetIds: teamIds("security"),
    enabled: true,
    createdAt: daysAgo(121),
    updatedAt: daysAgo(31),
    evaluations24h: 8_214,
    blocks24h: 219,
    createdById: "usr_16",
  },
  {
    id: "pol_9",
    name: "30-day conversation retention",
    description:
      "Purges stored conversation content 30 days after the session ends.",
    category: "retention",
    enforcement: "enforce",
    scope: "organization",
    targetIds: [],
    enabled: true,
    createdAt: daysAgo(118),
    updatedAt: daysAgo(118),
    evaluations24h: 0,
    blocks24h: 0,
    createdById: "usr_16",
  },
  {
    id: "pol_10",
    name: "Security redaction strictness",
    description:
      "Runs the red-team suite with a tighter redaction threshold and masks on first match.",
    category: "pii_redaction",
    enforcement: "enforce",
    scope: "team",
    targetIds: teamIds("compliance"),
    enabled: true,
    createdAt: daysAgo(96),
    updatedAt: daysAgo(12),
    evaluations24h: 3_108,
    blocks24h: 88,
    createdById: "usr_22",
  },
  {
    id: "pol_11",
    name: "Marketing content sampling",
    description:
      "Samples 5% of marketing generations for human review of brand and claim accuracy.",
    category: "retention",
    enforcement: "audit",
    scope: "team",
    targetIds: teamIds("finance"),
    enabled: true,
    createdAt: daysAgo(74),
    updatedAt: daysAgo(26),
    evaluations24h: 41_220,
    blocks24h: 0,
    createdById: "usr_19",
  },
  {
    id: "pol_12",
    name: "Retire gpt-3-5-turbo",
    description:
      "Discontinued: the legacy model is already blocked at the gateway, so the policy is inert.",
    category: "model_allowlist",
    enforcement: "disabled",
    scope: "organization",
    targetIds: [],
    enabled: false,
    createdAt: daysAgo(58),
    updatedAt: daysAgo(44),
    evaluations24h: 0,
    blocks24h: 0,
    createdById: "usr_1",
  },
];
