/**
 * Acme Corp — the demo organization.
 *
 * These are pure dimensions: identity, description and relationships only. No
 * spend, request count or usage figure lives here; every number is aggregated
 * from `traffic.ts`. Teams and projects reference people by id, and the traffic
 * generator reads those same ids, which is what makes every rollup reconcile.
 */

import { DEMO_NOW_ISO, demoDaysAgo } from "./clock";
import type {
  DemoModel,
  DemoOrganization,
  DemoPerson,
  DemoProject,
  DemoProvider,
  DemoTeam,
} from "./types";

/* ------------------------------------------------------------------ *
 * Organization
 * ------------------------------------------------------------------ */

export const organization: DemoOrganization = {
  id: "org_acme",
  name: "Acme Corp",
  slug: "acme",
  plan: "scale",
  status: "active",
  createdAt: "2024-03-11T09:00:00Z",

};

/* ------------------------------------------------------------------ *
 * Teams
 * ------------------------------------------------------------------ */

export const teams: DemoTeam[] = [
  {
    id: "team_eng",
    name: "Engineering",
    slug: "engineering",
    description:
      "Builds and operates the Client Portal, Internal ERP, Mobile Application and Research workloads.",
    leadId: "per_sarah",
    memberIds: ["per_sarah", "per_marcus", "per_priya", "per_tom"],
    projectIds: ["proj_portal", "proj_erp", "proj_mobile", "proj_research"],
    createdAt: "2024-03-11T09:00:00Z",
    monthlyBudget: 220,
  },
  {
    id: "team_design",
    name: "Design",
    slug: "design",
    description:
      "Designs and reviews the Client Portal and Marketing Website experiences, including accessibility and content review.",
    leadId: "per_jonas",
    memberIds: ["per_jonas", "per_emily"],
    projectIds: ["proj_portal", "proj_marketing"],
    createdAt: "2024-03-20T11:00:00Z",
    monthlyBudget: 60,
  },
  {
    id: "team_marketing",
    name: "Marketing",
    slug: "marketing",
    description:
      "Runs campaign content generation and SEO analysis for the Marketing Website.",
    leadId: "per_aisha",
    memberIds: ["per_aisha", "per_daniel"],
    projectIds: ["proj_marketing"],
    createdAt: "2024-04-02T14:30:00Z",
    monthlyBudget: 90,
  },
  {
    id: "team_management",
    name: "Management",
    slug: "management",
    description:
      "Sets policy, approves spend and reviews Research output. The only team with organization-wide visibility.",
    leadId: "per_grace",
    memberIds: ["per_grace", "per_victor"],
    projectIds: ["proj_research", "proj_erp"],
    createdAt: "2024-03-11T09:00:00Z",
    monthlyBudget: 40,
  },
];

/* ------------------------------------------------------------------ *
 * People
 * ------------------------------------------------------------------ */

