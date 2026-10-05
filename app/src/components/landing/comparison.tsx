import { Check, Minus } from "lucide-react";

import { Container, SectionHeading } from "@/components/landing/section";
import { COMPARISON } from "@/lib/landing-data";

const NOT_IMPLEMENTED = "Not implemented";

function Cell({ value, ours = false }: { value: string; ours?: boolean }) {
  const affirmative = ours && value !== NOT_IMPLEMENTED;

  if (affirmative) {
    return (
      <span className="inline-flex items-start gap-1.5 font-medium">
        <Check className="mt-0.5 size-3.5 shrink-0 text-primary" aria-hidden />
        <span>{value}</span>
      </span>
    );
  }

  return (
    <span className="inline-flex items-start gap-1.5 text-muted-foreground">
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
                    <Cell value={row.sentinel} ours />
                  </td>
                  <td className="py-4 pr-4">
                    <Cell value={row.inHouse} />
                  </td>
                  <td className="py-4">
                    <Cell value={row.providerNative} />
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
