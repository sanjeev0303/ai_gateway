/**
 * Deterministic pseudo-randomness.
 *
 * Every number in the demo dataset is generated from a fixed seed, so the
 * product renders byte-identically on the server and the client and across
 * reloads. Nothing here may use Math.random() or Date.now().
 */

/** mulberry32 — small, fast, good enough distribution for fixtures. */
export function rng(seed: number): () => number {
  let a = seed >>> 0;
  return () => {
    a = (a + 0x6d2b79f5) >>> 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

/** Stable 32-bit hash, used to derive a sub-seed from a string. */
export function hash(seed: string): number {
  let h = 2166136261;
  for (let i = 0; i < seed.length; i += 1) {
    h ^= seed.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return h >>> 0;
}

/** Integer in [min, max]. */
export function randInt(r: () => number, min: number, max: number): number {
  return min + Math.floor(r() * (max - min + 1));
}

/** Float in [min, max). */
export function randFloat(r: () => number, min: number, max: number): number {
  return min + r() * (max - min);
}

export interface Weighted<T> {
  item: T;
  weight: number;
}

/**
 * Pick one entry proportionally to its weight. Weights need not be normalised
 * and non-positive weights are skipped, so callers can pass raw ratios.
 */
export function weightedPick<T>(r: () => number, entries: readonly Weighted<T>[]): T {
  let total = 0;
  for (const entry of entries) if (entry.weight > 0) total += entry.weight;
  if (total <= 0) return entries[0]!.item;
  let roll = r() * total;
  for (const entry of entries) {
    if (entry.weight <= 0) continue;
    roll -= entry.weight;
    if (roll <= 0) return entry.item;
  }
  return entries[entries.length - 1]!.item;
}

/** Zipf-like rank weighting: index 0 gets `skew`x the weight of index 1. */
export function rankWeights(count: number, skew = 1): number[] {
  return Array.from({ length: count }, (_, i) => 1 / Math.pow(i + 1, skew));
}

/** Round to `places` decimals — keeps fixture values free of float noise. */
export function round(value: number, places = 2): number {
  const factor = 10 ** places;
  return Math.round(value * factor) / factor;
}

/** `req_a1b2c3` style ids, deterministic from a counter. */
export function id(prefix: string, index: number, width = 6): string {
  const body = Math.abs(index).toString(36).padStart(width, "0");
  return `${prefix}_${body}`;
}