export const people: DemoPerson[] = [
  {
    id: "per_sarah",
    name: "Sarah Chen",
    email: "sarah.chen@acme.example",
    title: "Staff Engineer",
    role: "owner",
    status: "active",
    teamId: "team_eng",
    projectIds: ["proj_portal", "proj_erp"],
    createdAt: "2024-03-11T09:05:00Z",
    lastActiveAt: demoDaysAgo(0),
    avatarHue: 214,
    hasPersonalToken: true,
    mfaEnrolled: true,
  },
  {
    id: "per_marcus",
    name: "Marcus Webb",
    email: "marcus.webb@acme.example",
    title: "Senior Backend Engineer",
    role: "developer",
    status: "active",
    teamId: "team_eng",
    projectIds: ["proj_erp", "proj_research"],
    createdAt: "2024-05-06T10:20:00Z",
    lastActiveAt: demoDaysAgo(0),
    avatarHue: 158,
    hasPersonalToken: true,
    mfaEnrolled: true,
  },
  {
    id: "per_priya",
    name: "Priya Nair",
    email: "priya.nair@acme.example",
    title: "ML Engineer",
    role: "developer",
    status: "active",
    teamId: "team_eng",
    projectIds: ["proj_research", "proj_mobile"],
    createdAt: "2024-06-18T08:45:00Z",
    lastActiveAt: demoDaysAgo(0),
    avatarHue: 276,
    hasPersonalToken: false,
    mfaEnrolled: true,
  },
  {
    id: "per_tom",
    name: "Tom Alvarez",
    email: "tom.alvarez@acme.example",
    title: "Frontend Engineer",
    role: "developer",
    status: "active",
    teamId: "team_eng",
    projectIds: ["proj_portal", "proj_mobile"],
    createdAt: "2024-07-30T13:10:00Z",
    lastActiveAt: demoDaysAgo(1),
    avatarHue: 28,
    hasPersonalToken: true,
    mfaEnrolled: false,
  },
  {
    id: "per_jonas",
    name: "Jonas Weber",
    email: "jonas.weber@acme.example",
    title: "Design Systems Lead",
    role: "admin",
    status: "active",
    teamId: "team_design",
    projectIds: ["proj_portal", "proj_marketing"],
    createdAt: "2024-04-08T09:30:00Z",
    lastActiveAt: demoDaysAgo(0),
    avatarHue: 340,
    hasPersonalToken: false,
    mfaEnrolled: true,
  },
  {
    id: "per_emily",
    name: "Emily Nakamura",
    email: "emily.nakamura@acme.example",
    title: "Product Designer",
    role: "member",
    status: "active",
    teamId: "team_design",
    projectIds: ["proj_portal", "proj_marketing"],
    createdAt: "2024-09-16T11:15:00Z",
    lastActiveAt: demoDaysAgo(0),
    avatarHue: 190,
    hasPersonalToken: true,
    mfaEnrolled: true,
  },
  {
    id: "per_aisha",
    name: "Aisha Rahman",
    email: "aisha.rahman@acme.example",
    title: "Growth Marketing Lead",
    role: "admin",
    status: "active",
    teamId: "team_marketing",
    projectIds: ["proj_marketing"],
    createdAt: "2024-04-02T14:35:00Z",
    lastActiveAt: demoDaysAgo(0),
    avatarHue: 46,
    hasPersonalToken: true,
    mfaEnrolled: true,
  },
  {
    id: "per_daniel",
    name: "Daniel O'Brien",
    email: "daniel.obrien@acme.example",
    title: "Content Strategist",
    role: "member",
    status: "active",
    teamId: "team_marketing",
    projectIds: ["proj_marketing"],
    createdAt: "2024-08-12T15:50:00Z",
    lastActiveAt: demoDaysAgo(0),
    avatarHue: 96,
    hasPersonalToken: false,
    mfaEnrolled: true,
  },
  {
    id: "per_grace",
    name: "Grace Lindqvist",
    email: "grace.lindqvist@acme.example",
    title: "Head of Product",
    role: "owner",
    status: "active",
    teamId: "team_management",
    projectIds: ["proj_research", "proj_erp"],
    createdAt: "2024-03-11T09:10:00Z",
    lastActiveAt: demoDaysAgo(0),
    avatarHue: 302,
    hasPersonalToken: true,
    mfaEnrolled: true,
  },
  {
    id: "per_victor",
    name: "Victor Adeyemi",
    email: "victor.adeyemi@acme.example",
    title: "VP Finance & Operations",
    role: "viewer",
    status: "active",
    teamId: "team_management",
    projectIds: ["proj_research"],
    createdAt: "2024-03-15T12:00:00Z",
    lastActiveAt: demoDaysAgo(2),
    avatarHue: 12,
    hasPersonalToken: false,
    mfaEnrolled: true,
  },
];

/* ------------------------------------------------------------------ *
 * Projects
 * ------------------------------------------------------------------ */

