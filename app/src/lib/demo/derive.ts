import { DEMO_NOW, demoDayLabel } from "./clock";
import {
  models,
  personById,
  projectById,
  providerById,
  teamById,
} from "./entities";
import { round } from "./rng";
import { comboMeta, requests, trafficCells } from "./traffic";
import { blockingPolicies, guardrails } from "./governance";
import {
  DATE_RANGE_DAYS,
  EMPTY_FILTERS,
  REQUEST_KIND_LABEL,
  filtersAreEmpty,
} from "./types";
import type {
  Breakdown,
  DateRange,
  DemoModelGrant,
  DimensionRollup,
  ProjectEnvironment,
  RequestKind,
  RequestStatus,
  SeriesPoint,
  Totals,
  TrafficCell,
  TrafficFilters,
  TrafficSeries,
  TrafficSlice,
} from "./types";

const STATUS_SHARE: Record<RequestStatus, number> = {
  success: 0.965,
  error: 0.024,
  blocked: 0.011,
};

type DimensionKey =
  | "byPerson"
  | "byTeam"
  | "byProject"
  | "byModel"
  | "byProvider"
  | "byKind"
  | "byEnvironment";

const COMBO_META_FIELD = {
  byPerson: "personId",
  byTeam: "teamId",
  byProject: "projectId",
  byModel: "modelId",
  byProvider: "providerId",
  byKind: "kind",
  byEnvironment: "environment",
} as const satisfies Record<DimensionKey, keyof (typeof comboMeta)[string]>;

const modelMap = new Map(models.map((model) => [model.id, model]));

interface Accum {
  requests: number;
  inputTokens: number;
  outputTokens: number;
  cost: number;
  errors: number;
  blocked: number;
  /** Request-weighted sum of model time-to-first-token, for latency estimates. */
  ttftWeighted: number;
}

const emptyAccum = (): Accum => ({
  requests: 0,
  inputTokens: 0,
  outputTokens: 0,
  cost: 0,
  errors: 0,
  blocked: 0,
  ttftWeighted: 0,
});

function addAccum(target: Accum, slice: TrafficSlice, ttftMs = 0): void {
  target.requests += slice.requests;
  target.inputTokens += slice.inputTokens;
  target.outputTokens += slice.outputTokens;
  target.cost += slice.cost;
  target.errors += slice.errors;
  target.blocked += slice.blocked;
  target.ttftWeighted += slice.requests * ttftMs;
}

export function cellsForRange(range: DateRange): TrafficCell[] {
  const days = DATE_RANGE_DAYS[range];
  if (days >= trafficCells.length) return trafficCells;
  return trafficCells.slice(trafficCells.length - days);
}

function statusFactor(filters: TrafficFilters): number {
  if (filters.statuses.length === 0) return 1;
  return filters.statuses.reduce((sum, status) => sum + STATUS_SHARE[status], 0);
}

function comboPasses(
  comboId: string,
  filters: TrafficFilters,
): { meta: (typeof comboMeta)[string] } | null {
  const meta = comboMeta[comboId];
  if (!meta) return null;
  if (filters.personIds.length > 0 && !filters.personIds.includes(meta.personId))
    return null;
  if (filters.teamIds.length > 0 && !filters.teamIds.includes(meta.teamId))
    return null;
  if (filters.projectIds.length > 0 && !filters.projectIds.includes(meta.projectId))
    return null;
  if (filters.modelIds.length > 0 && !filters.modelIds.includes(meta.modelId))
    return null;
  if (filters.providerIds.length > 0 && !filters.providerIds.includes(meta.providerId))
    return null;
  if (filters.kinds.length > 0 && !filters.kinds.includes(meta.kind)) return null;
  if (
    filters.environments.length > 0 &&
    !filters.environments.includes(meta.environment as ProjectEnvironment)
  )
    return null;
  return { meta };
}

