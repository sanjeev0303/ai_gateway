import { Check, Minus } from "lucide-react";

import { Container, SectionHeading } from "@/components/landing/section";
import { COMPARISON } from "@/lib/landing-data";
import { cn } from "@/lib/utils";

function Cell({ value, highlight }: { value: string; highlight: boolean }) {
  const affirmative = value === "Yes" || value === "Per request, at write time" || value === "Evaluated at the gateway" || value === "One append-only log" || value === "One environment variable";

  if (affirmative) {
    return (
      <span className={cn("inline-flex items-start gap-1.5", highlight && "font-medium")}>
        <Check className="mt-0.5 size-3.5 shrink-0 text-primary" aria-hidden />
        <span>{value}</span>
      </span>
    );
  }

  return (
    <span className={cn("inline-flex items-start gap-1.5 text-muted-foreground", highlight && "font-medium text-foreground")}>
      <Minus className="mt-0.5 size-3.5 shrink-0" aria-hidden />
      <span>{value}</span>
    </span>
  );
}

export function Comparison() {
  const [capability, inHouse, providerNative] = COMPARISON.columns;

  return (
    <section className="border-b">
      <Container className="py-20 sm:py-24 lg:py-28">
        <SectionHeading
          eyebrow={COMPARISON.eyebrow}
          title={COMPARISON.headline}
          body={COMPARISON.body}
        />

        <div className="mt-12 overflow-x-auto">
          <table className="w-full min-w-[680px] border-collapse text-left text-sm">
            <caption className="sr-only">
              Comparison of Sentinel, building in-house, and provider-native tooling
            </caption>
            <thead>
              <tr className="border-b">
                <th scope="col" className="w-[34%] py-3 pr-4 font-medium text-muted-foreground">
                  Capability
                </th>
                <th scope="col" className="py-3 pr-4 font-semibold">
                  {capability}
                </th>
                <th scope="col" className="py-3 pr-4 font-medium text-muted-foreground">
                  {inHouse}
                </th>
                <th scope="col" className="py-3 font-medium text-muted-foreground">
                  {providerNative}
                </th>
              </tr>
            </thead>
            <tbody>
              {COMPARISON.rows.map((row) => (
                <tr key={row.capability} className="border-b last:border-0">
                  <th scope="row" className="py-4 pr-4 font-normal">
                    {row.capability}
                  </th>
                  <td className="py-4 pr-4">
                    <Cell value={row.sentinel} highlight />
                  </td>
                  <td className="py-4 pr-4">
                    <Cell value={row.inHouse} highlight={false} />
                  </td>
                  <td className="py-4">
                    <Cell value={row.providerNative} highlight={false} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Container>
    </section>
  );
}