export const projects: DemoProject[] = [
  {
    id: "proj_portal",
    name: "Client Portal",
    slug: "client-portal",
    description:
      "Customer-facing workspace. Summarises support threads, drafts renewal summaries and answers account questions.",
    environment: "production",
    teamId: "team_eng",
    ownerId: "per_sarah",
    status: "active",
    createdAt: "2024-03-14T08:00:00Z",
    allowedModelIds: [
      "mdl_claude_sonnet",
      "mdl_claude_haiku",
      "mdl_gpt5",
      "mdl_gpt5_mini",
      "mdl_gemini_flash",
      "mdl_gemini_pro",
      "mdl_embed_openai",
    ],
    monthlyBudget: 120,
    repository: "acme/client-portal",
  },
  {
    id: "proj_erp",
    name: "Internal ERP",
    slug: "internal-erp",
    description:
      "Finance and operations back office. Extracts contract obligations, classifies invoices and answers warehouse questions.",
    environment: "staging",
    teamId: "team_eng",
    ownerId: "per_marcus",
    status: "active",
    createdAt: "2024-05-08T08:00:00Z",
    allowedModelIds: [
      "mdl_claude_opus",
      "mdl_claude_sonnet",
      "mdl_gpt41",
      "mdl_gpt5",
      "mdl_o4_mini",
      "mdl_embed_openai",
    ],
    monthlyBudget: 80,
    repository: "acme/internal-erp",
  },
  {
    id: "proj_marketing",
    name: "Marketing Website",
    slug: "marketing-website",
    description:
      "Public marketing site. Generates campaign copy, rewrites error states and analyses on-page SEO.",
    environment: "production",
    teamId: "team_marketing",
    ownerId: "per_aisha",
    status: "active",
    createdAt: "2024-04-05T08:00:00Z",
    allowedModelIds: [
      "mdl_claude_haiku",
      "mdl_claude_sonnet",
      "mdl_gpt5_mini",
      "mdl_gemini_flash",
      "mdl_gemini_2_flash",
      "mdl_imagen4",
      "mdl_embed_openai",
    ],
    monthlyBudget: 70,
    repository: "acme/marketing-website",
  },
  {
    id: "proj_mobile",
    name: "Mobile Application",
    slug: "mobile-application",
    description:
      "iOS and Android client. Classifies inbound feedback and drafts release notes ahead of each ship.",
    environment: "development",
    teamId: "team_eng",
    ownerId: "per_tom",
    status: "active",
    createdAt: "2024-07-31T08:00:00Z",
    allowedModelIds: [
      "mdl_claude_haiku",
      "mdl_claude_3_5_haiku",
      "mdl_gpt5_mini",
      "mdl_gemini_flash",
      "mdl_gemini_embed",
    ],
    monthlyBudget: 25,
    repository: "acme/mobile-application",
  },
  {
    id: "proj_research",
    name: "Research",
    slug: "research",
    description:
      "Model evaluation and embedding index. Bulk-embeddings and long-context experiments; no customer data.",
    environment: "development",
    teamId: "team_eng",
    ownerId: "per_priya",
    status: "active",
    createdAt: "2024-06-20T08:00:00Z",
    allowedModelIds: [
      "mdl_claude_opus",
      "mdl_claude_sonnet",
      "mdl_gpt41",
      "mdl_o4_mini",
      "mdl_gemini_pro",
      "mdl_embed_openai",
      "mdl_gemini_embed",
    ],
    monthlyBudget: 30,
    repository: "acme/research",
  },
];

/* ------------------------------------------------------------------ *
 * Providers
 * ------------------------------------------------------------------ */

export const providers: DemoProvider[] = [
  {
    id: "prov_anthropic",
    name: "anthropic",
    displayName: "Anthropic",
    status: "operational",
    createdAt: "2024-03-11T10:00:00Z",
    platformFeeMonthly: null,
    region: "us-east-1",
    docsUrl: "https://docs.anthropic.com",
    modelIds: [
      "mdl_claude_opus",
      "mdl_claude_sonnet",
      "mdl_claude_haiku",
      "mdl_claude_3_5_haiku",
    ],
  },
  {
    id: "prov_openai",
    name: "openai",
    displayName: "OpenAI",
    status: "operational",
    createdAt: "2024-03-11T10:05:00Z",
    platformFeeMonthly: null,
    region: "us-east-1",
    docsUrl: "https://platform.openai.com/docs",
    modelIds: [
      "mdl_gpt5",
      "mdl_gpt5_mini",
      "mdl_gpt41",
      "mdl_o4_mini",
      "mdl_embed_openai",
    ],
  },
  {
    id: "prov_google",
    name: "google",
    displayName: "Google",
    status: "degraded",
    createdAt: "2024-04-11T10:00:00Z",
    platformFeeMonthly: null,
    region: "us-central1",
    docsUrl: "https://ai.google.dev/docs",
    modelIds: [
      "mdl_gemini_pro",
      "mdl_gemini_flash",
      "mdl_gemini_2_flash",
      "mdl_gemini_embed",
      "mdl_imagen4",
    ],
  },
];

/* ------------------------------------------------------------------ *
 * Models
 * ------------------------------------------------------------------ */

