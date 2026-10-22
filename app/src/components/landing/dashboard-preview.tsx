import { cn } from "@/lib/utils";

const KPIS = [
  { label: "Requests 24h", value: "12,446" },
  { label: "Attributed spend", value: "$98,420" },
  { label: "Blocked by policy", value: "1,240" },
] as const;

const SERIES = [38, 52, 44, 61, 49, 72, 58, 66, 81, 63, 77, 92] as const;

const RAIL = ["Overview", "Live activity", "Costs", "Models", "Policies", "Budgets"] as const;

const EVENTS = [
  { model: "claude-opus-4", team: "Customer Experience", decision: "Allowed" },
  { model: "gpt-4o", team: "Commerce", decision: "Allowed" },
  { model: "gemini-1.5-pro", team: "Platform", decision: "Blocked" },
] as const;

/** Illustrative product visual. Values are static sample data, not live metrics. */
export function DashboardPreview({ className }: { className?: string }) {
  return (
    <figure
      className={cn(
        "relative overflow-hidden border border-[rgba(111,78,55,0.24)]",
        className,
      )}
      style={{ background: "rgba(245,243,240,0.95)" }}
    >
      {/* Top bar */}
      <div
        className="flex items-center justify-between px-4 py-2.5 border-b border-[rgba(111,78,55,0.10)]"
        style={{ background: "rgba(229,228,226,0.9)" }}
      >
        <div className="flex items-center gap-3">
          <div className="flex gap-1.5" aria-hidden>
            <span className="size-2.5 rounded-full bg-[#B54B2A]/50" />
            <span className="size-2.5 rounded-full bg-[#A8924A]/50" />
            <span className="size-2.5 rounded-full bg-[#4A7A5A]/50" />
          </div>
          <span
            className="ml-2 text-[11px] text-[#7A6458]"
            style={{ fontFamily: "var(--font-fira-code)" }}
          >
            sentinel / overview
          </span>
        </div>
        <span
          className="inline-flex items-center gap-1.5 border border-[rgba(74,122,90,0.25)] px-2 py-0.5 text-[10px] font-medium"
          style={{ color: "#4A7A5A", fontFamily: "var(--font-fira-code)", background: "rgba(74,122,90,0.06)" }}
        >
          <span className="size-1 rounded-full bg-[#4A7A5A]" aria-hidden />
          Gateway Live
        </span>
      </div>

      {/* Body: sidebar + content */}
      <div className="grid grid-cols-[88px_1fr] sm:grid-cols-[112px_1fr]">
        {/* Sidebar */}
        <nav
          className="hidden border-r border-[rgba(111,78,55,0.10)] p-2.5 sm:block"
          aria-label="Preview navigation"
          style={{ background: "rgba(229,228,226,0.5)" }}
        >
          <ul className="space-y-0.5">
            {RAIL.map((item, index) => (
              <li key={item}>
                <span
                  className={cn(
                    "block truncate px-2 py-1.5 text-[11px] transition-colors",
                    index === 0
                      ? "font-semibold text-[#6F4E37] border-l-2 border-[#6F4E37] pl-1.5"
                      : "text-[#7A6458]"
                  )}
                >
                  {item}
                </span>
              </li>
            ))}
          </ul>
        </nav>

        {/* Content */}
        <div className="space-y-3 p-3.5 sm:p-4">
          {/* KPI row */}
          <div className="grid grid-cols-3 gap-2.5">
            {KPIS.map((kpi) => (
              <div
                key={kpi.label}
                className="border border-[rgba(111,78,55,0.12)] p-2.5"
                style={{ background: "rgba(245,243,240,0.8)" }}
              >
                <p className="truncate text-[10px] text-[#7A6458]">{kpi.label}</p>
                <p
                  className="mt-0.5 text-sm font-semibold tabular-nums text-[#1C1412]"
                  style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.03em", fontSize: "1rem" }}
                >
                  {kpi.value}
                </p>
              </div>
            ))}
          </div>

          {/* Bar chart */}
          <div
            className="border border-[rgba(111,78,55,0.12)] p-3"
            style={{ background: "rgba(245,243,240,0.8)" }}
          >
            <div className="flex items-center justify-between">
              <p className="text-[10px] font-medium text-[#7A6458]">Request volume</p>
              <span className="text-[10px] text-[#7A6458]">last 12 intervals</span>
            </div>
            <div className="mt-2.5 flex h-16 items-end gap-1" aria-hidden>
              {SERIES.map((height, index) => (
                <span
                  key={index}
                  style={{ height: `${height}%` }}
                  className={cn(
                    "flex-1",
                    index === SERIES.length - 1
                      ? "bg-[#6F4E37]"
                      : "bg-[rgba(111,78,55,0.22)]"
                  )}
                />
              ))}
            </div>
          </div>

          {/* Events */}
          <ul className="space-y-1.5">
            {EVENTS.map((event) => (
              <li
                key={event.model}
                className="flex items-center justify-between gap-2 border border-[rgba(111,78,55,0.10)] px-2.5 py-2"
                style={{ background: "rgba(245,243,240,0.8)" }}
              >
                <span className="min-w-0">
                  <span
                    className="block truncate text-[10px] font-medium text-[#1C1412]"
                    style={{ fontFamily: "var(--font-fira-code)" }}
                  >
                    {event.model}
                  </span>
                  <span className="block truncate text-[10px] text-[#7A6458]">{event.team}</span>
                </span>
                <span
                  className="shrink-0 px-2 py-0.5 text-[9px] font-semibold border"
                  style={{
                    borderColor: event.decision === "Blocked" ? "rgba(181,75,42,0.25)" : "rgba(74,122,90,0.25)",
                    color: event.decision === "Blocked" ? "#B54B2A" : "#4A7A5A",
                    background: event.decision === "Blocked" ? "rgba(181,75,42,0.06)" : "rgba(74,122,90,0.06)",
                    fontFamily: "var(--font-fira-code)",
                  }}
                >
                  {event.decision}
                </span>
              </li>
            ))}
          </ul>
        </div>
      </div>

      <figcaption
        className="border-t border-[rgba(111,78,55,0.10)] px-4 py-2 text-[10px] text-[#7A6458]"
        style={{ fontFamily: "var(--font-fira-code)" }}
      >
        Illustrative preview. Real-time control plane demonstration.
      </figcaption>
    </figure>
  );
}
