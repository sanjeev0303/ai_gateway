import { MARKETING_PROVIDERS } from "@/lib/constants";
import { Container } from "@/components/landing/section";

export function ProviderStrip() {
  return (
    <section className="border-b">
      <Container className="py-12">
        <p className="text-center text-sm text-muted-foreground">
          One endpoint in front of every provider
        </p>
        <ul className="mt-8 flex flex-wrap items-center justify-center gap-x-8 gap-y-4 sm:gap-x-12">
          {MARKETING_PROVIDERS.map((provider) => (
            <li
              key={provider}
              className="text-base font-medium tracking-tight text-muted-foreground/70"
            >
              {provider}
            </li>
          ))}
        </ul>
      </Container>
    </section>
  );
}
