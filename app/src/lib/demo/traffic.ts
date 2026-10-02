import { DEMO_NOW, demoDayKey } from "./clock";
import { models, people, personById, projectById } from "./entities";
import { hash, id, randFloat, randInt, rng, round, weightedPick } from "./rng";
import type {
  DemoRequest,
  HourBucket,
  ModelModality,
  RequestKind,
  RequestStatus,
  TrafficCell,
  TrafficSlice,
} from "./types";

const HISTORY_DAYS = 90;
const DETAILED_DAYS = 10;
const BASE_REQUESTS = 1_830;
const LATENCY_SAMPLES_PER_DAY = 96;
const PROMPT_VERSION = "2026_09_v4";

/** Traffic share per project. Marketing volume is high but cheap per call. */
const PROJECT_WEIGHT: Record<string, number> = {
  proj_portal: 34,
  proj_marketing: 26,
  proj_erp: 16,
  proj_research: 15,
  proj_mobile: 9,
};

/** Usage intensity per person, independent of the projects they can bill to. */
const PERSON_WEIGHT: Record<string, number> = {
  per_sarah: 18,
  per_marcus: 15,
  per_priya: 13,
  per_tom: 10,
  per_jonas: 7,
  per_emily: 6,
  per_aisha: 16,
  per_daniel: 12,
  per_grace: 8,
  per_victor: 3,
};

/** Call-shape mix per project. Each entry is a [kind, weight] pair. */
const KIND_MIX: Record<string, [RequestKind, number][]> = {
  proj_portal: [
    ["chat", 58],
    ["completion", 16],
    ["embedding", 10],
    ["tool_call", 11],
    ["vision", 5],
  ],
  proj_marketing: [
    ["completion", 40],
    ["batch", 22],
    ["chat", 20],
    ["embedding", 8],
    ["vision", 10],
  ],
  proj_erp: [
    ["completion", 30],
    ["tool_call", 26],
    ["chat", 18],
    ["embedding", 16],
    ["vision", 10],
  ],
  proj_research: [
    ["embedding", 38],
    ["batch", 24],
    ["chat", 18],
    ["tool_call", 12],
    ["vision", 8],
  ],
  proj_mobile: [
    ["chat", 34],
    ["completion", 26],
    ["embedding", 20],
    ["tool_call", 12],
    ["vision", 8],
  ],
};

/** Model popularity, biased towards cheap and fast models. */
const MODEL_WEIGHT: Record<string, number> = {
  mdl_claude_haiku: 26,
  mdl_gpt5_mini: 20,
  mdl_gemini_flash: 18,
  mdl_claude_sonnet: 16,
  mdl_embed_openai: 14,
  mdl_gpt5: 7,
  mdl_gemini_pro: 5,
  mdl_o4_mini: 4,
  mdl_gpt41: 3,
  mdl_claude_3_5_haiku: 3,
  mdl_gemini_2_flash: 2,
  mdl_gemini_embed: 2,
  mdl_claude_opus: 1.4,
  mdl_imagen4: 0.6,
};

const TOKEN_PROFILE: Record<RequestKind, { input: number; output: number }> = {
  chat: { input: 1_450, output: 620 },
  completion: { input: 880, output: 360 },
  embedding: { input: 1_150, output: 0 },
  tool_call: { input: 2_600, output: 180 },
  vision: { input: 1_750, output: 320 },
  batch: { input: 5_200, output: 900 },
};

const KIND_MODALITIES: Record<RequestKind, ModelModality[]> = {
  chat: ["text", "multimodal"],
  completion: ["text", "multimodal"],
  embedding: ["embedding"],
  tool_call: ["text", "multimodal"],
  vision: ["vision", "multimodal"],
  batch: ["text", "multimodal"],
};

const ERROR_SHARE = 0.024;
const BLOCKED_SHARE = 0.011;
const STATUS_MIX: [RequestStatus, number][] = [
  ["success", 100 - (ERROR_SHARE + BLOCKED_SHARE) * 100],
  ["error", ERROR_SHARE * 100],
  ["blocked", BLOCKED_SHARE * 100],
];

