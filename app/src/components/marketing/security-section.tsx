import { MARKETING_SECURITY } from "@/lib/constants";
import { cn } from "@/lib/utils";

interface SecuritySectionProps {
  eyebrow?: string;
  heading?: string;
  subheading?: string;
  className?: string;
}

export function SecuritySection({
  eyebrow = "Trust",
  heading = "Built to survive a procurement review",
  subheading = "The controls below are the ones buyers actually ask about, and the ones we answer without a call.",
  className,
}: SecuritySectionProps) {
  return (
    <section className={cn("border-b bg-muted/30", className)}>
      <div className="mx-auto max-w-7xl px-4 py-20 sm:px-6">
        <div className="max-w-2xl">
          <p className="text-sm font-semibold uppercase tracking-wide text-primary">
            {eyebrow}
          </p>
          <h2 className="mt-3 text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
            {heading}
          </h2>
          <p className="mt-4 text-lg text-muted-foreground text-pretty">
            {subheading}
          </p>
        </div>

        <ul className="mt-12 grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
          {MARKETING_SECURITY.map((item) => {
            const Icon = item.icon;
            return (
              <li
                key={item.title}
                className="rounded-xl border bg-card p-6 text-card-foreground"
              >
                <span className="inline-flex size-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                  <Icon className="size-5" aria-hidden />
                </span>
                <h3 className="mt-4 font-semibold">{item.title}</h3>
                <p className="mt-2 text-sm text-muted-foreground text-pretty">
                  {item.description}
                </p>
              </li>
            );
          })}
        </ul>
      </div>
    </section>
  );
}
