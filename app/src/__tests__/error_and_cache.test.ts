import test from "node:test";
import assert from "node:assert/strict";

interface WidgetResult<T> {
  data?: T;
  error?: string;
  loading: boolean;
}

async function loadDashboardWidgets(
  fetchMetrics: () => Promise<{ requests: number }>,
  fetchAlerts: () => Promise<Array<{ id: string; severity: string }>>
): Promise<{
  metricsWidget: WidgetResult<{ requests: number }>;
  alertsWidget: WidgetResult<Array<{ id: string; severity: string }>>;
}> {
  const [metricsRes, alertsRes] = await Promise.allSettled([
    fetchMetrics(),
    fetchAlerts(),
  ]);

  return {
    metricsWidget:
      metricsRes.status === "fulfilled"
        ? { data: metricsRes.value, loading: false }
        : { error: (metricsRes.reason as Error).message, loading: false },
    alertsWidget:
      alertsRes.status === "fulfilled"
        ? { data: alertsRes.value, loading: false }
        : { error: (alertsRes.reason as Error).message, loading: false },
  };
}

test("Error Resilience: secondary widget failure does not break primary metrics", async () => {
  const mockFetchMetrics = async () => ({ requests: 120 });
  const mockFetchAlerts = async () => {
    throw new Error("Failed to load alerts: Service unavailable");
  };

  const dashboard = await loadDashboardWidgets(mockFetchMetrics, mockFetchAlerts);

  // Primary metrics loaded successfully
  assert.equal(dashboard.metricsWidget.error, undefined);
  assert.equal(dashboard.metricsWidget.data?.requests, 120);

  // Secondary alerts widget shows isolated error
  assert.equal(dashboard.alertsWidget.data, undefined);
  assert.ok(dashboard.alertsWidget.error?.includes("Service unavailable"));
});

class QueryCache {
  private cache = new Map<string, { data: unknown; timestamp: number }>();
  private ttlMs = 5000;

  get(key: string) {
    const entry = this.cache.get(key);
    if (!entry) return undefined;
    if (Date.now() - entry.timestamp > this.ttlMs) {
      this.cache.delete(key);
      return undefined;
    }
    return entry.data;
  }

  set(key: string, data: unknown) {
    this.cache.set(key, { data, timestamp: Date.now() });
  }

  invalidate(prefix: string) {
    for (const key of this.cache.keys()) {
      if (key.startsWith(prefix)) {
        this.cache.delete(key);
      }
    }
  }
}

test("Cache Invalidation: mutation clears related cached keys", () => {
  const cache = new QueryCache();
  cache.set("usage:summary", { requests: 10 });
  cache.set("usage:timeseries", [1, 2, 3]);
  cache.set("projects:list", ["proj1"]);

  assert.notEqual(cache.get("usage:summary"), undefined);
  assert.notEqual(cache.get("usage:timeseries"), undefined);
  assert.notEqual(cache.get("projects:list"), undefined);

  // Invalidate usage queries after request
  cache.invalidate("usage:");

  assert.equal(cache.get("usage:summary"), undefined);
  assert.equal(cache.get("usage:timeseries"), undefined);
  assert.notEqual(cache.get("projects:list"), undefined);
});