export const models: DemoModel[] = [
  /* --- Anthropic --- */
  {
    id: "mdl_claude_opus",
    name: "claude-opus-4-1",
    displayName: "Claude Opus 4.1",
    providerId: "prov_anthropic",
    modality: "multimodal",
    status: "ga",
    contextWindowTokens: 200_000,
    maxOutputTokens: 32_000,
    inputPerMTok: 15,
    outputPerMTok: 75,
    description:
      "Highest-capability Anthropic model. Used for contract extraction and long-context evaluation.",
    releasedAt: "2025-08-05T00:00:00Z",
    ttftMs: 480,
  },
  {
    id: "mdl_claude_sonnet",
    name: "claude-sonnet-4-5",
    displayName: "Claude Sonnet 4.5",
    providerId: "prov_anthropic",
    modality: "multimodal",
    status: "ga",
    contextWindowTokens: 200_000,
    maxOutputTokens: 64_000,
    inputPerMTok: 3,
    outputPerMTok: 15,
    description:
      "Balanced default across Acme. Handles support summarisation, extraction and drafting.",
    releasedAt: "2025-09-29T00:00:00Z",
    ttftMs: 320,
  },
  {
    id: "mdl_claude_haiku",
    name: "claude-haiku-4-5",
    displayName: "Claude Haiku 4.5",
    providerId: "prov_anthropic",
    modality: "text",
    status: "ga",
    contextWindowTokens: 200_000,
    maxOutputTokens: 32_000,
    inputPerMTok: 1,
    outputPerMTok: 5,
    description: "Fast and inexpensive. Carries the bulk of classification and routing traffic.",
    releasedAt: "2025-10-15T00:00:00Z",
    ttftMs: 180,
  },
  {
    id: "mdl_claude_3_5_haiku",
    name: "claude-3-5-haiku-latest",
    displayName: "Claude 3.5 Haiku",
    providerId: "prov_anthropic",
    modality: "text",
    status: "deprecated",
    contextWindowTokens: 200_000,
    maxOutputTokens: 8_192,
    inputPerMTok: 0.8,
    outputPerMTok: 4,
    description: "Superseded by Haiku 4.5. Still pinned by the Mobile Application build.",
    releasedAt: "2024-11-04T00:00:00Z",
    ttftMs: 165,
    retiredOn: "2026-12-01T00:00:00Z",
  },

  /* --- OpenAI --- */
  {
    id: "mdl_gpt5",
    name: "gpt-5",
    displayName: "GPT-5",
    providerId: "prov_openai",
    modality: "multimodal",
    status: "ga",
    contextWindowTokens: 400_000,
    maxOutputTokens: 128_000,
    inputPerMTok: 1.25,
    outputPerMTok: 10,
    description:
      "Frontier reasoning model. Reserved for the ERP contract pipeline and portal escalations.",
    releasedAt: "2025-08-07T00:00:00Z",
    ttftMs: 520,
  },
  {
    id: "mdl_gpt5_mini",
    name: "gpt-5-mini",
    displayName: "GPT-5 mini",
    providerId: "prov_openai",
    modality: "text",
    status: "ga",
    contextWindowTokens: 400_000,
    maxOutputTokens: 128_000,
    inputPerMTok: 0.25,
    outputPerMTok: 2,
    description: "Low-cost workhorse for copy generation and high-volume classification.",
    releasedAt: "2025-08-07T00:00:00Z",
    ttftMs: 240,
  },
  {
    id: "mdl_gpt41",
    name: "gpt-4.1",
    displayName: "GPT-4.1",
    providerId: "prov_openai",
    modality: "multimodal",
    status: "ga",
    contextWindowTokens: 1_000_000,
    maxOutputTokens: 32_768,
    inputPerMTok: 2,
    outputPerMTok: 8,
    description: "One-million-token context. Drives the Research long-context evaluation suite.",
    releasedAt: "2025-04-14T00:00:00Z",
    ttftMs: 400,
  },
  {
    id: "mdl_o4_mini",
    name: "o4-mini",
    displayName: "o4-mini",
    providerId: "prov_openai",
    modality: "text",
    status: "ga",
    contextWindowTokens: 200_000,
    maxOutputTokens: 100_000,
    inputPerMTok: 1.1,
    outputPerMTok: 4.4,
    description: "Reasoning-tuned. Used for invoice classification and evaluation scoring.",
    releasedAt: "2025-04-16T00:00:00Z",
    ttftMs: 610,
  },
  {
    id: "mdl_embed_openai",
    name: "text-embedding-3-large",
    displayName: "text-embedding-3-large",
    providerId: "prov_openai",
    modality: "embedding",
    status: "ga",
    contextWindowTokens: 8_191,
    maxOutputTokens: 0,
    inputPerMTok: 0.13,
    outputPerMTok: 0,
    description: "Bulk embedding index for support articles and contract retrieval.",
    releasedAt: "2024-01-25T00:00:00Z",
    ttftMs: 40,
  },

  /* --- Google --- */
  {
    id: "mdl_gemini_pro",
    name: "gemini-2.5-pro",
    displayName: "Gemini 2.5 Pro",
    providerId: "prov_google",
    modality: "multimodal",
    status: "ga",
    contextWindowTokens: 1_000_000,
    maxOutputTokens: 65_536,
    inputPerMTok: 1.25,
    outputPerMTok: 10,
    description: "Long-context multimodal. Used in Research and ERP document review.",
    releasedAt: "2025-06-17T00:00:00Z",
    ttftMs: 640,
  },
  {
    id: "mdl_gemini_flash",
    name: "gemini-2.5-flash",
    displayName: "Gemini 2.5 Flash",
    providerId: "prov_google",
    modality: "multimodal",
    status: "ga",
    contextWindowTokens: 1_000_000,
    maxOutputTokens: 65_536,
    inputPerMTok: 0.3,
    outputPerMTok: 2.5,
    description: "Fast multimodal default for the portal and marketing site.",
    releasedAt: "2025-06-17T00:00:00Z",
    ttftMs: 260,
  },
  {
    id: "mdl_gemini_2_flash",
    name: "gemini-2.0-flash",
    displayName: "Gemini 2.0 Flash",
    providerId: "prov_google",
    modality: "multimodal",
    status: "deprecated",
    contextWindowTokens: 1_000_000,
    maxOutputTokens: 8_192,
    inputPerMTok: 0.1,
    outputPerMTok: 0.4,
    description: "Legacy pricing tier retained for the marketing site batch job.",
    releasedAt: "2025-02-05T00:00:00Z",
    ttftMs: 210,
    retiredOn: "2026-10-15T00:00:00Z",
  },
  {
    id: "mdl_gemini_embed",
    name: "gemini-embedding-001",
    displayName: "Gemini Embedding 001",
    providerId: "prov_google",
    modality: "embedding",
    status: "ga",
    contextWindowTokens: 2_048,
    maxOutputTokens: 0,
    inputPerMTok: 0.15,
    outputPerMTok: 0,
    description: "Secondary embedding index used by Research experiments.",
    releasedAt: "2025-06-01T00:00:00Z",
    ttftMs: 45,
  },
  {
    id: "mdl_imagen4",
    name: "imagen-4.0-generate",
    displayName: "Imagen 4",
    providerId: "prov_google",
    modality: "vision",
    status: "ga",
    contextWindowTokens: 0,
    maxOutputTokens: 0,
    inputPerMTok: 0,
    outputPerMTok: 0,
    description: "Image generation for marketing site hero treatments and social assets.",
    releasedAt: "2025-07-23T00:00:00Z",
    ttftMs: 1_850,
    flatCostPerCall: 0.04,
  },
];

