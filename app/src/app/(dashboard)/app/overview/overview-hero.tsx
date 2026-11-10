import Link from "next/link";
import { Activity, ArrowRight, CheckCircle2, KeyRound, Route, ShieldCheck } from "lucide-react";

import { GatewayPrism } from "@/components/brand/gateway-prism";
import { GuillocheRosette } from "@ai-governance/console-ui/brand/guilloche";
import { Button } from "@ai-governance/console-ui/ui/button";
import { cn } from "@ai-governance/console-ui/lib/utils";
import { HeroToggle } from "./hero-toggle";

export const HERO_COOKIE = "sentinel_overview_hero";

interface StageChipProps {
  href: string;
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  detail: string;
  /** Which side of the prism it sits on; leave out when there is no prism. */
  side?: "left" | "right";
}

/** One stage of the request path, joined to the prism by a short copper lead. */
function StageChip({ href, icon: Icon, title, detail, side }: StageChipProps) {
  return (
    <Link
      href={href}
      className={cn(
        "group relative flex min-w-0 items-center gap-2.5 rounded-2xl border border-border bg-card/85 py-2 pr-3.5 pl-2 shadow-[0_1px_2px_rgba(20,26,51,0.05)] backdrop-blur-sm",
        "transition-[border-color,box-shadow,transform] duration-200 ease-(--ease-out-strong) hover:border-border-strong hover:shadow-(--shadow-float) active:scale-[0.98]",
        "focus-visible:ring-3 focus-visible:ring-ring/30 focus-visible:outline-none",
        side && "after:absolute after:top-1/2 after:h-px after:w-4 after:bg-linear-to-r after:from-copper/0 after:to-copper/60 max-xl:after:hidden",
        side === "left" && "after:-right-4",
        side === "right" && "after:-left-4 after:rotate-180",
      )}
    >
      <span className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-accent text-copper">
        <Icon className="size-4" />
      </span>
      <span className="flex min-w-0 flex-col">
        <span className="truncate text-xs font-semibold text-foreground">{title}</span>
        <span className="truncate text-xs text-muted-foreground tabular-nums">{detail}</span>
      </span>
    </Link>
  );
}

interface SetupItem {
  title: string;
  href: string;
  cta: string;
  done: boolean;
}

interface OverviewHeroProps {
  /** The stored choice: "collapsed", "expanded", or nothing yet. */
  stored?: string;
  providerCount: number;
  accessRuleCount: number;
  keyCount: number;
  requestCount: number;
  setup: SetupItem[];
}

function plural(n: number, one: string, many: string) {
  return `${n.toLocaleString("en-US")} ${n === 1 ? one : many}`;
}