function accumulate(
  cell: TrafficCell,
  filters: TrafficFilters,
  filtered: boolean,
): Accum {
  const acc = emptyAccum();
  if (!filtered) {
    addAccum(acc, {
      requests: cell.requests,
      inputTokens: cell.inputTokens,
      outputTokens: cell.outputTokens,
      cost: cell.cost,
      errors: cell.errors,
      blocked: cell.blocked,
    });
    for (const [comboId, slice] of Object.entries(cell.byCombo)) {
      const ttft = modelMap.get(comboMeta[comboId]?.modelId ?? "")?.ttftMs ?? 0;
      acc.ttftWeighted += slice.requests * ttft;
    }
    return acc;
  }

  const factor = statusFactor(filters);
  for (const [comboId, slice] of Object.entries(cell.byCombo)) {
    const passed = comboPasses(comboId, filters);
    if (!passed) continue;
    const ttft = modelMap.get(passed.meta.modelId)?.ttftMs ?? 0;
    addAccum(
      acc,
      {
        requests: slice.requests * factor,
        inputTokens: slice.inputTokens * factor,
        outputTokens: slice.outputTokens * factor,
        cost: slice.cost * factor,
        errors: slice.errors * factor,
        blocked: slice.blocked * factor,
      },
      ttft,
    );
  }
  return acc;
}

function toTotals(acc: Accum, p50: number, p95: number): Totals {
  const totalTokens = acc.inputTokens + acc.outputTokens;
  const requests = Math.round(acc.requests);
  const errors = Math.round(acc.errors);
  const success = Math.max(requests - errors, 0);
  return {
    requests,
    inputTokens: acc.inputTokens,
    outputTokens: acc.outputTokens,
    totalTokens,
    cost: round(acc.cost, 4),
    errors,
    blocked: Math.round(acc.blocked),
    p50LatencyMs: Math.round(p50),
    p95LatencyMs: Math.round(p95),
    errorRate: requests > 0 ? round(errors / requests, 4) : 0,
    successRate: requests > 0 ? round(success / requests, 4) : 0,
  };
}

function latencyFromAcc(acc: Accum): { p50: number; p95: number } {
  if (acc.requests <= 0) return { p50: 0, p95: 0 };
  const meanTtft = acc.ttftWeighted / acc.requests;
  return { p50: meanTtft * 1.15, p95: meanTtft * 2.2 };
}

function buildDailySeries(
  range: DateRange,
  filters: TrafficFilters,
  filtered: boolean,
): TrafficSeries {
  const cells = cellsForRange(range);
  const points: SeriesPoint[] = [];
  const acc = emptyAccum();
  let p50Weighted = 0;
  let p95Weighted = 0;
  let weight = 0;

  for (const cell of cells) {
    const cellAcc = accumulate(cell, filters, filtered);
    points.push({
      label: demoDayLabel(cell.day),
      requests: Math.round(cellAcc.requests),
      inputTokens: Math.round(cellAcc.inputTokens),
      outputTokens: Math.round(cellAcc.outputTokens),
      cost: round(cellAcc.cost, 4),
      errors: Math.round(cellAcc.errors),
    });
    addAccum(acc, {
      requests: cellAcc.requests,
      inputTokens: cellAcc.inputTokens,
      outputTokens: cellAcc.outputTokens,
      cost: cellAcc.cost,
      errors: cellAcc.errors,
      blocked: cellAcc.blocked,
    });
    if (cellAcc.requests > 0) {
      p50Weighted += cell.p50LatencyMs * cellAcc.requests;
      p95Weighted += cell.p95LatencyMs * cellAcc.requests;
      weight += cellAcc.requests;
    }
  }

  const totals = filtered
    ? toTotals(acc, ...latencyEstimate(acc))
    : toTotals(
        acc,
        weight > 0 ? p50Weighted / weight : 0,
        weight > 0 ? p95Weighted / weight : 0,
      );

  return { granularity: "day", range, points, totals };
}

function latencyEstimate(acc: Accum): [number, number] {
  const { p50, p95 } = latencyFromAcc(acc);
  return [p50, p95];
}

