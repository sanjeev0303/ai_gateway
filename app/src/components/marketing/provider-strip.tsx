import { MARKETING_PROVIDERS } from "@/lib/constants";
import { cn } from "@/lib/utils";

export function ProviderStrip({ className }: { className?: string }) {
  return (
    <section className={cn("border-b", className)}>
      <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6">
        <p className="text-center text-sm text-muted-foreground">
          One endpoint in front of every provider
        </p>
        <ul className="mt-8 flex flex-wrap items-center justify-center gap-x-10 gap-y-4">
          {MARKETING_PROVIDERS.map((provider) => (
            <li
              key={provider}
              className="text-lg font-semibold tracking-tight text-muted-foreground/70"
            >
              {provider}
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}
