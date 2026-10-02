import { Container, SectionHeading } from "@/components/landing/section";
import { GOVERNANCE } from "@/lib/landing-data";

/**
 * The single dark section. The project's `@custom-variant dark (&:is(.dark *))`
 * plus its `.dark` variable block mean adding `dark` here re-resolves the
 * semantic tokens to the real dark palette instead of hardcoding colours.
 */
export function Governance() {
  return (
    <section className="dark border-b bg-background text-foreground">
      <Container className="py-20 sm:py-24 lg:py-28">
        <SectionHeading
          eyebrow={GOVERNANCE.eyebrow}
          title={GOVERNANCE.headline}
          body={GOVERNANCE.body}
        />

        <ul className="mt-14 grid gap-x-12 gap-y-10 md:grid-cols-2">
          {GOVERNANCE.capabilities.map((item) => {
            const Icon = item.icon;
            return (
              <li key={item.title} className="border-t border-border pt-6">
                <span className="flex size-9 items-center justify-center rounded-lg bg-primary/15 text-primary">
                  <Icon className="size-4" aria-hidden />
                </span>
                <h3 className="mt-4 text-lg font-semibold tracking-tight">{item.title}</h3>
                <p className="mt-2 text-sm leading-relaxed text-muted-foreground text-pretty">
                  {item.body}
                </p>
              </li>
            );
          })}
        </ul>

        <dl className="mt-14 grid gap-px overflow-hidden rounded-xl border bg-border sm:grid-cols-2 lg:grid-cols-4">
          {GOVERNANCE.evidence.map((item) => (
            <div key={item.label} className="bg-background p-5">
              <dt className="text-xs text-muted-foreground">{item.label}</dt>
              <dd className="mt-1.5 text-sm font-medium">{item.value}</dd>
            </div>
          ))}
        </dl>
      </Container>
    </section>
  );
}