function buildHourlySeries(
  range: DateRange,
  filters: TrafficFilters,
  filtered: boolean,
): TrafficSeries {
  const cutoff = DEMO_NOW.getTime() - 24 * 3_600_000;
  const pool = filtered ? filterRequests(filters) : requests;

  const totals = new Map<
    number,
    { requests: number; inputTokens: number; outputTokens: number; cost: number; errors: number; blocked: number; ttft: number }
  >();

  for (const request of pool) {
    if (new Date(request.occurredAt).getTime() <= cutoff) continue;
    const existing = totals.get(request.hour) ?? {
      requests: 0,
      inputTokens: 0,
      outputTokens: 0,
      cost: 0,
      errors: 0,
      blocked: 0,
      ttft: 0,
    };
    existing.requests += 1;
    existing.inputTokens += request.inputTokens;
    existing.outputTokens += request.outputTokens;
    existing.cost += request.cost;
    existing.errors += request.status === "error" ? 1 : 0;
    existing.blocked += request.status === "blocked" ? 1 : 0;
    existing.ttft += request.ttftMs;
    totals.set(request.hour, existing);
  }

  const start = (DEMO_NOW.getUTCHours() - 23 + 24) % 24;
  const order = Array.from({ length: 24 }, (_, i) => (start + i) % 24);

  const points: SeriesPoint[] = order.map((hour) => {
    const bucket = totals.get(hour);
    return {
      label: `${String(hour).padStart(2, "0")}:00`,
      requests: bucket?.requests ?? 0,
      inputTokens: bucket?.inputTokens ?? 0,
      outputTokens: bucket?.outputTokens ?? 0,
      cost: round(bucket?.cost ?? 0, 4),
      errors: bucket?.errors ?? 0,
    };
  });

  const acc = emptyAccum();
  for (const hour of order) {
    const bucket = totals.get(hour);
    if (!bucket) continue;
    addAccum(acc, {
      requests: bucket.requests,
      inputTokens: bucket.inputTokens,
      outputTokens: bucket.outputTokens,
      cost: bucket.cost,
      errors: bucket.errors,
      blocked: bucket.blocked,
    });
    acc.ttftWeighted += bucket.ttft;
  }

  return {
    granularity: "hour",
    range,
    points,
    totals: toTotals(acc, ...latencyEstimate(acc)),
  };
}

export function buildSeries(
  range: DateRange,
  filters: TrafficFilters = EMPTY_FILTERS,
): TrafficSeries {
  const filtered = !filtersAreEmpty(filters);
  return range === "24h"
    ? buildHourlySeries(range, filters, filtered)
    : buildDailySeries(range, filters, filtered);
}

export function totalsFor(
  range: DateRange,
  filters: TrafficFilters = EMPTY_FILTERS,
): Totals {
  return buildSeries(range, filters).totals;
}

function describe(dim: DimensionKey, entryId: string): { name: string; detail: string } {
  switch (dim) {
    case "byPerson": {
      const person = personById.get(entryId);
      return {
        name: person?.name ?? entryId,
        detail: person?.title ?? "",
      };
    }
    case "byTeam": {
      const team = teamById.get(entryId);
      return {
        name: team?.name ?? entryId,
        detail: `${team?.memberIds.length ?? 0} members`,
      };
    }
    case "byProject": {
      const project = projectById.get(entryId);
      return {
        name: project?.name ?? entryId,
        detail: project?.environment ?? "",
      };
    }
    case "byModel": {
      const model = modelMap.get(entryId);
      return {
        name: model?.displayName ?? entryId,
        detail: model?.name ?? "",
      };
    }
    case "byProvider": {
      const provider = providerById.get(entryId);
      return {
        name: provider?.name ?? entryId,
        detail: provider?.status ?? "",
      };
    }
    case "byKind": {
      return {
        name: REQUEST_KIND_LABEL[entryId as RequestKind] ?? entryId,
        detail: "Request type",
      };
    }
    case "byEnvironment": {
      return {
        name: entryId,
        detail: "Environment",
      };
    }
  }
}

type Bump = (key: string) => Accum;

function aggregate24h(dim: DimensionKey, filters: TrafficFilters, bump: Bump): void {
  const cutoff = DEMO_NOW.getTime() - 24 * 3_600_000;
  for (const request of filterRequests(filters)) {
    if (new Date(request.occurredAt).getTime() <= cutoff) continue;
    addAccum(
      bump(request[COMBO_META_FIELD[dim]]),
      {
        requests: 1,
        inputTokens: request.inputTokens,
        outputTokens: request.outputTokens,
        cost: request.cost,
        errors: request.status === "error" ? 1 : 0,
        blocked: request.status === "blocked" ? 1 : 0,
      },
      request.ttftMs,
    );
  }
}