/** Policy ids referenced by blocked requests. Defined in ./policies. */
const BLOCKING_POLICY_IDS = [
  "pol_pii_redaction",
  "pol_no_training",
  "pol_code_secrets",
  "pol_cost_ceiling",
];

const GUARDRAIL_IDS = ["grd_pii_output", "grd_secrets_output", "grd_toxicity"];

const SUMMARY_TEMPLATES: Record<RequestKind, string[]> = {
  chat: [
    "Summarised the last 6 support threads for account {acct}",
    "Answered a workspace question about invoice {inv}",
    "Drafted a reply to a customer asking about {topic}",
    "Explained the difference between two plan tiers",
  ],
  completion: [
    "Rewrote the empty-state copy for {topic}",
    "Completed a campaign description for {topic}",
    "Filled in a release note entry for {topic}",
  ],
  embedding: [
    "Embedded {n} support documents for semantic search",
    "Embedded the {topic} knowledge base chunks",
    "Embedded {n} renewal records for retrieval",
  ],
  tool_call: [
    "Called crm.lookup_account then drafted a renewal summary",
    "Called billing.get_invoice and extracted payment terms",
    "Called search.index and refreshed {n} documents",
  ],
  vision: [
    "Reviewed a product screenshot and listed UI defects",
    "Described a campaign asset and produced alt text",
    "Generated a hero treatment variant for {topic}",
  ],
  batch: [
    "Batched {n} campaign variants for {topic}",
    "Batched {n} SEO page rewrites",
    "Batched {n} document classification jobs",
  ],
};

const TOPIC_POOL = [
  "the Q3 renewal",
  "onboarding",
  "billing",
  "the mobile release",
  "SOC 2 evidence",
  "the pricing page",
  "warehouse sync",
  "the migration",
];

const ACCOUNT_POOL = ["Northwind", "Acme Logistics", "Bright Harbor", "Kestrel Co"];
const INVOICE_POOL = ["INV-20481", "INV-20492", "INV-20507", "INV-20513"];

/** UTC demand curve: business hours peak around 14:00, troughs overnight. */
const HOUR_SHAPE = Array.from(
  { length: 24 },
  (_, hour) =>
    0.3 + 0.7 * (0.5 + 0.5 * Math.cos(((hour - 14) / 24) * 2 * Math.PI)),
);

const HOURS_ELAPSED_TODAY = DEMO_NOW.getUTCHours();

const modelMap = new Map(models.map((model) => [model.id, model]));

interface Combo {
  id: string;
  personId: string;
  teamId: string;
  projectId: string;
  modelId: string;
  providerId: string;
  environment: string;
  kind: RequestKind;
  weight: number;
  share: number;
  inputAvg: number;
  outputAvg: number;
  unitCost: number;
}

/** Static metadata per combo id, used to test filter predicates. */
export const comboMeta: Record<
  string,
  Pick<Combo, "personId" | "teamId" | "projectId" | "modelId" | "providerId" | "environment" | "kind">
> = {};

function unitCostFor(
  modelId: string,
  inputAvg: number,
  outputAvg: number,
): number {
  const model = modelMap.get(modelId);
  if (!model) return 0;
  if (model.flatCostPerCall !== undefined) return model.flatCostPerCall;
  return (
    (inputAvg / 1_000_000) * model.inputPerMTok +
    (outputAvg / 1_000_000) * model.outputPerMTok
  );
}

