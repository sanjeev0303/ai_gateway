"use client";

import { useState } from "react";
import { Check, Minus, ShieldAlert, X } from "lucide-react";

import { cn } from "@ai-governance/console-ui/lib/utils";

type StageState = "pass" | "redact" | "stop" | "skip";

interface Scenario {
  id: string;
  label: string;
  team: string;
  model: string;
  prompt: string;
  /** One state per stage. Everything after a stop is never reached. */
  stages: StageState[];
  status: number;
  verdict: string;
  note: string;
}

const STAGES = ["Key", "Guardrails", "Policy", "Limits", "Route", "Record"];

// Each outcome mirrors what the gateway returned when tested with this request.
const SCENARIOS: Scenario[] = [
  {
    id: "allowed",
    label: "Everyday prompt",
    team: "Support",
    model: "gemini-2.5-flash",
    prompt: "Summarise this ticket in two sentences.",
    stages: ["pass", "pass", "pass", "pass", "pass", "pass"],
    status: 200,
    verdict: "Routed to Google Gemini",
    note: "Recorded with its tokens, cost and latency.",
  },
  {
    id: "redact",
    label: "Contains an email",
    team: "Support",
    model: "gemini-2.5-flash",
    prompt: "Reply to priya@example.com about the refund.",
    stages: ["pass", "redact", "pass", "pass", "pass", "pass"],
    status: 200,
    verdict: "Forwarded with the address masked",
    note: "The provider and the stored conversation see [REDACTED_EMAIL].",
  },
  {
    id: "pii",
    label: "Contains an SSN",
    team: "Finance",
    model: "gemini-2.5-flash",
    prompt: "Verify the customer SSN 123-45-6789.",
    stages: ["pass", "stop", "skip", "skip", "skip", "skip"],
    status: 403,
    verdict: "Blocked by a guardrail",
    note: "Nothing reached a provider. The block is logged as a guardrail event.",
  },
  {
    id: "policy",
    label: "Blocked model",
    team: "Engineering",
    model: "gemini-2.5-pro",
    prompt: "Refactor this module.",
    stages: ["pass", "pass", "stop", "skip", "skip", "skip"],
    status: 403,
    verdict: "Model blocked by policy",
    note: "The organization blocklist names this model.",
  },
  {
    id: "rate",
    label: "Over the rate limit",
    team: "Marketing",
    model: "gemini-2.5-flash",
    prompt: "Draft a fourth headline this minute.",
    stages: ["pass", "pass", "pass", "stop", "skip", "skip"],
    status: 429,
    verdict: "Rate limit reached",
    note: "Marketing may send three requests a minute.",
  },
];

const ICON: Record<StageState, React.ComponentType<{ className?: string }>> = {
  pass: Check,
  redact: ShieldAlert,
  stop: X,
  skip: Minus,
};

const NODE: Record<StageState, string> = {
  pass: "border-emerald-600/40 bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400",
  redact: "border-amber-600/50 bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300",
  stop: "border-rose-600/50 bg-rose-50 text-rose-700 dark:bg-rose-950 dark:text-rose-400",
  skip: "border-border bg-muted text-muted-foreground",
};

const STATE_LABEL: Record<StageState, string> = {
  pass: "passed",
  redact: "masked personal data",
  stop: "stopped here",
  skip: "not reached",
};

/** Pick a request and follow the path the gateway takes with it. */
export function RequestInspector() {
  const [active, setActive] = useState(SCENARIOS[0]!.id);
  const scenario = SCENARIOS.find((s) => s.id === active) ?? SCENARIOS[0]!;
  const ok = scenario.status < 400;

  // The path fills up to the stage where the request stops, or all the way.
  const stopAt = scenario.stages.indexOf("stop");
  const reached = stopAt === -1 ? STAGES.length - 1 : stopAt;
  const fill = `${(reached / (STAGES.length - 1)) * 100}%`;

  return (
    <div className="rounded-[22px] border border-border bg-popover p-5 shadow-(--shadow-dialog) sm:p-6">
      <div className="mb-4 flex flex-col gap-0.5">
        <h2 className="font-sans text-base font-semibold tracking-[-0.01em]">Try a request</h2>
        <p className="text-sm text-muted-foreground">Pick one and follow what the gateway does with it.</p>
      </div>

      <div role="group" aria-label="Sample requests" className="mb-4 flex flex-wrap gap-1.5">
        {SCENARIOS.map((s) => (
          <button
            key={s.id}
            type="button"
            aria-pressed={s.id === active}
            onClick={() => setActive(s.id)}
            className={cn(
              "h-9 rounded-full border px-3.5 text-[0.8125rem] font-medium transition-[background-color,border-color,color,transform] duration-150 active:scale-[0.97]",
              "focus-visible:ring-3 focus-visible:ring-ring/30 focus-visible:outline-none",
              s.id === active
                ? "border-primary bg-primary text-primary-foreground"
                : "border-border bg-card text-muted-foreground hover:border-border-strong hover:text-foreground",
            )}
          >
            {s.label}
          </button>
        ))}
      </div>

      <div className="mb-6 rounded-xl bg-muted/60 px-4 py-3">
        <p className="mb-1 flex flex-wrap gap-x-3 text-xs text-muted-foreground">
          <span>Team <span className="font-medium text-foreground">{scenario.team}</span></span>
          <span>Model <span className="font-mono text-foreground">{scenario.model}</span></span>
        </p>
        <p className="text-[0.9375rem] text-foreground">&ldquo;{scenario.prompt}&rdquo;</p>
      </div>

      {/* The request's path, left to right */}
      <div className="relative mb-6">
        <div aria-hidden className="absolute top-[15px] right-[8.33%] left-[8.33%] h-0.5 rounded-full bg-border">
          <div
            className={cn(
              "h-full rounded-full transition-[width,background-color] duration-500 ease-(--ease-out-strong)",
              ok ? "bg-emerald-600 dark:bg-emerald-500" : "bg-rose-600 dark:bg-rose-500",
            )}
            style={{ width: fill }}
          />
        </div>
        <ol className="relative grid grid-cols-6">
          {STAGES.map((stage, i) => {
            const state = scenario.stages[i]!;
            const Icon = ICON[state];
            return (
              <li key={stage} className="flex min-w-0 flex-col items-center gap-1.5 text-center">
                <span
                  style={{ transitionDelay: `${i * 40}ms` }}
                  className={cn(
                    "flex size-8 items-center justify-center rounded-full border transition-[background-color,border-color,color] duration-200",
                    NODE[state],
                  )}
                >
                  <Icon className="size-3.5" />
                </span>
                <span className={cn("max-w-full text-[0.6875rem] font-medium sm:text-xs", state === "skip" && "text-muted-foreground")}>
                  {stage}
                </span>
                <span className="sr-only">: {STATE_LABEL[state]}</span>
              </li>
            );
          })}
        </ol>
      </div>

      <p aria-live="polite" className="flex min-h-14 flex-wrap items-baseline gap-x-2 gap-y-1 text-sm">
        <span
          className={cn(
            "rounded-full px-2 py-0.5 text-xs font-semibold tabular-nums",
            ok ? "bg-emerald-500/12 text-emerald-700 dark:text-emerald-400" : "bg-rose-500/12 text-rose-700 dark:text-rose-400",
          )}
        >
          {scenario.status}
        </span>
        <span className="font-medium">{scenario.verdict}.</span>
        <span className="text-muted-foreground">{scenario.note}</span>
      </p>
    </div>
  );
}
