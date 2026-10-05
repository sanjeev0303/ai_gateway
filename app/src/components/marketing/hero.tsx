import Link from "next/link";
import { ArrowRight, Check, ShieldCheck } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { APP_DESCRIPTION, APP_NAME, APP_TAGLINE } from "@/lib/constants";

interface HeroProps {
  /** Marketing pages pass their own headline; the homepage uses the tagline. */
  headline?: string;
  subheadline?: string;
}

const CAPABILITIES = [
  {
    title: "One request shape, provider-native out",
    detail: "Anthropic messages in; auth and version headers are rewritten for Anthropic and OpenAI upstreams.",
  },
  {
    title: "Governance in the request path",
    detail: "Auth, policy snapshot, rate and budget checks, then forward.",
  },
  {
    title: "Idempotent usage accounting",
    detail: "Keyed per request, rolled up hourly, daily, and monthly.",
  },
  {
    title: "Multi-tenant control plane",
    detail: "Organizations, teams, projects, and scoped credentials.",
  },
];

export function Hero({
  headline = APP_TAGLINE,
  subheadline = APP_DESCRIPTION,
}: HeroProps) {
  return (
    <section className="border-b bg-gradient-to-b from-muted/50 to-background">
      <div className="mx-auto grid max-w-7xl gap-12 px-4 py-20 sm:px-6 lg:grid-cols-2 lg:items-center lg:py-28">
        <div>
          <p className="inline-flex items-center gap-2 rounded-full border bg-background px-3 py-1 text-xs font-medium text-muted-foreground">
            <ShieldCheck className="size-3.5 text-primary" aria-hidden />
            Governance enforced in the request path
          </p>

          <h1 className="mt-5 text-4xl font-semibold tracking-tight text-balance sm:text-5xl">
            {headline}
          </h1>

          <p className="mt-5 max-w-xl text-lg text-muted-foreground text-pretty">
            {subheadline}
          </p>

          <div className="mt-8 flex flex-wrap items-center gap-3">
            <Button asChild size="lg">
              <Link href="/app/overview">
                Open the dashboard
                <ArrowRight data-icon="inline-end" aria-hidden />
              </Link>
            </Button>
            <Button asChild size="lg" variant="outline">
              <Link href="/pricing">See pricing</Link>
            </Button>
          </div>

          <p className="mt-4 text-sm text-muted-foreground">
            No credit card required. {APP_NAME} is free for up to five people.
          </p>
        </div>

        <Card className="shadow-sm">
          <CardContent className="space-y-4">
            <p className="text-sm font-medium">What the gateway does</p>
            <ul className="space-y-3">
              {CAPABILITIES.map((capability) => (
                <li key={capability.title} className="flex gap-2.5">
                  <Check
                    className="mt-0.5 size-4 shrink-0 text-primary"
                    aria-hidden
                  />
                  <div className="space-y-0.5">
                    <p className="text-sm font-medium">{capability.title}</p>
                    <p className="text-xs text-muted-foreground text-pretty">
                      {capability.detail}
                    </p>
                  </div>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      </div>
    </section>
  );
}