function buildCombos(): Combo[] {
  const combos: Combo[] = [];
  for (const person of people) {
    const personWeight = PERSON_WEIGHT[person.id] ?? 1;
    if (person.status !== "active") continue;
    for (const projectId of person.projectIds) {
      const project = projectById.get(projectId);
      if (!project || project.status !== "active") continue;
      const projectWeight = PROJECT_WEIGHT[projectId] ?? 1;
      const mix = KIND_MIX[projectId] ?? [["chat", 1]];
      for (const [kind, kindWeight] of mix) {
        const eligible = project.allowedModelIds.filter((modelId) => {
          const model = modelMap.get(modelId);
          if (!model || model.status === "deprecated") return false;
          return KIND_MODALITIES[kind].includes(model.modality);
        });
        if (eligible.length === 0) continue;
        for (const modelId of eligible) {
          const modelWeight = MODEL_WEIGHT[modelId] ?? 1;
          const profile = TOKEN_PROFILE[kind];
          const comboId = `${person.id}|${projectId}|${kind}|${modelId}`;
          combos.push({
            id: comboId,
            personId: person.id,
            teamId: project.teamId,
            projectId,
            modelId,
            providerId: modelMap.get(modelId)?.providerId ?? "",
            environment: project.environment,
            kind,
            weight: personWeight * projectWeight * kindWeight * modelWeight,
            share: 0,
            inputAvg: profile.input,
            outputAvg: profile.output,
            unitCost: unitCostFor(modelId, profile.input, profile.output),
          });
        }
      }
    }
  }
  const total = combos.reduce((sum, combo) => sum + combo.weight, 0);
  for (const combo of combos) {
    combo.share = combo.weight / total;
    comboMeta[combo.id] = {
      personId: combo.personId,
      teamId: combo.teamId,
      projectId: combo.projectId,
      modelId: combo.modelId,
      providerId: combo.providerId,
      environment: combo.environment,
      kind: combo.kind,
    };
  }
  return combos;
}

const COMBOS = buildCombos();

function requestsForDay(offset: number, r: () => number): number {
  const day = new Date(`${demoDayKey(offset)}T00:00:00Z`);
  const weekday = day.getUTCDay();
  const weekendFactor = weekday === 0 || weekday === 6 ? 0.42 : 1;
  const trend = 1 + (HISTORY_DAYS - 1 - offset) * 0.0016;
  const noise = 0.94 + r() * 0.12;
  const hours = offset === 0 ? HOURS_ELAPSED_TODAY : 24;
  const scaled = BASE_REQUESTS * weekendFactor * trend * noise * (hours / 24);
  return Math.max(60, Math.round(scaled));
}

function emptySlice(): TrafficSlice {
  return {
    requests: 0,
    inputTokens: 0,
    outputTokens: 0,
    cost: 0,
    errors: 0,
    blocked: 0,
  };
}

function addTo(slice: TrafficSlice, other: TrafficSlice): void {
  slice.requests += other.requests;
  slice.inputTokens += other.inputTokens;
  slice.outputTokens += other.outputTokens;
  slice.cost += other.cost;
  slice.errors += other.errors;
  slice.blocked += other.blocked;
}

function ensure(
  bucket: Record<string, TrafficSlice>,
  key: string,
): TrafficSlice {
  const existing = bucket[key];
  if (existing) return existing;
  const created = emptySlice();
  bucket[key] = created;
  return created;
}

function percentile(sorted: readonly number[], p: number): number {
  if (sorted.length === 0) return 0;
  const index = (sorted.length - 1) * p;
  const low = Math.floor(index);
  const high = Math.ceil(index);
  return sorted[low] + (sorted[high] - sorted[low]) * (index - low);
}

function sampleLatency(combo: Combo, r: () => number): number {
  const model = modelMap.get(combo.modelId);
  const ttft = Math.round((model?.ttftMs ?? 800) * randFloat(r, 0.8, 1.9));
  const output = combo.outputAvg > 0 ? combo.outputAvg * randFloat(r, 0.6, 1.4) : 0;
  return Math.round(ttft + (output / 45) * 1_000);
}

function dayLatencyPercentiles(day: string): { p50: number; p95: number } {
  const r = rng(hash(`${day}:latency`));
  const samples: number[] = [];
  for (let i = 0; i < LATENCY_SAMPLES_PER_DAY; i += 1) {
    const combo = weightedPick(
      r,
      COMBOS.map((entry) => ({ item: entry, weight: entry.weight })),
    );
    samples.push(sampleLatency(combo, r));
  }
  samples.sort((a, b) => a - b);
  return {
    p50: Math.round(percentile(samples, 0.5)),
    p95: Math.round(percentile(samples, 0.95)),
  };
}

