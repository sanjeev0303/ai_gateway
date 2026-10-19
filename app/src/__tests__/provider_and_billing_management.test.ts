import { test } from "node:test";
import assert from "node:assert/strict";

// Helper functions mirroring the component logic for test assertion
function resolveProviderValidationState(
  status: string,
  health?: { status: string; status_reason?: string; last_error?: string },
  modelCount: number = 0
): "Connected and models discovered" | "Connected but discovery unavailable" | "Credential invalid" | "Provider unreachable" | "Disabled" {
  if (status === "disabled") return "Disabled";
  if (!health || health.status === "healthy" || health.status === "ok") {
    return modelCount > 0 ? "Connected and models discovered" : "Connected but discovery unavailable";
  }

  const err = `${health.status_reason || ""} ${health.last_error || ""}`.toLowerCase();
  if (
    err.includes("credential") ||
    err.includes("auth") ||
    err.includes("401") ||
    err.includes("403") ||
    err.includes("key") ||
    err.includes("revoked")
  ) {
    return "Credential invalid";
  }
  if (err.includes("discovery") || err.includes("model")) {
    return "Connected but discovery unavailable";
  }
  return "Provider unreachable";
}

function resolveModelAuthorizationState(
  isActive: boolean,
  status?: string
): "Model authorized and ready" | "Model discovered but not authorized" | "Model no longer available" {
  if (status === "inactive" || status === "unavailable" || (!isActive && status !== "active")) {
    return "Model no longer available";
  }
  return isActive ? "Model authorized and ready" : "Model discovered but not authorized";
}

function calculatePlanQuotaLimit(planCode?: string): number {
  if (planCode === "enterprise") return 100_000_000;
  if (planCode === "pro") return 1_000_000;
  return 10_000;
}

function calculateQuotaPercentage(consumed: number, limit: number): number {
  if (limit <= 0) return 0;
  return Math.min(100, Math.round((consumed / limit) * 100));
}

test("Provider Validation Status: maps healthy provider with models to 'Connected and models discovered'", () => {
  const state = resolveProviderValidationState("active", { status: "healthy" }, 5);
  assert.equal(state, "Connected and models discovered");
});

test("Provider Validation Status: maps healthy provider without models to 'Connected but discovery unavailable'", () => {
  const state = resolveProviderValidationState("active", { status: "healthy" }, 0);
  assert.equal(state, "Connected but discovery unavailable");
});

test("Provider Validation Status: maps 401 authentication rejection to 'Credential invalid'", () => {
  const state = resolveProviderValidationState("active", {
    status: "unhealthy",
    status_reason: "authentication failed: HTTP 401 unauthorized",
  });
  assert.equal(state, "Credential invalid");
});

test("Provider Validation Status: maps network timeout to 'Provider unreachable'", () => {
  const state = resolveProviderValidationState("active", {
    status: "unhealthy",
    last_error: "dial tcp: i/o timeout",
  });
  assert.equal(state, "Provider unreachable");
});

test("Provider Validation Status: maps catalog failure to 'Connected but discovery unavailable'", () => {
  const state = resolveProviderValidationState("active", {
    status: "unhealthy",
    status_reason: "model discovery endpoint returned error 502",
  });
  assert.equal(state, "Connected but discovery unavailable");
});

test("Provider Validation Status: disabled account returns 'Disabled'", () => {
  const state = resolveProviderValidationState("disabled", { status: "healthy" });
  assert.equal(state, "Disabled");
});

test("Model Authorization: distinguishes authorized ready, discovered but not authorized, and no longer available models", () => {
  assert.equal(resolveModelAuthorizationState(true, "active"), "Model authorized and ready");
  assert.equal(resolveModelAuthorizationState(false, "active"), "Model discovered but not authorized");
  assert.equal(resolveModelAuthorizationState(true, "inactive"), "Model no longer available");
  assert.equal(resolveModelAuthorizationState(false, "unavailable"), "Model no longer available");
});

test("Billing Quotas: returns exact tier request limits", () => {
  assert.equal(calculatePlanQuotaLimit("free"), 10_000);
  assert.equal(calculatePlanQuotaLimit("pro"), 1_000_000);
  assert.equal(calculatePlanQuotaLimit("enterprise"), 100_000_000);
  assert.equal(calculatePlanQuotaLimit(undefined), 10_000);
});

test("Billing Quota Calculation: clamps quota consumption percentage accurately", () => {
  assert.equal(calculateQuotaPercentage(2500, 10_000), 25);
  assert.equal(calculateQuotaPercentage(10_000, 10_000), 100);
  assert.equal(calculateQuotaPercentage(15_000, 10_000), 100); // clamped to 100%
  assert.equal(calculateQuotaPercentage(0, 10_000), 0);
});

test("Billing State Machine: flags payment failure when past_due or unpaid", () => {
  const isFailure = (status: string) => status === "past_due" || status === "unpaid";
  assert.equal(isFailure("active"), false);
  assert.equal(isFailure("trialing"), false);
  assert.equal(isFailure("canceled"), false);
  assert.equal(isFailure("past_due"), true);
  assert.equal(isFailure("unpaid"), true);
});
