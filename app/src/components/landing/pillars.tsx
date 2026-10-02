import { Container, SectionHeading } from "@/components/landing/section";
import { PILLARS } from "@/lib/landing-data";

export function Pillars() {
  return (
    <section className="border-b">
      <Container className="py-20 sm:py-24 lg:py-28">
        <SectionHeading
          eyebrow={PILLARS.eyebrow}
          title={PILLARS.headline}
          body={PILLARS.body}
        />

        <ul className="mt-14 grid gap-x-12 gap-y-10 md:grid-cols-2">
          {PILLARS.items.map((pillar) => {
            const Icon = pillar.icon;
            return (
              <li key={pillar.title} className="border-t pt-6">
                <div className="flex items-center gap-3">
                  <span className="flex size-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
                    <Icon className="size-4" aria-hidden />
                  </span>
                  <span className="font-mono text-xs text-muted-foreground tabular-nums">
                    {pillar.index}
                  </span>
                </div>

                <h3 className="mt-4 text-xl font-semibold tracking-tight">{pillar.title}</h3>
                <p className="mt-2 text-sm leading-relaxed text-muted-foreground text-pretty">
                  {pillar.body}
                </p>

                <ul className="mt-4 flex flex-wrap gap-2">
                  {pillar.points.map((point) => (
                    <li
                      key={point}
                      className="rounded-full border px-2.5 py-1 text-xs text-muted-foreground"
                    >
                      {point}
                    </li>
                  ))}
                </ul>
              </li>
            );
          })}
        </ul>
      </Container>
    </section>
  );
}
