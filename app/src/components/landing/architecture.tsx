import { Container, SectionHeading } from "@/components/landing/section";
import { ARCHITECTURE } from "@/lib/landing-data";

export function Architecture() {
  return (
    <section className="border-b">
      <Container className="grid gap-12 py-20 sm:py-24 lg:grid-cols-12 lg:gap-16 lg:py-28">
        <div className="lg:col-span-5">
          <SectionHeading
            eyebrow={ARCHITECTURE.eyebrow}
            title={ARCHITECTURE.headline}
            body={ARCHITECTURE.body}
          />

          <ol className="mt-10 space-y-0">
            {ARCHITECTURE.steps.map((step, index) => (
              <li key={step.label} className="flex gap-4">
                <div className="flex flex-col items-center">
                  <span className="flex size-6 shrink-0 items-center justify-center rounded-full border font-mono text-[10px] tabular-nums">
                    {index + 1}
                  </span>
                  {index < ARCHITECTURE.steps.length - 1 ? (
                    <span className="w-px flex-1 bg-border" aria-hidden />
                  ) : null}
                </div>
                <div className="pb-5">
                  <p className="text-sm font-medium">{step.label}</p>
                  <p className="mt-0.5 text-xs text-muted-foreground">{step.detail}</p>
                </div>
              </li>
            ))}
          </ol>
        </div>

        <div className="lg:col-span-7">
          <div className="rounded-xl border bg-card p-5 sm:p-6">
            <h3 className="text-sm font-semibold tracking-tight">Fields on every record</h3>
            <dl className="mt-4 divide-y">
              {ARCHITECTURE.fields.map((row) => (
                <div key={row.field} className="grid gap-1 py-3.5 sm:grid-cols-[160px_1fr] sm:gap-4">
                  <dt className="font-mono text-xs">{row.field}</dt>
                  <dd className="text-xs leading-relaxed text-muted-foreground">{row.detail}</dd>
                </div>
              ))}
            </dl>
          </div>
        </div>
      </Container>
    </section>
  );
}
