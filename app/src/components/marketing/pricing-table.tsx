import Link from "next/link";
import { Check } from "lucide-react";

import { Button } from "@/components/ui/button";
import { PRICING_TIERS } from "@/lib/constants";
import { cn } from "@/lib/utils";

type Tier = (typeof PRICING_TIERS)[number];

/** `featured` only exists on the Growth entry in the const tuple. */
const isFeatured = (tier: Tier) => "featured" in tier && tier.featured === true;

function Price({ tier }: { tier: Tier }) {
  if (tier.monthlyPrice === null) {
    return <span className="text-4xl font-semibold tracking-tight">Custom</span>;
  }

  return (
    <span className="flex items-baseline gap-1">
      <span className="text-4xl font-semibold tracking-tight tabular-nums">
        ${tier.monthlyPrice}
      </span>
      <span className="text-sm text-muted-foreground">/month</span>
    </span>
  );
}

interface PricingTableProps {
  className?: string;
}

export function PricingTable({ className }: PricingTableProps) {
  return (
    <section className={className}>
      <div className="mx-auto max-w-7xl px-4 py-20 sm:px-6">
        <div className="max-w-2xl">
          <p className="text-sm font-semibold uppercase tracking-wide text-primary">
            Pricing
          </p>
          <h2 className="mt-3 text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
            Priced per request, not per seat theatre
          </h2>
          <p className="mt-4 text-lg text-muted-foreground text-pretty">
            Every plan includes the gateway. You only pay more when you need
            policy enforcement, retention, and controls.
          </p>
        </div>

        <div className="mt-12 grid gap-6 lg:grid-cols-3">
          {PRICING_TIERS.map((tier) => {
            const featured = isFeatured(tier);
            return (
              <div
                key={tier.name}
                className={cn(
                  "flex flex-col rounded-xl border p-6",
                  featured
                    ? "border-primary bg-card shadow-md ring-1 ring-primary"
                    : "bg-card",
                )}
              >
                <div className="flex items-center gap-2">
                  <h3 className="text-lg font-semibold">{tier.name}</h3>
                  {featured && (
                    <span className="rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
                      Most popular
                    </span>
                  )}
                </div>

                <p className="mt-2 min-h-10 text-sm text-muted-foreground text-pretty">
                  {tier.tagline}
                </p>

                <div className="mt-6">
                  <Price tier={tier} />
                </div>

                <Button
                  asChild
                  className="mt-6"
                  variant={featured ? "default" : "outline"}
                >
                  <Link href="/app/overview">{tier.cta}</Link>
                </Button>

                <ul className="mt-6 space-y-2.5 text-sm">
                  {tier.features.map((feature) => (
                    <li key={feature} className="flex items-start gap-2">
                      <Check
                        className="mt-0.5 size-4 shrink-0 text-primary"
                        aria-hidden
                      />
                      <span className="text-muted-foreground">{feature}</span>
                    </li>
                  ))}
                </ul>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
