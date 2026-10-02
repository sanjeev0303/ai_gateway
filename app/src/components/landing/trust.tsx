import { Container, SectionHeading } from "@/components/landing/section";
import { TRUST } from "@/lib/landing-data";

export function Trust() {
  return (
    <section className="border-b">
      <Container className="py-20 sm:py-24 lg:py-28">
        <SectionHeading
          eyebrow={TRUST.eyebrow}
          title={TRUST.headline}
          body={TRUST.body}
        />

        <ul className="mt-14 grid gap-px overflow-hidden rounded-xl border bg-border md:grid-cols-2 lg:grid-cols-3">
          {TRUST.items.map((mechanism) => {
            const Icon = mechanism.icon;
            return (
              <li key={mechanism.title} className="bg-card p-6">
                <span className="flex size-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
                  <Icon className="size-4" aria-hidden />
                </span>
                <h3 className="mt-4 font-semibold tracking-tight">{mechanism.title}</h3>
                <p className="mt-2 text-sm leading-relaxed text-muted-foreground text-pretty">
                  {mechanism.body}
                </p>
              </li>
            );
          })}
        </ul>
      </Container>
    </section>
  );
}
