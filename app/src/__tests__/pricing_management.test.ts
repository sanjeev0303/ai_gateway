import { test } from "node:test";
import assert from "node:assert/strict";

function formatMicrosRate(micros: number): string {
  if (micros <= 0) return "$0.00 / 1M";
  const perMillion = (micros / 1_000_000) * 1_000_000;
  return `$${perMillion.toFixed(2)} / 1M`;
}

function calculateCost(
  inputMicros: number,
  outputMicros: number,
  cacheMicros: number,
  promptTokens: number,
  completionTokens: number,
  cachedTokens: number = 0
): number {
  if (promptTokens < 0 || completionTokens < 0 || cachedTokens < 0) {
    throw new Error("Invalid token counts");
  }
  const regularPrompt = Math.max(0, promptTokens - cachedTokens);
  return (
    regularPrompt * inputMicros +
    completionTokens * outputMicros +
    cachedTokens * cacheMicros
  );
}

function modelSupportsCachedTokens(providerKind: string, modelSlug: string): boolean {
  if (providerKind === "anthropic" || providerKind === "gemini") {
    return true;
  }
  if (providerKind === "openai" && modelSlug.includes("gpt-4o")) {
    return true;
  }
  return false;
}

test("Pricing Formatting: converts micro-dollars per token to dollar rate per million tokens", () => {
  assert.equal(formatMicrosRate(2.5), "$2.50 / 1M");
  assert.equal(formatMicrosRate(10), "$10.00 / 1M");
  assert.equal(formatMicrosRate(0), "$0.00 / 1M");
  assert.equal(formatMicrosRate(-1), "$0.00 / 1M");
});

test("Cost Calculation: accurately computes prompt, completion, and cached token micro-dollars", () => {
  // Input: 3 µ$/token, Output: 15 µ$/token, Cached: 1.5 µ$/token
  // 1,000 prompt tokens (with 400 cached), 500 completion tokens
  // Cost = (600 * 3) + (500 * 15) + (400 * 1.5) = 1800 + 7500 + 600 = 9900 µ$
  const cost = calculateCost(3, 15, 1.5, 1000, 500, 400);
  assert.equal(cost, 9900);
});

test("Cost Calculation: rejects negative token inputs", () => {
  assert.throws(
    () => calculateCost(3, 15, 0, -100, 500, 0),
    /Invalid token counts/
  );
});

test("Cached Token Eligibility: accurately identifies models supporting cache-read pricing", () => {
  assert.equal(modelSupportsCachedTokens("anthropic", "claude-3-5-sonnet"), true);
  assert.equal(modelSupportsCachedTokens("gemini", "gemini-1.5-pro"), true);
  assert.equal(modelSupportsCachedTokens("openai", "gpt-4o"), true);
  assert.equal(modelSupportsCachedTokens("openai", "gpt-3.5-turbo"), false);
  assert.equal(modelSupportsCachedTokens("huggingface", "meta-llama/Llama-3-8B"), false);
});
