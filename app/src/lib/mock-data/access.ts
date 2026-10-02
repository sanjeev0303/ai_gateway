import { models } from "./models";
import { NOW } from "./clock";
import type { ModelAccessGrant, ModelDenial, ModelEntitlement } from "@/types";

const daysAgo = (n: number) => new Date(NOW.getTime() - n * 86_400_000).toISOString();
const daysAhead = (n: number) => new Date(NOW.getTime() + n * 86_400_000).toISOString();

/**
 * Grant and denial labels are resolved from the model fixture so a renamed
 * model can never leave a stale string behind on the access page.
 */
const modelName = (id: string) =>
  models.find((model) => model.id === id)?.displayName ?? id;

export const modelAccessGrants: ModelAccessGrant[] = [
  {
    id: "mag_1",
    name: "Frontier models for platform engineering",
    modelIds: ["mdl_claude_sonnet", "mdl_claude_opus", "mdl_gpt5"],
    principalIds: ["tm_platform"],
    scope: "team",
    status: "active",
    createdAt: daysAgo(240),
    requests24h: 48_210,
  },
  {
    id: "mag_2",
    name: "Support assistant — approved tier only",
    modelIds: ["mdl_claude_haiku", "mdl_gemini_flash"],
    principalIds: ["tm_support"],
    scope: "team",
    status: "active",
    createdAt: daysAgo(180),
    requests24h: 22_940,
  },
  {
    id: "mag_3",
    name: "Self-serve catalog for all members",
    modelIds: [
      "mdl_gpt5_mini",
      "mdl_gemini_flash",
      "mdl_mistral_small",
      "mdl_embed_v4",
    ],
    principalIds: [],
    scope: "organization",
    status: "active",
    createdAt: daysAgo(365),
    requests24h: 9_318,
  },
  {
    id: "mag_4",
    name: "Legal — frontier review models",
    modelIds: ["mdl_claude_opus"],
    principalIds: ["tm_legal"],
    scope: "team",
    status: "active",
    createdAt: daysAgo(120),
    expiresAt: daysAhead(45),
    requests24h: 1_204,
  },
  {
    id: "mag_5",
    name: "Data science — private Bedrock catalog",
    modelIds: ["mdl_nova_pro", "mdl_titan_text", "mdl_embed_v4"],
    principalIds: ["tm_data"],
    scope: "team",
    status: "pending",
    createdAt: daysAgo(6),
    requests24h: 0,
  },
  {
    id: "mag_6",
    name: "Legacy Turbo pilot (retired upstream)",
    modelIds: ["mdl_legacy_turbo"],
    principalIds: ["tm_platform"],
    scope: "team",
    status: "expired",
    createdAt: daysAgo(420),
    expiresAt: daysAgo(30),
    requests24h: 0,
  },
];

export const modelDenials: ModelDenial[] = [
  {
    id: "mdn_1",
    principalName: "Marketing",
    modelName: "Claude Opus 4.5",
    reason: "No frontier grant for this team; request an exception in the console.",
    blockedAt: daysAgo(0),
    requestsBlocked24h: 184,
  },
  {
    id: "mdn_2",
    principalName: "Legal compliance",
    modelName: "GPT-5",
    reason: "Data residency policy restricts this model to EU regions.",
    blockedAt: daysAgo(1),
    requestsBlocked24h: 42,
  },
  {
    id: "mdn_3",
    principalName: "Customer support",
    modelName: "Nova Pro",
    reason: "Voice and vision inputs are not cleared for the support workload.",
    blockedAt: daysAgo(2),
    requestsBlocked24h: 17,
  },
  {
    id: "mdn_4",
    principalName: "Security engineering",
    modelName: "Command R+",
    reason: "Model is on the deprecated list; use Mistral Large instead.",
    blockedAt: daysAgo(3),
    requestsBlocked24h: 3,
  },
  {
    id: "mdn_5",
    principalName: "Data insights",
    modelName: "Legacy Turbo",
    reason: "Model retired upstream on 2026-08-30; all calls are refused.",
    blockedAt: daysAgo(30),
    requestsBlocked24h: 0,
  },
];

/** Plan-gated entitlements, rendered as the entitlement matrix on the page. */
export const modelEntitlements: ModelEntitlement[] = [
  {
    tier: "free",
    modelNames: [
      modelName("mdl_gpt5_mini"),
      modelName("mdl_gemini_flash"),
      modelName("mdl_embed_v4"),
    ],
    spendCap: 50,
    requiresApproval: false,
  },
  {
    tier: "starter",
    modelNames: [
      modelName("mdl_claude_haiku"),
      modelName("mdl_gpt5_mini"),
      modelName("mdl_gemini_flash"),
      modelName("mdl_mistral_small"),
      modelName("mdl_embed_v4"),
    ],
    spendCap: 500,
    requiresApproval: false,
  },
  {
    tier: "growth",
    modelNames: [
      modelName("mdl_claude_sonnet"),
      modelName("mdl_gpt5"),
      modelName("mdl_o3"),
      modelName("mdl_gemini_pro"),
      modelName("mdl_mistral_large"),
      modelName("mdl_embed_v4"),
    ],
    spendCap: 5_000,
    requiresApproval: false,
  },
  {
    tier: "enterprise",
    modelNames: models
      .filter((model) => model.status !== "retired")
      .map((model) => model.displayName),
    spendCap: null,
    requiresApproval: true,
  },
];
