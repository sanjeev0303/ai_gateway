import { models } from "./models";
import { NOW } from "./clock";
import { projects } from "./projects";
import { providers } from "./providers";
import { users } from "./users";
import type { ActivityEntry, ActivityEvent, ActivityStatus } from "@/types";

const minutesAgo = (m: number) =>
  new Date(NOW.getTime() - m * 60_000).toISOString();

/** Seeded PRNG so the fixture is organic but byte-identical on every render. */
function mulberry32(seed: number) {
  let a = seed;
  return () => {
    a |= 0;
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

const rand = mulberry32(20260929);
const pick = <T,>(items: readonly T[]): T =>
  items[Math.floor(rand() * items.length)]!;

const EVENTS: readonly ActivityEvent[] = [
  "completion",
  "chat",
  "embedding",
  "tool_call",
];
const STATUSES: readonly ActivityStatus[] = [
  "success",
  "success",
  "success",
  "success",
  "success",
  "error",
  "blocked",
  "pending",
];

const SUMMARIES: Record<ActivityEvent, readonly string[]> = {
  completion: [
    "summarized 4 support threads",
    "rewrote checkout error copy",
    "drafted renewal summary",
    "answered warehouse question",
    "extracted 12 contract obligations",
    "classified inbound ticket",
  ],
  chat: [
    "multi-turn session, 8 turns",
    "clarification loop on schema",
    "answered follow-up question",
    "session continued after tool error",
  ],
  embedding: [
    "embedded 512 document chunks",
    "indexed 1,240 support articles",
    "embedded 96 customer records",
    "refreshed segment vectors",
  ],
  tool_call: [
    "called sql.query, 3 rows returned",
    "called crm.lookup, contact resolved",
    "called search.index, 12 hits",
    "called ticket.create, ticket opened",
  ],
};

const userName = (id: string) =>
  users.find((u) => u.id === id)?.name ?? "Unknown user";

export const activityEntries: ActivityEntry[] = Array.from(
  { length: 120 },
  (_, index) => {
    const project = pick(projects);
    const allowed = project.allowedModelIds;
    const model = models.find((m) => m.id === pick(allowed)) ?? models[0]!;
    const provider = providers.find((p) => p.id === model.providerId)!;
    const event = pick(EVENTS);
    const status = pick(STATUSES);
    const inputTokens = Math.round(400 + rand() * 9_600);
    const outputTokens = Math.round(80 + rand() * 2_400);
    const cost =
      (inputTokens / 1_000_000) * model.pricing.inputPerMTok +
      (outputTokens / 1_000_000) * model.pricing.outputPerMTok;
    const member = project.ownerId;

    return {
      id: `act_${String(index + 1).padStart(4, "0")}`,
      occurredAt: minutesAgo(index * 7 + Math.round(rand() * 6)),
      userId: member,
      userName: userName(member),
      projectId: project.id,
      projectName: project.name,
      modelId: model.id,
      modelName: model.displayName,
      providerId: provider.id,
      providerName: provider.displayName,
      event,
      status,
      inputTokens,
      outputTokens,
      cost,
      latencyMs: Math.round(180 + rand() * 3_400),
      summary: pick(SUMMARIES[event]),
      requestId: `req_${(0x5f3a91 + index * 7919).toString(16).padStart(8, "0")}`,
    };
  },
);
