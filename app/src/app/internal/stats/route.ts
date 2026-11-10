import { timingSafeEqual } from "node:crypto";
import { availableParallelism } from "node:os";

// Resource figures for the platform monitor, in the same shape the Go services
// report (backend/internal/platform/runtimestats). Guarded by MONITOR_TOKEN;
// without one the endpoint does not exist.
export const dynamic = "force-dynamic";
export const runtime = "nodejs";

function tokenMatches(presented: string, expected: string): boolean {
  const a = Buffer.from(presented);
  const b = Buffer.from(expected);
  return a.length === b.length && timingSafeEqual(a, b);
}

export function GET(request: Request) {
  const token = process.env.MONITOR_TOKEN ?? "";
  const presented = (request.headers.get("authorization") ?? "").replace(/^Bearer /, "");
  if (!token || !tokenMatches(presented, token)) {
    return new Response("Not found", { status: 404 });
  }

  const cpu = process.cpuUsage();
  const memory = process.memoryUsage();
  return Response.json(
    {
      uptime_seconds: process.uptime(),
      cpu_seconds: (cpu.user + cpu.system) / 1_000_000,
      num_cpu: availableParallelism(),
      rss_bytes: memory.rss,
      heap_bytes: memory.heapUsed,
    },
    { headers: { "Cache-Control": "no-store" } },
  );
}