/* ------------------------------------------------------------------ *
 * Lookup indexes
 * ------------------------------------------------------------------ */

const byId = <T extends { id: string }>(items: readonly T[]) =>
  new Map(items.map((item) => [item.id, item]));

export const personById = byId(people);
export const teamById = byId(teams);
export const projectById = byId(projects);
export const providerById = byId(providers);
export const modelById = byId(models);

export const personName = (personId: string): string =>
  personById.get(personId)?.name ?? "Unknown user";
export const teamName = (teamId: string): string =>
  teamById.get(teamId)?.name ?? "Unassigned";
export const projectName = (projectId: string): string =>
  projectById.get(projectId)?.name ?? "Unknown project";
export const providerName = (providerId: string): string =>
  providerById.get(providerId)?.displayName ?? "Unknown provider";
export const modelName = (modelId: string): string =>
  modelById.get(modelId)?.displayName ?? "Unknown model";

/** Ids in a stable display order, so exports and search results are stable. */
export const peopleIds = people.map((person) => person.id);
export const teamsIds = teams.map((team) => team.id);
export const projectsIds = projects.map((project) => project.id);
export const providersIds = providers.map((provider) => provider.id);
export const modelsIds = models.map((model) => model.id);

/** Active (non-invited, non-suspended) people — the selectable principal set. */
export const activePeople = people.filter((person) => person.status === "active");
export const activeProjects = projects.filter((project) => project.status === "active");
export const gaModels = models.filter((model) => model.status === "ga");
export const deprecatedModels = models.filter(
  (model) => model.status === "deprecated",
);

/** Demo org snapshot time, re-exported so pages need one import. */
export const DATASET_GENERATED_AT = DEMO_NOW_ISO;
