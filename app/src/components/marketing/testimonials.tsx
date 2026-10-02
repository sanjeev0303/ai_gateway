import { MARKETING_TESTIMONIALS } from "@/lib/constants";
import { cn } from "@/lib/utils";

interface TestimonialsProps {
  heading?: string;
  subheading?: string;
  className?: string;
}

export function Testimonials({
  heading = "What the first quarter looked like",
  subheading = "Three teams, three different reasons to buy, one shared objection: the number matched.",
  className,
}: TestimonialsProps) {
  return (
    <section className={cn("border-b", className)}>
      <div className="mx-auto max-w-7xl px-4 py-20 sm:px-6">
        <div className="max-w-2xl">
          <h2 className="text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
            {heading}
          </h2>
          <p className="mt-4 text-lg text-muted-foreground text-pretty">
            {subheading}
          </p>
        </div>

        <ul className="mt-12 grid gap-6 lg:grid-cols-3">
          {MARKETING_TESTIMONIALS.map((entry) => (
            <li
              key={entry.name}
              className="flex flex-col rounded-xl border bg-card p-6 text-card-foreground"
            >
              <blockquote className="flex-1 text-sm leading-relaxed text-pretty">
                {entry.quote}
              </blockquote>
              <footer className="mt-6 border-t pt-4">
                <p className="text-sm font-medium">{entry.name}</p>
                <p className="text-sm text-muted-foreground">
                  {entry.role}, {entry.company}
                </p>
              </footer>
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}
