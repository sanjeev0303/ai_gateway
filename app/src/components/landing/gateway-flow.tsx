import { ArrowRight } from "lucide-react";

import { Container, SectionHeading } from "@/components/landing/section";
import { GATEWAY_FLOW } from "@/lib/landing-data";
import { cn } from "@/lib/utils";

function Hop({
  label,
  detail,
  emphasis,
}: {
  label: string;
  detail: string;
  emphasis?: boolean;
}) {
  return (
    <div
      className={cn(
        "rounded-lg border p-4",
        emphasis ? "border-primary/40 bg-primary/[0.04]" : "bg-card",
      )}
    >
      <p className="text-sm font-semibold tracking-tight">{label}</p>
      <p className="mt-1 text-xs text-muted-foreground">{detail}</p>
    </div>
  );
}

export function GatewayFlow() {
  const { inbound, outbound, loop } = GATEWAY_FLOW;

  return (
    <section className="border-b">
      <Container className="py-20 sm:py-24 lg:py-28">
        <SectionHeading
          eyebrow={GATEWAY_FLOW.eyebrow}
          title={GATEWAY_FLOW.headline}
          body={GATEWAY_FLOW.body}
        />

        <ol className="mt-12 grid items-stretch gap-3 md:grid-cols-[1fr_auto_1fr_auto_1fr]">
          {inbound.map((step, index) => (
            <div key={step.label} className="contents">
              <li className="md:[&>div]:h-full">
                <Hop {...step} emphasis={index === 1} />
              </li>
              <li className="flex items-center justify-center" aria-hidden>
                <ArrowRight className="size-4 rotate-90 text-muted-foreground md:rotate-0" />
              </li>
            </div>
          ))}
          <li className="md:[&>div]:h-full">
            <Hop {...outbound} />
          </li>
        </ol>

        <div className="mt-12 border-t pt-10">
          <h3 className="text-sm font-semibold tracking-tight">
            The loop that runs on every request
          </h3>

          <ol className="mt-6 grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
            {loop.map((step, index) => (
              <li key={step.title}>
                <div className="flex items-center gap-2">
                  <span className="flex size-6 shrink-0 items-center justify-center rounded-full border font-mono text-[10px] tabular-nums">
                    {index + 1}
                  </span>
                  <h4 className="text-sm font-medium">{step.title}</h4>
                </div>
                <p className="mt-2 pl-8 text-sm leading-relaxed text-muted-foreground text-pretty">
                  {step.detail}
                </p>
              </li>
            ))}
          </ol>
        </div>
      </Container>
    </section>
  );
}
