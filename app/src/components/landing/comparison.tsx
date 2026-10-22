import { ScrollReveal } from "@/components/landing/scroll-reveal";
import { Container, SectionLabel } from "@/components/landing/section";
import { COMPARISON } from "@/lib/landing-data";

const NOT_IMPLEMENTED = "Not implemented";

function Cell({ value, ours = false }: { value: string; ours?: boolean }) {
  const affirmative = ours && value !== NOT_IMPLEMENTED;

  return (
    <span
      className="text-xs leading-relaxed"
      style={{
        color: affirmative ? "#1C1412" : "rgba(74,58,48,0.55)",
        fontWeight: affirmative ? 600 : 400,
      }}
    >
      {affirmative && (
        <span
          className="inline-block mr-2 align-middle size-1 rounded-full"
          style={{ background: "#6F4E37" }}
          aria-hidden
        />
      )}
      {value}
    </span>
  );
}

export function Comparison() {
  const [capability, inHouse, providerNative] = COMPARISON.columns;

  return (
    <section
      className="relative overflow-hidden border-b border-[rgba(111,78,55,0.10)]"
      style={{ background: "var(--canvas-warm)" }}
    >
      <Container className="py-24 sm:py-32 lg:py-40">
        <ScrollReveal>
          <div className="max-w-xl mb-16">
            <SectionLabel>{COMPARISON.eyebrow}</SectionLabel>
            <h2
              className="mt-4 font-display text-[clamp(2.5rem,5vw,4rem)] font-semibold text-[#1C1412] leading-[0.95]"
              style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.04em" }}
            >
              {COMPARISON.headline}
            </h2>
            <p className="mt-6 text-base leading-relaxed text-[#4A3A30] max-w-[48ch]" style={{ letterSpacing: "-0.01em" }}>
              {COMPARISON.body}
            </p>
          </div>
        </ScrollReveal>

        <ScrollReveal delay={0.15}>
          <div className="overflow-hidden border border-[rgba(111,78,55,0.16)]">
            <div className="overflow-x-auto">
              <table className="w-full min-w-175 border-collapse text-left">
                <caption className="sr-only">
                  Comparison of Sentinel, building in-house, and provider-native tooling
                </caption>
                <thead>
                  <tr style={{ borderBottom: "1px solid rgba(111,78,55,0.12)" }}>
                    <th
                      scope="col"
                      className="w-[32%] py-4 pl-7 pr-4 text-[10px] font-semibold uppercase text-[#7A6458]"
                      style={{ fontFamily: "var(--font-fira-code)", letterSpacing: "0.10em" }}
                    >
                      Capability
                    </th>
                    <th
                      scope="col"
                      className="w-[28%] py-4 px-5 text-[10px] font-bold uppercase text-[#6F4E37]"
                      style={{
                        fontFamily: "var(--font-fira-code)",
                        letterSpacing: "0.10em",
                        background: "rgba(111,78,55,0.04)",
                        borderLeft: "1px solid rgba(111,78,55,0.14)",
                        borderRight: "1px solid rgba(111,78,55,0.14)",
                      }}
                    >
                      {capability}
                    </th>
                    <th
                      scope="col"
                      className="w-[20%] py-4 px-5 text-[10px] font-semibold uppercase text-[#7A6458]"
                      style={{ fontFamily: "var(--font-fira-code)", letterSpacing: "0.10em" }}
                    >
                      {inHouse}
                    </th>
                    <th
                      scope="col"
                      className="w-[20%] py-4 pl-5 pr-7 text-[10px] font-semibold uppercase text-[#7A6458]"
                      style={{ fontFamily: "var(--font-fira-code)", letterSpacing: "0.10em" }}
                    >
                      {providerNative}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {COMPARISON.rows.map((row) => (
                    <tr
                      key={row.capability}
                      style={{ borderBottom: "1px solid rgba(111,78,55,0.07)" }}
                    >
                      <th
                        scope="row"
                        className="py-4 pl-7 pr-4 text-xs font-semibold text-[#1C1412]"
                        style={{ letterSpacing: "-0.01em" }}
                      >
                        {row.capability}
                      </th>
                      <td
                        className="py-4 px-5"
                        style={{
                          background: "rgba(111,78,55,0.03)",
                          borderLeft: "1px solid rgba(111,78,55,0.10)",
                          borderRight: "1px solid rgba(111,78,55,0.10)",
                        }}
                      >
                        <Cell value={row.sentinel} ours />
                      </td>
                      <td className="py-4 px-5">
                        <Cell value={row.inHouse} />
                      </td>
                      <td className="py-4 pl-5 pr-7">
                        <Cell value={row.providerNative} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </ScrollReveal>
      </Container>
    </section>
  );
}