function summarize(kind: RequestKind, r: () => number): string {
  const pool = SUMMARY_TEMPLATES[kind];
  let text = pool[randInt(r, 0, pool.length - 1)];
  text = text
    .replace("{topic}", TOPIC_POOL[randInt(r, 0, TOPIC_POOL.length - 1)])
    .replace("{acct}", ACCOUNT_POOL[randInt(r, 0, ACCOUNT_POOL.length - 1)])
    .replace("{inv}", INVOICE_POOL[randInt(r, 0, INVOICE_POOL.length - 1)])
    .replace("{n}", String(randInt(r, 8, 240)));
  return text;
}

function uniformHours(
  totals: HourBucket,
  offset: number,
): HourBucket[] {
  const hours = offset === 0 ? HOURS_ELAPSED_TODAY : 24;
  const shapeTotal = HOUR_SHAPE.slice(0, hours).reduce((sum, w) => sum + w, 0);
  const buckets: HourBucket[] = [];
  for (let hour = 0; hour < hours; hour += 1) {
    const share = HOUR_SHAPE[hour] / shapeTotal;
    buckets.push({
      requests: Math.round(totals.requests * share),
      inputTokens: Math.round(totals.inputTokens * share),
      outputTokens: Math.round(totals.outputTokens * share),
      cost: round(totals.cost * share, 6),
    });
  }
  const last = buckets[buckets.length - 1]!;
  last.requests += totals.requests - buckets.reduce((sum, b) => sum + b.requests, 0);
  last.inputTokens += totals.inputTokens - buckets.reduce((sum, b) => sum + b.inputTokens, 0);
  last.outputTokens += totals.outputTokens - buckets.reduce((sum, b) => sum + b.outputTokens, 0);
  last.cost = round(totals.cost - buckets.slice(0, -1).reduce((sum, b) => sum + b.cost, 0), 6);
  return buckets;
}

function buildClosedFormCell(
  offset: number,
  day: string,
  dayRequests: number,
  r: () => number,
): TrafficCell {
  const cell: TrafficCell = {
    day,
    requests: 0,
    inputTokens: 0,
    outputTokens: 0,
    cost: 0,
    errors: 0,
    blocked: 0,
    p50LatencyMs: 0,
    p95LatencyMs: 0,
    hourly: [],
    byPerson: {},
    byProject: {},
    byModel: {},
    byProvider: {},
    byTeam: {},
    byKind: {} as Record<RequestKind, TrafficSlice>,
    byEnvironment: {} as Record<string, TrafficSlice>,
    byStatus: {} as Record<RequestStatus, TrafficSlice>,
    byCombo: {},
    detailed: false,
  };

  for (const combo of COMBOS) {
    const expected = dayRequests * combo.share;
    if (expected < 0.01) continue;
    const model = modelMap.get(combo.modelId);
    const project = projectById.get(combo.projectId);
    if (!model || !project) continue;

    const errors = expected * ERROR_SHARE;
    const blocked = expected * BLOCKED_SHARE;
    const billed = expected - errors - blocked;

    const jitter = randFloat(r, 0.78, 1.22);
    const inputTokens = Math.round(combo.inputAvg * jitter * (billed + errors));
    const outputTokens = Math.round(combo.outputAvg * jitter * billed);
    const cost =
      combo.unitCost * billed +
      unitCostFor(combo.modelId, combo.inputAvg * jitter, 0) * errors;

    const slice: TrafficSlice = {
      requests: expected,
      inputTokens,
      outputTokens,
      cost,
      errors,
      blocked,
    };

    cell.requests += expected;
    cell.inputTokens += inputTokens;
    cell.outputTokens += outputTokens;
    cell.cost += cost;
    cell.errors += errors;
    cell.blocked += blocked;

    addTo(ensure(cell.byPerson, combo.personId), slice);
    addTo(ensure(cell.byProject, combo.projectId), slice);
    addTo(ensure(cell.byModel, combo.modelId), slice);
    addTo(ensure(cell.byProvider, model.providerId), slice);
    addTo(ensure(cell.byTeam, combo.teamId), slice);
    addTo(ensure(cell.byKind, combo.kind), slice);
    addTo(ensure(cell.byEnvironment, project.environment), slice);
    const statusErrorCost = slice.cost * (errors / expected);
    const statusErrorIn = slice.inputTokens * (errors / expected);
    addTo(ensure(cell.byStatus, "success"), {
      requests: billed,
      inputTokens: slice.inputTokens - statusErrorIn,
      outputTokens,
      cost: slice.cost - statusErrorCost,
      errors: 0,
      blocked: 0,
    });
    addTo(ensure(cell.byStatus, "error"), {
      requests: errors,
      inputTokens: statusErrorIn,
      outputTokens: 0,
      cost: statusErrorCost,
      errors,
      blocked: 0,
    });
    addTo(ensure(cell.byStatus, "blocked"), {
      requests: blocked,
      inputTokens: 0,
      outputTokens: 0,
      cost: 0,
      errors: 0,
      blocked,
    });
    addTo(ensure(cell.byCombo, combo.id), slice);
  }

  const latency = dayLatencyPercentiles(day);
  cell.p50LatencyMs = latency.p50;
  cell.p95LatencyMs = latency.p95;
  cell.hourly = uniformHours(
    {
      requests: cell.requests,
      inputTokens: cell.inputTokens,
      outputTokens: cell.outputTokens,
      cost: cell.cost,
    },
    offset,
  );
  return cell;
}

