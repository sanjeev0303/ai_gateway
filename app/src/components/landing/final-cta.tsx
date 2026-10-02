import { Container } from "@/components/landing/section";
import { Button } from "@/components/ui/button";
import { FINAL_CTA } from "@/lib/landing-data";

export function FinalCta() {
  return (
    <section className="border-b">
      <Container className="py-20 sm:py-24 lg:py-32">
        <div className="mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
            {FINAL_CTA.headline}
          </h2>
          <p className="mt-4 text-lg leading-relaxed text-muted-foreground text-pretty">
            {FINAL_CTA.body}
          </p>

          <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
            <Button asChild size="lg" className="w-full sm:w-auto">
              <a href={FINAL_CTA.primary.href}>{FINAL_CTA.primary.label}</a>
            </Button>
            <Button asChild size="lg" variant="outline" className="w-full sm:w-auto">
              <a href={FINAL_CTA.secondary.href}>{FINAL_CTA.secondary.label}</a>
            </Button>
          </div>
        </div>
      </Container>
    </section>
  );
}
