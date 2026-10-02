import { cn } from "@/lib/utils";

const KPIS = [
  { label: "Requests 24h", value: "2,189" },
  { label: "Attributed spend", value: "$14.2k" },
  { label: "Blocked by policy", value: "22" },
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
    <figure className={cn("overflow-hidden rounded-xl border bg-card shadow-sm", className)}>
      <div className="flex items-center gap-2 border-b bg-muted/40 px-4 py-2.5">
        <span className="flex gap-1.5" aria-hidden>
          <span className="size-2.5 rounded-full bg-border-strong" />
          <span className="size-2.5 rounded-full bg-border-strong" />
          <span className="size-2.5 rounded-full bg-border-strong" />
        </span>
        <span className="ml-2 font-mono text-[11px] text-muted-foreground">
          sentinel / overview
        </span>
      </div>

      <div className="grid grid-cols-[88px_1fr] sm:grid-cols-[112px_1fr]">
        <nav className="hidden border-r bg-muted/20 p-2.5 sm:block" aria-label="Preview navigation">
          <ul className="space-y-0.5">
            {RAIL.map((item, index) => (
              <li key={item}>
                <span
                  className={cn(
                    "block truncate rounded-md px-2 py-1.5 text-[11px]",
                    index === 0
                      ? "bg-primary/10 font-medium text-primary"
                      : "text-muted-foreground",
                  )}
                >
                  {item}
                </span>
              </li>
            ))}
          </ul>
        </nav>

        <div className="space-y-3 p-3.5 sm:p-4">
          <div className="grid grid-cols-3 gap-2.5">
            {KPIS.map((kpi) => (
              <div key={kpi.label} className="rounded-lg border bg-background p-2.5">
                <p className="truncate text-[10px] text-muted-foreground">{kpi.label}</p>
                <p className="mt-0.5 text-sm font-semibold tabular-nums">{kpi.value}</p>
              </div>
            ))}
          </div>

          <div className="rounded-lg border bg-background p-3">
            <div className="flex items-center justify-between">
              <p className="text-[10px] text-muted-foreground">Request volume</p>
              <span className="text-[10px] text-muted-foreground">last 12 intervals</span>
            </div>
            <div className="mt-2.5 flex h-16 items-end gap-1" aria-hidden>
              {SERIES.map((height, index) => (
                <span
                  key={index}
                  style={{ height: `${height}%` }}
                  className={cn(
                    "flex-1 rounded-sm",
                    index === SERIES.length - 1 ? "bg-primary" : "bg-primary/25",
                  )}
                />
              ))}
            </div>
          </div>

          <ul className="space-y-1.5">
            {EVENTS.map((event) => (
              <li
                key={event.model}
                className="flex items-center justify-between gap-2 rounded-lg border bg-background px-2.5 py-2"
              >
                <span className="min-w-0">
                  <span className="block truncate font-mono text-[10px]">{event.model}</span>
                  <span className="block truncate text-[10px] text-muted-foreground">
                    {event.team}
                  </span>
                </span>
                <span
                  className={cn(
                    "shrink-0 rounded-full px-1.5 py-0.5 text-[9px] font-medium",
                    event.decision === "Blocked"
                      ? "bg-destructive/10 text-destructive"
                      : "bg-primary/10 text-primary",
                  )}
                >
                  {event.decision}
                </span>
              </li>
            ))}
          </ul>
        </div>
      </div>

      <figcaption className="border-t px-4 py-2.5 text-[11px] text-muted-foreground">
        Illustrative preview. Values are sample data from the demo environment.
      </figcaption>
    </figure>
  );
}