function buildDetailedCell(
  offset: number,
  day: string,
  dayRequests: number,
  sink: DemoRequest[],
  startIndex: number,
): TrafficCell {
  const r = rng(hash(`${day}:records`));
  const hours = offset === 0 ? HOURS_ELAPSED_TODAY : 24;
  const hourShapeTotal = HOUR_SHAPE.slice(0, hours).reduce((sum, w) => sum + w, 0);
  const hourEntries = Array.from({ length: hours }, (_, hour) => ({
    item: hour,
    weight: HOUR_SHAPE[hour] / hourShapeTotal,
  }));

  const cell: TrafficCell = {
    day,
    requests: 0,
    inputTokens: 0,
    outputTokens: 0,
    cost: 0,
    errors: 0,
    blocked: 0,
    p50LatencyMs: 0,
    p95LatencyMs: 0,
    hourly: Array.from({ length: hours }, () => ({
      requests: 0,
      inputTokens: 0,
      outputTokens: 0,
      cost: 0,
    })),
    byPerson: {},
    byProject: {},
    byModel: {},
    byProvider: {},
    byTeam: {},
    byKind: {} as Record<RequestKind, TrafficSlice>,
    byEnvironment: {} as Record<string, TrafficSlice>,
    byStatus: {} as Record<RequestStatus, TrafficSlice>,
    byCombo: {},
    detailed: true,
  };

  const latencies: number[] = [];
  let requestIndex = startIndex;

  for (let i = 0; i < dayRequests; i += 1) {
    const combo = weightedPick(
      r,
      COMBOS.map((entry) => ({ item: entry, weight: entry.weight })),
    );
    const model = modelMap.get(combo.modelId);
    const project = projectById.get(combo.projectId);
    const person = personById.get(combo.personId);
    if (!model || !project || !person) continue;

    const hour = weightedPick(r, hourEntries);
    const status = weightedPick(
      r,
      STATUS_MIX.map(([status, weight]) => ({ item: status, weight })),
    );

    const jitter = randFloat(r, 0.78, 1.22);
    let inputTokens = Math.min(
      Math.round(combo.inputAvg * jitter),
      model.contextWindowTokens,
    );
    let outputTokens = Math.min(
      Math.round(combo.outputAvg * jitter),
      model.maxOutputTokens,
    );
    let cost = unitCostFor(combo.modelId, inputTokens, outputTokens);

    if (status === "error") {
      outputTokens = 0;
      cost = unitCostFor(combo.modelId, inputTokens, 0);
    } else if (status === "blocked") {
      inputTokens = 0;
      outputTokens = 0;
      cost = 0;
    }

    const ttftMs = Math.round(model.ttftMs * randFloat(r, 0.8, 1.9));
    const latencyMs = Math.round(
      ttftMs + (outputTokens > 0 ? (outputTokens / 45) * 1_000 : 0),
    );
    latencies.push(latencyMs);

    const minute = randInt(r, 0, 59);
    const second = randInt(r, 0, 59);
    const occurredAt = `${day}T${String(hour).padStart(2, "0")}:${String(minute).padStart(2, "0")}:${String(second).padStart(2, "0")}.000Z`;

    const request: DemoRequest = {
      id: id("req", requestIndex, 7),
      occurredAt,
      day,
      hour,
      personId: person.id,
      teamId: project.teamId,
      projectId: project.id,
      modelId: model.id,
      providerId: model.providerId,
      environment: project.environment,
      kind: combo.kind,
      status,
      inputTokens,
      outputTokens,
      cost,
      latencyMs,
      ttftMs,
      summary: summarize(combo.kind, r),
    };
    if (status === "blocked") {
      const useGuardrail = r() < 0.4;
      if (useGuardrail) {
        request.blockedByGuardrailId =
          GUARDRAIL_IDS[randInt(r, 0, GUARDRAIL_IDS.length - 1)];
      } else {
        request.blockedByPolicyId =
          BLOCKING_POLICY_IDS[randInt(r, 0, BLOCKING_POLICY_IDS.length - 1)];
      }
    }
    requestIndex += 1;
    sink.push(request);

    const slice: TrafficSlice = {
      requests: 1,
      inputTokens,
      outputTokens,
      cost,
      errors: status === "error" ? 1 : 0,
      blocked: status === "blocked" ? 1 : 0,
    };
    cell.requests += 1;
    cell.inputTokens += inputTokens;
    cell.outputTokens += outputTokens;
    cell.cost += cost;
    cell.errors += slice.errors;
    cell.blocked += slice.blocked;

    const bucket = cell.hourly[hour];
    bucket.requests += 1;
    bucket.inputTokens += inputTokens;
    bucket.outputTokens += outputTokens;
    bucket.cost += cost;

    addTo(ensure(cell.byPerson, person.id), slice);
    addTo(ensure(cell.byProject, project.id), slice);
    addTo(ensure(cell.byModel, model.id), slice);
    addTo(ensure(cell.byProvider, model.providerId), slice);
    addTo(ensure(cell.byTeam, project.teamId), slice);
    addTo(ensure(cell.byKind, combo.kind), slice);
    addTo(ensure(cell.byEnvironment, project.environment), slice);
    addTo(ensure(cell.byStatus, status), slice);
    addTo(ensure(cell.byCombo, combo.id), slice);
  }

  cell.cost = round(cell.cost, 6);
  latencies.sort((a, b) => a - b);
  cell.p50LatencyMs = Math.round(percentile(latencies, 0.5));
  cell.p95LatencyMs = Math.round(percentile(latencies, 0.95));
  return cell;
}

