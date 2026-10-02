import Link from "next/link";
import { ArrowRight, ShieldCheck } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { APP_DESCRIPTION, APP_NAME, APP_TAGLINE } from "@/lib/constants";
import { costSummary, organizationSummary } from "@/lib/mock-data";

interface HeroProps {
  /** Marketing pages pass their own headline; the homepage uses the tagline. */
  headline?: string;
  subheadline?: string;
}

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
            Policy enforcement in under 40ms
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
          <CardContent className="space-y-5">
            <div className="flex items-baseline justify-between">
              <div>
                <p className="text-sm font-medium">Policy compliance</p>
                <p className="text-xs text-muted-foreground">
                  Rolling 30 days, all projects
                </p>
              </div>
              <p className="text-2xl font-semibold tabular-nums">
                {organizationSummary.policyCompliance}%
              </p>
            </div>
            <Progress value={organizationSummary.policyCompliance} />

            <div className="space-y-2 border-t pt-4 text-sm">
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground">Spend this month</span>
                <span className="font-medium tabular-nums">
                  ${costSummary.mtd.toLocaleString("en-US")}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground">Open alerts</span>
                <span className="font-medium tabular-nums">
                  {organizationSummary.openAlerts}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground">Blended cost per MTok</span>
                <span className="font-medium tabular-nums">
                  ${costSummary.blendedCostPerMTok}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground">Month forecast</span>
                <span className="font-medium tabular-nums">
                  ${costSummary.forecast.projected.toLocaleString("en-US")}
                </span>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </section>
  );
}
