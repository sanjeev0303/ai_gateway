import { ArrowRight } from "lucide-react";

import { Container, SectionHeading } from "@/components/landing/section";
import { Button } from "@/components/ui/button";
import { PRICING_TEASER } from "@/lib/landing-data";

export function PricingTeaser() {
  const { eyebrow, headline, body, points, primary, secondary } = PRICING_TEASER;

  return (
    <section className="border-b">
      <Container className="py-20 sm:py-24 lg:py-28">
        <SectionHeading eyebrow={eyebrow} title={headline} body={body} />

        <div className="mt-12 grid gap-10 lg:grid-cols-12 lg:gap-16">
          <ul className="lg:col-span-7">
            {points.map((point, index) => (
              <li
                key={point}
                className="flex items-center gap-4 border-t py-5 first:border-t-0 first:pt-0"
              >
                <span className="font-mono text-xs text-muted-foreground tabular-nums">
                  {String(index + 1).padStart(2, "0")}
                </span>
                <span className="text-base">{point}</span>
              </li>
            ))}
          </ul>

          <div className="flex flex-col justify-start gap-3 lg:col-span-5">
            <Button asChild size="lg" className="w-full justify-between">
              <a href={primary.href}>
                {primary.label}
                <ArrowRight className="size-4" aria-hidden />
              </a>
            </Button>
            <Button asChild size="lg" variant="outline" className="w-full">
              <a href={secondary.href}>{secondary.label}</a>
            </Button>
            <p className="mt-1 text-sm text-muted-foreground">{body}</p>
          </div>
        </div>
      </Container>
    </section>
  );
}
