import Link from "next/link";
import { ArrowRight } from "lucide-react";

import { DashboardPreview } from "@/components/landing/dashboard-preview";
import { Container } from "@/components/landing/section";
import { Button } from "@/components/ui/button";
import { HERO } from "@/lib/landing-data";

export function Hero() {
  return (
    <section className="border-b bg-gradient-to-b from-muted/50 to-background">
      <Container className="grid gap-12 py-20 sm:py-24 lg:grid-cols-12 lg:items-center lg:gap-10 lg:py-28">
        <div className="lg:col-span-6">
          <span className="inline-flex items-center gap-2 rounded-full border bg-background px-3 py-1 text-xs font-medium">
            <span className="size-1.5 rounded-full bg-primary motion-safe:animate-pulse" aria-hidden />
            {HERO.eyebrow}
          </span>

          <h1 className="mt-6 text-4xl font-semibold tracking-tight text-balance sm:text-5xl lg:text-6xl">
            {HERO.headline}
          </h1>

          <p className="mt-6 max-w-xl text-lg leading-relaxed text-muted-foreground text-pretty">
            {HERO.subhead}
          </p>

          <div className="mt-8 flex flex-wrap items-center gap-3">
            <Button asChild size="lg" className="h-11 px-5">
              <Link href={HERO.primary.href}>
                {HERO.primary.label}
                <ArrowRight data-icon="inline-end" aria-hidden />
              </Link>
            </Button>
            <Button asChild size="lg" variant="outline" className="h-11 px-5">
              <Link href={HERO.secondary.href}>{HERO.secondary.label}</Link>
            </Button>
          </div>

          <p className="mt-4 text-sm text-muted-foreground">{HERO.microcopy}</p>

          <dl className="mt-10 grid grid-cols-1 gap-6 border-t pt-8 sm:grid-cols-3">
            {HERO.stats.map((stat) => (
              <div key={stat.label}>
                <dt className="text-xs text-muted-foreground">{stat.label}</dt>
                <dd className="mt-1 text-2xl font-semibold tracking-tight tabular-nums">
                  {stat.value}
                </dd>
              </div>
            ))}
          </dl>
        </div>

        <div className="lg:col-span-6">
          <DashboardPreview />
        </div>
      </Container>
    </section>
  );
}