function aggregateBy(
  range: DateRange,
  filters: TrafficFilters,
  dim: DimensionKey,
  filtered: boolean,
): Map<string, Accum> {
  const result = new Map<string, Accum>();
  const bump = (key: string): Accum => {
    const existing = result.get(key);
    if (existing) return existing;
    const created = emptyAccum();
    result.set(key, created);
    return created;
  };

  if (range === "24h") {
    aggregate24h(dim, filters, bump);
    return result;
  }

  for (const cell of cellsForRange(range)) {
    if (!filtered) {
      const source = cell[dim] as Record<string, TrafficSlice>;
      for (const [key, slice] of Object.entries(source)) {
        const ttft =
          dim === "byModel"
            ? (modelMap.get(key)?.ttftMs ?? 0)
            : 900;
        addAccum(bump(key), slice, ttft);
      }
      continue;
    }
    const factor = statusFactor(filters);
    for (const [comboId, slice] of Object.entries(cell.byCombo)) {
      const passed = comboPasses(comboId, filters);
      if (!passed) continue;
      const key = passed.meta[COMBO_META_FIELD[dim]] as string;
      const ttft = modelMap.get(passed.meta.modelId)?.ttftMs ?? 0;
      addAccum(
        bump(key),
        {
          requests: slice.requests * factor,
          inputTokens: slice.inputTokens * factor,
          outputTokens: slice.outputTokens * factor,
          cost: slice.cost * factor,
          errors: slice.errors * factor,
          blocked: slice.blocked * factor,
        },
        ttft,
      );
    }
  }
  return result;
}

function toRollups(
  map: Map<string, Accum>,
  dim: DimensionKey,
  range: DateRange,
): DimensionRollup[] {
  const days = DATE_RANGE_DAYS[range];
  const totalCost = [...map.values()].reduce((sum, acc) => sum + acc.cost, 0);
  return [...map.entries()]
    .map(([entryId, acc]) => {
      const { name, detail } = describe(dim, entryId);
      const requests = Math.round(acc.requests);
      const errors = Math.round(acc.errors);
      const { p95 } = latencyFromAcc(acc);
      return {
        id: entryId,
        name,
        detail,
        requests,
        inputTokens: Math.round(acc.inputTokens),
        outputTokens: Math.round(acc.outputTokens),
        totalTokens: Math.round(acc.inputTokens + acc.outputTokens),
        cost: round(acc.cost, 4),
        errors,
        blocked: Math.round(acc.blocked),
        errorRate: requests > 0 ? round(errors / requests, 4) : 0,
        p95LatencyMs: Math.round(p95),
        share: totalCost > 0 ? round(acc.cost / totalCost, 4) : 0,
        perDay: round(requests / days, 1),
      };
    })
    .filter((rollup) => rollup.requests > 0)
    .sort((a, b) => b.cost - a.cost);
}

export function breakdownFor(
  range: DateRange,
  filters: TrafficFilters = EMPTY_FILTERS,
): Breakdown {
  const filtered = !filtersAreEmpty(filters);
  const build = (dim: DimensionKey): DimensionRollup[] =>
    toRollups(aggregateBy(range, filters, dim, filtered), dim, range);

  return {
    byPerson: build("byPerson"),
    byTeam: build("byTeam"),
    byProject: build("byProject"),
    byModel: build("byModel"),
    byProvider: build("byProvider"),
    byKind: build("byKind"),
    byEnvironment: build("byEnvironment"),
  };
}

export function rollupFor(
  dim: DimensionKey,
  entryId: string,
  range: DateRange,
  filters: TrafficFilters = EMPTY_FILTERS,
): DimensionRollup | null {
  const filtered = !filtersAreEmpty(filters);
  const map = aggregateBy(range, filters, dim, filtered);
  const acc = map.get(entryId);
  if (!acc) return null;
  return toRollups(map, dim, range).find((rollup) => rollup.id === entryId) ?? null;
}