export function OverviewHero({ stored, providerCount, accessRuleCount, keyCount, requestCount, setup }: OverviewHeroProps) {
  const done = setup.filter((s) => s.done).length;
  const remaining = setup.filter((s) => !s.done);

  // The large introduction is opt-in: the compact band shows by default, and the
  // setup checklist below still appears until every step is done.
  const collapsed = stored !== "expanded";

  const stages = {
    policy: { href: "/app/govern/model-access", icon: ShieldCheck, title: "Policy enforced", detail: plural(accessRuleCount, "access rule", "access rules") },
    keys: { href: "/app/platform/gateway-credentials", icon: KeyRound, title: "Keys issued", detail: plural(keyCount, "active key", "active keys") },
    route: { href: "/app/platform/provider-accounts", icon: Route, title: "Model routed", detail: plural(providerCount, "provider", "providers") },
    usage: { href: "/app/observe/usage", icon: Activity, title: "Usage recorded", detail: plural(requestCount, "request", "requests") },
  };

  const band = (
    <div className="flex items-center gap-2 rounded-2xl border border-border bg-card/70 p-2.5">
      <div className="grid min-w-0 flex-1 grid-cols-2 gap-2 lg:grid-cols-4">
        <StageChip {...stages.policy} />
        <StageChip {...stages.keys} />
        <StageChip {...stages.route} />
        <StageChip {...stages.usage} />
      </div>
      <div className="hidden md:block">
        <HeroToggle cookieName={HERO_COOKIE} collapsed />
      </div>
    </div>
  );

  return (
    <section aria-label="Your gateway at a glance" className="flex flex-col gap-4">
      {/* The compact default shows no band: its counts repeat Health and the setup checklist. */}
      {collapsed ? null : (
        <>
          <div className="md:hidden">{band}</div>
          <div className="relative hidden overflow-hidden rounded-[20px] border border-border-strong bg-card md:block">
            <div aria-hidden className="ambient pointer-events-none absolute inset-0" />
            <div className="absolute top-3 right-3 z-10">
              <HeroToggle cookieName={HERO_COOKIE} collapsed={false} />
            </div>
            <div className="relative grid items-center gap-8 p-9 xl:grid-cols-[minmax(0,1fr)_auto]">
              <div className="flex max-w-xl flex-col gap-5">
                <h2 className="font-display text-[clamp(1.875rem,2.6vw,2.625rem)] leading-[1.02] text-foreground">
                  Every request, <span className="text-primary">checked and sealed.</span>
                </h2>
                <p className="max-w-[46ch] text-[0.9375rem] leading-relaxed text-muted-foreground">
                  Every model request passes one gateway: checked against your guardrails and
                  policies, routed to the right provider, and recorded with its tokens and cost.
                </p>
                <div className="flex flex-wrap items-center gap-3">
                  <Button asChild size="lg">
                    <Link href="/app/govern/policies">
                      Manage policies
                      <ArrowRight data-icon="inline-end" />
                    </Link>
                  </Button>
                  <Button asChild size="lg" variant="outline">
                    <Link href="/app/observe/usage">View usage</Link>
                  </Button>
                </div>
              </div>

              <div className="grid grid-cols-2 items-center gap-3 xl:grid-cols-[auto_auto_auto] xl:gap-4">
                <div className="flex flex-col gap-3 xl:gap-12">
                  <StageChip side="left" {...stages.policy} />
                  <StageChip side="left" {...stages.keys} />
                </div>
                <div className="relative mx-auto hidden size-[248px] place-items-center xl:grid">
                  <GuillocheRosette className="absolute inset-0 size-full" />
                  <GatewayPrism size={112} className="relative" />
                </div>
                <div className="flex flex-col gap-3 xl:gap-12">
                  <StageChip side="right" {...stages.route} />
                  <StageChip side="right" {...stages.usage} />
                </div>
              </div>
            </div>
          </div>
        </>
      )}

      {remaining.length > 0 && (
        <div className="flex flex-col gap-4 rounded-2xl border border-border bg-card p-5 lg:flex-row lg:items-center lg:gap-6">
          <div className="flex shrink-0 flex-col gap-2">
            <h2 id="setup-heading" className="text-[0.9375rem] font-semibold text-foreground">
              Finish setting up
            </h2>
            <div className="flex items-center gap-3">
              <div
                className="h-1.5 w-40 overflow-hidden rounded-full bg-muted"
                role="progressbar"
                aria-valuenow={done}
                aria-valuemin={0}
                aria-valuemax={setup.length}
                aria-label="Setup progress"
              >
                <div
                  className="h-full rounded-full bg-copper transition-[width] duration-500 ease-(--ease-out-strong)"
                  style={{ width: `${(done / setup.length) * 100}%` }}
                />
              </div>
              <span className="text-xs text-muted-foreground tabular-nums">
                {done} of {setup.length} done
              </span>
            </div>
          </div>
          <ol className="grid flex-1 grid-cols-1 gap-2 sm:grid-cols-3 2xl:grid-cols-5">
            {setup.map((item) => (
              <li key={item.title}>
                <Link
                  href={item.href}
                  className={cn(
                    "flex h-full items-center gap-2.5 rounded-xl border px-3 py-2.5 text-[0.8125rem] transition-[border-color,background-color] duration-150 focus-visible:ring-3 focus-visible:ring-ring/30 focus-visible:outline-none",
                    item.done
                      ? "border-transparent bg-muted/50 text-muted-foreground hover:bg-muted"
                      : "border-border-strong bg-card font-medium text-foreground hover:border-copper/60",
                  )}
                >
                  {item.done ? (
                    <CheckCircle2 className="size-4 shrink-0 text-emerald-600 dark:text-emerald-400" aria-hidden />
                  ) : (
                    <ArrowRight className="size-4 shrink-0 text-copper" aria-hidden />
                  )}
                  <span className="truncate">{item.done ? item.title : item.cta}</span>
                  {item.done && <span className="sr-only">(done)</span>}
                </Link>
              </li>
            ))}
          </ol>
        </div>
      )}
    </section>
  );
}
