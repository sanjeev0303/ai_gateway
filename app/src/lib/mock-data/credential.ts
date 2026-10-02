import { NOW } from "./clock";
import { providers } from "./providers";
import type { Credential } from "@/types";

const daysAgo = (n: number) =>
  new Date(NOW.getTime() - n * 86_400_000).toISOString();
const daysAhead = (n: number) =>
  new Date(NOW.getTime() + n * 86_400_000).toISOString();
const hoursSince = (h: number) =>
  new Date(NOW.getTime() - h * 3_600_000).toISOString();

const provider = (id: string) => {
  const match = providers.find((candidate) => candidate.id === id);
  if (!match) throw new Error(`credential references unknown provider: ${id}`);
  return match;
};

interface Seed {
  id: string;
  name: string;
  kind: Credential["kind"];
  providerId?: string;
  /** Used only for first-party endpoints with no upstream provider. */
  providerName?: string;
  status: Credential["status"];
  maskedValue: string;
  createdAtDaysAgo: number;
  expiresInDays?: number;
  lastUsedHoursAgo: number;
  usedByProjectCount: number;
}

const SEEDS: readonly Seed[] = [
  {
    id: "cred_1",
    name: "Anthropic production key",
    kind: "api_key",
    providerId: "prv_anthropic",
    status: "healthy",
    maskedValue: "sk-ant-...f4c2",
    createdAtDaysAgo: 402,
    lastUsedHoursAgo: 0.4,
    usedByProjectCount: 7,
  },
  {
    id: "cred_2",
    name: "OpenAI production key",
    kind: "api_key",
    providerId: "prv_openai",
    status: "healthy",
    maskedValue: "sk-proj-...9a71",
    createdAtDaysAgo: 388,
    lastUsedHoursAgo: 0.1,
    usedByProjectCount: 6,
  },
  {
    id: "cred_3",
    name: "Google AI service account",
    kind: "service_account",
    providerId: "prv_google",
    status: "healthy",
    maskedValue: "AIza...3e08",
    createdAtDaysAgo: 244,
    lastUsedHoursAgo: 1.8,
    usedByProjectCount: 3,
  },
  {
    id: "cred_4",
    name: "Mistral team key",
    kind: "api_key",
    providerId: "prv_mistral",
    status: "expiring",
    maskedValue: "sk-mis-...b0d9",
    createdAtDaysAgo: 398,
    expiresInDays: 11,
    lastUsedHoursAgo: 6.2,
    usedByProjectCount: 2,
  },
  {
    id: "cred_5",
    name: "Cohore production key",
    kind: "api_key",
    providerId: "prv_cohere",
    status: "expiring",
    maskedValue: "co-...71ea",
    createdAtDaysAgo: 366,
    expiresInDays: 4,
    lastUsedHoursAgo: 2.9,
    usedByProjectCount: 2,
  },
  {
    id: "cred_6",
    name: "Bedrock IAM role",
    kind: "service_account",
    providerId: "prv_bedrock",
    status: "healthy",
    maskedValue: "AROA...5c17",
    createdAtDaysAgo: 232,
    lastUsedHoursAgo: 0.6,
    usedByProjectCount: 2,
  },
  {
    id: "cred_7",
    name: "Eval harness runner token",
    kind: "api_key",
    status: "healthy",
    maskedValue: "sen_...2f44",
    createdAtDaysAgo: 158,
    lastUsedHoursAgo: 3.4,
    usedByProjectCount: 1,
  },
  {
    id: "cred_8",
    name: "Warehouse read-only account",
    kind: "service_account",
    status: "healthy",
    maskedValue: "sa_...88b3",
    createdAtDaysAgo: 176,
    lastUsedHoursAgo: 0.9,
    usedByProjectCount: 3,
  },
  {
    id: "cred_9",
    name: "Incident pager webhook",
    kind: "webhook",
    status: "healthy",
    maskedValue: "https://hooks.../d41f",
    createdAtDaysAgo: 140,
    lastUsedHoursAgo: 11.6,
    usedByProjectCount: 0,
  },
  {
    id: "cred_10",
    name: "Legacy GPT-3.5 key",
    kind: "api_key",
    providerId: "prv_openai",
    status: "revoked",
    maskedValue: "sk-...4b17",
    createdAtDaysAgo: 690,
    lastUsedHoursAgo: 24 * 58,
    usedByProjectCount: 0,
  },
  {
    id: "cred_11",
    name: "Contract review OAuth app",
    kind: "oauth",
    status: "expired",
    maskedValue: "oauth:...c902",
    createdAtDaysAgo: 420,
    expiresInDays: -3,
    lastUsedHoursAgo: 24 * 3,
    usedByProjectCount: 1,
  },
  {
    id: "cred_12",
    name: "Support CRM connector",
    kind: "oauth",
    status: "healthy",
    maskedValue: "oauth:...17ae",
    createdAtDaysAgo: 200,
    expiresInDays: 74,
    lastUsedHoursAgo: 0.2,
    usedByProjectCount: 2,
  },
];

export const credentials: Credential[] = SEEDS.map((seed) => {
  const upstream = seed.providerId ? provider(seed.providerId) : null;
  return {
    id: seed.id,
    name: seed.name,
    kind: seed.kind,
    ...(seed.providerId ? { providerId: seed.providerId } : {}),
    providerName: upstream?.displayName ?? seed.providerName ?? "Sentinel",
    status: seed.status,
    maskedValue: seed.maskedValue,
    createdAt: daysAgo(seed.createdAtDaysAgo),
    ...(seed.expiresInDays === undefined
      ? {}
      : { expiresAt: daysAhead(seed.expiresInDays) }),
    lastUsedAt: hoursSince(seed.lastUsedHoursAgo),
    usedByProjectCount: seed.usedByProjectCount,
  };
});