export function filterRequests(
  filters: TrafficFilters,
  limit?: number,
): typeof requests {
  const needle = filters.search.trim().toLowerCase();
  const matched = requests.filter((request) => {
    if (filters.personIds.length > 0 && !filters.personIds.includes(request.personId))
      return false;
    if (filters.teamIds.length > 0 && !filters.teamIds.includes(request.teamId))
      return false;
    if (
      filters.projectIds.length > 0 &&
      !filters.projectIds.includes(request.projectId)
    )
      return false;
    if (filters.modelIds.length > 0 && !filters.modelIds.includes(request.modelId))
      return false;
    if (
      filters.providerIds.length > 0 &&
      !filters.providerIds.includes(request.providerId)
    )
      return false;
    if (filters.kinds.length > 0 && !filters.kinds.includes(request.kind)) return false;
    if (
      filters.environments.length > 0 &&
      !filters.environments.includes(request.environment)
    )
      return false;
    if (filters.statuses.length > 0 && !filters.statuses.includes(request.status))
      return false;
    if (needle.length > 0 && !request.summary.toLowerCase().includes(needle))
      return false;
    return true;
  });
  return typeof limit === "number" ? matched.slice(0, limit) : matched;
}

const SCOPE_RANK: Record<DemoModelGrant["scope"], number> = {
  organization: 1,
  team: 2,
  project: 3,
};

export function grantAdmission24h(
  grants: readonly DemoModelGrant[],
): Map<string, number> {
  const counts = new Map<string, number>(grants.map((grant) => [grant.id, 0]));
  const candidates = new Map<string, DemoModelGrant[]>();
  for (const grant of grants) {
    if (grant.status !== "active") continue;
    for (const modelId of grant.modelIds) {
      const existing = candidates.get(modelId);
      if (existing) existing.push(grant);
      else candidates.set(modelId, [grant]);
    }
  }
  const cutoff = DEMO_NOW.getTime() - 24 * 3_600_000;
  for (const request of requests) {
    if (request.status === "blocked") continue;
    if (new Date(request.occurredAt).getTime() <= cutoff) continue;
    let winner: DemoModelGrant | undefined;
    for (const grant of candidates.get(request.modelId) ?? []) {
      const covered =
        grant.scope === "organization" ||
        (grant.scope === "team" &&
          grant.principalIds.includes(request.teamId)) ||
        (grant.scope === "project" &&
          grant.principalIds.includes(request.projectId));
      if (!covered) continue;
      if (!winner || SCOPE_RANK[grant.scope] > SCOPE_RANK[winner.scope])
        winner = grant;
    }
    if (winner) counts.set(winner.id, (counts.get(winner.id) ?? 0) + 1);
  }
  return counts;
}

export interface Denial24h {
  id: string;
  principal: string;
  model: string;
  reason: string;
  blockedAt: string;
  requests: number;
}

const blockedByName = new Map<string, string>([
  ...guardrails.map((guardrail) => [guardrail.id, guardrail.name] as const),
  ...blockingPolicies.map((policy) => [policy.id, policy.name] as const),
]);

export function denials24h(): Denial24h[] {
  const cutoff = DEMO_NOW.getTime() - 24 * 3_600_000;
  const grouped = new Map<
    string,
    { personId: string; modelId: string; source: string; blockedAt: string; count: number }
  >();
  for (const request of requests) {
    if (request.status !== "blocked") continue;
    if (new Date(request.occurredAt).getTime() <= cutoff) continue;
    const source =
      request.blockedByPolicyId ?? request.blockedByGuardrailId ?? "unknown";
    const key = `${request.personId}|${request.modelId}|${source}`;
    const existing = grouped.get(key);
    if (existing) {
      existing.count += 1;
      if (request.occurredAt > existing.blockedAt)
        existing.blockedAt = request.occurredAt;
    } else {
      grouped.set(key, {
        personId: request.personId,
        modelId: request.modelId,
        source,
        blockedAt: request.occurredAt,
        count: 1,
      });
    }
  }
  return [...grouped.entries()]
    .map(([key, group]) => ({
      id: `den24_${key.replace(/[^a-z0-9]+/gi, "_")}`,
      principal: personById.get(group.personId)?.name ?? group.personId,
      model: modelMap.get(group.modelId)?.displayName ?? group.modelId,
      reason: blockedByName.get(group.source) ?? "Policy",
      blockedAt: group.blockedAt,
      requests: group.count,
    }))
    .sort((a, b) => b.requests - a.requests || a.id.localeCompare(b.id));
}
