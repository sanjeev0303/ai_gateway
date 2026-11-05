import Link from "next/link";
import { FileClock, ShieldCheck, Wallet } from "lucide-react";

import { GatewayPrism } from "@/components/brand/gateway-prism";
import { Logo } from "@/components/logo";

const FACTS = [
  { icon: ShieldCheck, text: "Guardrails run before a prompt leaves" },
  { icon: Wallet, text: "Spend by team, project and model" },
  { icon: FileClock, text: "Every console change on the audit log" },
];

export default function AuthLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <div className="grid min-h-dvh flex-1 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
      <aside className="relative hidden overflow-hidden border-r border-border bg-card/45 p-12 lg:flex lg:flex-col lg:justify-between">
        <div aria-hidden className="ambient pointer-events-none absolute inset-0" />
        <Link href="/" className="relative w-fit rounded-lg focus-visible:ring-3 focus-visible:ring-ring/30 focus-visible:outline-none">
          <Logo />
        </Link>
        <div className="relative flex flex-col gap-6">
          <GatewayPrism size={140} className="-ml-6" />
          <p className="font-display text-[clamp(2.75rem,4vw,3.75rem)] leading-[0.98] text-foreground">
            Your AI <span className="text-copper">control plane.</span>
          </p>
          <p className="max-w-[40ch] text-[0.9375rem] leading-relaxed text-muted-foreground">
            See every model request your organization makes: who sent it, what it cost, and which
            policy it met.
          </p>
        </div>
        <ul className="relative flex flex-col gap-3">
          {FACTS.map(({ icon: Icon, text }) => (
            <li key={text} className="flex items-center gap-3 text-sm text-foreground/80">
              <span className="flex size-8 items-center justify-center rounded-xl border border-border bg-card text-copper">
                <Icon className="size-4" aria-hidden />
              </span>
              {text}
            </li>
          ))}
        </ul>
      </aside>
      <div className="flex flex-col">{children}</div>
    </div>
  );
}
