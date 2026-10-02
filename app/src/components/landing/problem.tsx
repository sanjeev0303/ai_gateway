import { Container, SectionHeading } from "@/components/landing/section";
import { PROBLEM } from "@/lib/landing-data";

export function Problem() {
  return (
    <section className="border-b">
      <Container className="grid gap-12 py-20 sm:py-24 lg:grid-cols-12 lg:gap-16 lg:py-28">
        <div className="lg:col-span-5">
          <SectionHeading
            eyebrow={PROBLEM.eyebrow}
            title={PROBLEM.headline}
            body={PROBLEM.body}
            titleClassName="lg:text-[2.75rem] lg:leading-[1.1]"
          />
        </div>

        <ul className="lg:col-span-7">
          {PROBLEM.points.map((point, index) => {
            const Icon = point.icon;
            return (
              <li
                key={point.title}
                className="grid gap-3 border-t py-7 first:border-t-0 first:pt-0 sm:grid-cols-[auto_1fr] sm:gap-6"
              >
                <span className="flex items-center gap-3 sm:flex-col sm:items-start">
                  <Icon className="size-5 shrink-0 text-primary" aria-hidden />
                  <span className="font-mono text-xs text-muted-foreground tabular-nums">
                    {String(index + 1).padStart(2, "0")}
                  </span>
                </span>

                <div>
                  <h3 className="font-semibold tracking-tight">{point.title}</h3>
                  <p className="mt-1.5 text-sm leading-relaxed text-muted-foreground text-pretty">
                    {point.body}
                  </p>
                </div>
              </li>
            );
          })}
        </ul>
      </Container>
    </section>
  );
}
