import { MARKETING_PERSONAS } from "@/lib/constants";
import { cn } from "@/lib/utils";

interface PersonaGridProps {
  eyebrow?: string;
  heading?: string;
  subheading?: string;
  className?: string;
}

export function PersonaGrid({
  eyebrow = "Who it is for",
  heading = "Different teams, same first week",
  subheading = "Everyone arrives for a different reason. They all end up reading the same request log.",
  className,
}: PersonaGridProps) {
  return (
    <section className={cn("border-b", className)}>
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
          {MARKETING_PERSONAS.map((persona) => {
            const Icon = persona.icon;
            return (
              <li
                key={persona.title}
                className="rounded-xl border bg-card p-6 text-card-foreground"
              >
                <span className="inline-flex size-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                  <Icon className="size-5" aria-hidden />
                </span>
                <h3 className="mt-4 font-semibold">{persona.title}</h3>
                <p className="mt-2 text-sm text-muted-foreground text-pretty">
                  {persona.description}
                </p>
              </li>
            );
          })}
        </ul>
      </div>
    </section>
  );
}
