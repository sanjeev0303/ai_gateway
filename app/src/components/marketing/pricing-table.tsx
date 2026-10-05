import Link from "next/link";
import { Check } from "lucide-react";

import { Button } from "@/components/ui/button";
import { PRICING_TIERS } from "@/lib/constants";

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
            Not priced yet
          </h2>
          <p className="mt-4 text-lg text-muted-foreground text-pretty">
            There is no billing, invoicing, or seat management in the product
            yet, so there is no price list to publish. Here is what the platform
            does today.
          </p>
        </div>

        <div className="mt-12 grid max-w-3xl gap-6">
          {PRICING_TIERS.map((tier) => (
            <div
              key={tier.name}
              className="flex flex-col rounded-xl border bg-card p-6 shadow-md"
            >
              <h3 className="text-lg font-semibold">{tier.name}</h3>

              <p className="mt-2 text-sm text-muted-foreground text-pretty">
                {tier.tagline}
              </p>

              <Button asChild className="mt-6 w-fit">
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
          ))}
        </div>
      </div>
    </section>
  );
}