function build(): { cells: TrafficCell[]; requests: DemoRequest[] } {
  const cells: TrafficCell[] = [];
  const requests: DemoRequest[] = [];
  let requestIndex = 1;

  for (let offset = HISTORY_DAYS - 1; offset >= 0; offset -= 1) {
    const day = demoDayKey(offset);
    const r = rng(hash(`${day}:demand`));
    const dayRequests = requestsForDay(offset, r);
    const detailed = offset < DETAILED_DAYS;
    const cell = detailed
      ? buildDetailedCell(offset, day, dayRequests, requests, requestIndex)
      : buildClosedFormCell(offset, day, dayRequests, r);
    if (detailed) requestIndex += dayRequests;
    cells.push(cell);
  }

  return { cells, requests };
}

const built = build();

/** All 90 days, oldest first. The only source of aggregate truth. */
export const trafficCells: TrafficCell[] = built.cells;

/** Individual requests for the most recent days, newest first. */
export const requests: DemoRequest[] = built.requests;

export const requestById = new Map(requests.map((request) => [request.id, request]));

export const LATENCY_SAMPLE_METHOD = `${LATENCY_SAMPLES_PER_DAY}-sample per day`;
export const PROMPT_TEMPLATE_VERSION = PROMPT_VERSION;
export const HISTORY_WINDOW_DAYS = HISTORY_DAYS;
export const DETAILED_WINDOW_DAYS = DETAILED_DAYS;
