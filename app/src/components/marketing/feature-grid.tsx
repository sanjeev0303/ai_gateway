import { MARKETING_FEATURES } from "@/lib/constants";
import { cn } from "@/lib/utils";

interface FeatureGridProps {
  /** Optional section heading; omit to render the grid alone. */
  eyebrow?: string;
  heading?: string;
  subheading?: string;
  className?: string;
}

export function FeatureGrid({
  eyebrow,
  heading,
  subheading,
  className,
}: FeatureGridProps) {
  return (
    <section className={cn("border-b", className)}>
      <div className="mx-auto max-w-7xl px-4 py-20 sm:px-6">
        {heading && (
          <div className="max-w-2xl">
            {eyebrow && (
              <p className="text-sm font-semibold uppercase tracking-wide text-primary">
                {eyebrow}
              </p>
            )}
            <h2 className="mt-3 text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
              {heading}
            </h2>
            {subheading && (
              <p className="mt-4 text-lg text-muted-foreground text-pretty">
                {subheading}
              </p>
            )}
          </div>
        )}

        <ul className="mt-12 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {MARKETING_FEATURES.map((feature) => {
            const Icon = feature.icon;
            return (
              <li
                key={feature.title}
                className="rounded-xl border bg-card p-6 text-card-foreground transition-shadow hover:shadow-sm"
              >
                <span className="inline-flex size-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                  <Icon className="size-5" aria-hidden />
                </span>
                <h3 className="mt-4 font-semibold">{feature.title}</h3>
                <p className="mt-2 text-sm text-muted-foreground text-pretty">
                  {feature.description}
                </p>
              </li>
            );
          })}
        </ul>
      </div>
    </section>
  );
}
