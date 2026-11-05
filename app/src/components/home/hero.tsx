import Link from "next/link";
import { ArrowRight, Check } from "lucide-react";

import { FlowCurves } from "@/components/brand/flow-curves";
import { GatewayPrism } from "@/components/brand/gateway-prism";
import { GuillocheBand, GuillocheRosette } from "@/components/brand/guilloche";
import { LogoMark } from "@/components/logo";
import { Container } from "./section";

/* A sample request, printed check by check. Sample figures, labelled SPECIMEN. */
const LEDGER = [
  { check: "Key", result: "Verified", detail: "team Payments" },
  { check: "Policy", result: "Allowed", detail: "gpt-4o-mini" },
  { check: "Guardrails", result: "Redacted", detail: "1 email address" },
  { check: "Budget", result: "Within", detail: "$41.20 of $500" },
  { check: "Route", result: "Sent", detail: "OpenAI" },
  { check: "Record", result: "Sealed", detail: "1,284 tokens · 812 ms" },
] as const;

const MICROPRINT = "TOLLSEAL · EVERY AI REQUEST CHECKED AND SEALED · ".repeat(14);

export function HomeHero() {
  return (
    <section aria-labelledby="hero-heading" className="relative -mt-[68px] overflow-hidden pt-24 pb-14 sm:pt-28 lg:pb-20">
      <div aria-hidden className="ambient pointer-events-none absolute inset-0" />
      <FlowCurves />
      <Container className="relative">
        {/* The note: the hero is printed on it, edge to edge. */}
        {/* The note is not clipped, so the rosette can break its right edge. */}
        <div className="relative rounded-[20px] border border-border-strong bg-card shadow-(--shadow-dialog)">
          <p aria-hidden className="pointer-events-none truncate rounded-t-[20px] border-b border-border px-6 py-1.5 font-mono text-[0.5625rem] tracking-[0.18em] text-muted-foreground select-none">
            {MICROPRINT}
          </p>

          {/* Serial numbers sit in opposite corners, as on a note. */}
          <p aria-hidden className="pointer-events-none absolute top-9 left-3.5 font-mono text-[0.625rem] tracking-[0.1em] text-muted-foreground select-none sm:left-4">
            No. TS·0921·A7F3
          </p>

          <div className="relative grid grid-cols-1 gap-10 px-6 pt-16 pb-10 sm:px-10 sm:pt-20 lg:grid-cols-[minmax(0,1.12fr)_minmax(0,0.88fr)] lg:gap-6 lg:px-12 lg:pt-20">
            <div className="relative z-10 flex flex-col gap-7">
              <h1 id="hero-heading" className="font-display text-[clamp(2.6rem,6.2vw,5.25rem)] leading-[0.96] text-foreground">
                Every AI provider. <span className="text-primary">One gateway key.</span>
              </h1>
              <p className="max-w-[48ch] text-[1.0625rem] leading-relaxed text-muted-foreground">
                Connect OpenAI, Anthropic, Gemini, Azure OpenAI, Hugging Face and your own
                endpoints once. Your people and services get one Tollseal key and call every model
                you allow from the aigw CLI, the TypeScript SDK or REST. Each request is checked
                on the way through and sealed into one record of what it cost.
              </p>
              <div className="flex flex-wrap items-center gap-3">
                {/* The seal, printed beside the action that earns it */}
                <span aria-hidden className="relative grid size-12 shrink-0 place-items-center sm:size-14">
                  <GuillocheRosette className="absolute inset-0 size-full" />
                  <LogoMark className="relative size-6" />
                </span>
                <Link
                  href="/signup"
                  className="group inline-flex h-12 items-center gap-2 rounded-full bg-primary px-7 text-[0.9375rem] font-medium text-primary-foreground shadow-[inset_0_1px_0_rgba(255,255,255,0.16),0_8px_18px_-8px_rgba(13,92,72,0.6)] transition-[background-color,transform] duration-150 ease-(--ease-out-strong) hover:bg-primary-hover focus-visible:ring-3 focus-visible:ring-ring/40 focus-visible:outline-none active:scale-[0.97]"
                >
                  Get started
                  <ArrowRight className="size-4 transition-transform duration-200 ease-(--ease-out-strong) group-hover:translate-x-0.5" aria-hidden />
                </Link>
                <a
                  href="#workflow"
                  className="inline-flex h-12 items-center rounded-full border border-border-strong bg-card px-7 text-[0.9375rem] font-medium text-foreground transition-[background-color,transform] duration-150 ease-(--ease-out-strong) hover:bg-accent focus-visible:ring-3 focus-visible:ring-ring/40 focus-visible:outline-none active:scale-[0.97]"
                >
                  See how a request travels
                </a>
              </div>
            </div>

            {/* The rosette and the prism it frames, set at the note's right edge. */}
            <div className="relative mx-auto grid aspect-square w-full max-w-[420px] place-items-center lg:max-w-[560px] lg:translate-x-[22%] lg:self-center">
              <GuillocheRosette className="absolute inset-0 size-full" />
              <GatewayPrism size={150} className="relative" />
            </div>
          </div>

          {/* The printed request: each check stamps in, left to right. */}
          <div className="relative overflow-hidden rounded-b-[20px] border-t border-border bg-background/60">
            <GuillocheBand className="absolute inset-x-0 top-0 opacity-70" />
            <div className="relative flex flex-wrap items-center justify-between gap-3 px-6 pt-6 sm:px-10 lg:px-12">
              <p className="font-mono text-xs text-muted-foreground">
                Request <span className="text-foreground">req_7f3a·0921</span> · POST /v1/chat/completions
              </p>
              <span className="-rotate-3 rounded-[4px] border border-rose-600/50 px-2 py-0.5 font-mono text-[0.6875rem] tracking-[0.2em] text-rose-700 dark:border-rose-400/50 dark:text-rose-400">
                SPECIMEN
              </span>
            </div>
            <ol
              aria-label="A sample request passing the six checks"
              data-ledger
              className="relative grid grid-cols-2 gap-y-3 px-6 pt-4 pb-7 sm:grid-cols-3 sm:px-10 lg:grid-cols-6 lg:px-12"
            >
              {LEDGER.map((row) => (
                <li key={row.check} data-stamp className="flex min-w-0 flex-col gap-1 border-l border-border py-1 pr-2 pl-3">
                  <span className="font-mono text-[0.6875rem] tracking-[0.12em] text-muted-foreground uppercase">{row.check}</span>
                  <span className="flex items-center gap-1.5 text-sm font-semibold text-foreground">
                    <Check className="size-3.5 shrink-0 text-emerald-600 dark:text-emerald-400" aria-hidden />
                    {row.result}
                  </span>
                  <span className="truncate font-mono text-xs text-muted-foreground">{row.detail}</span>
                </li>
              ))}
            </ol>
            <p aria-hidden className="pointer-events-none absolute right-6 bottom-2 font-mono text-[0.625rem] tracking-[0.1em] text-muted-foreground select-none sm:right-10 lg:right-12">
              TS·0921·A7F3
            </p>
          </div>
        </div>
      </Container>
    </section>
  );
}
