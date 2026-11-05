"use client";

import { useState } from "react";
import { Check, Copy } from "lucide-react";

import { cn } from "@/lib/utils";

const SAMPLES = [
  {
    id: "sdk",
    label: "TypeScript",
    code: `import { createGatewayClient } from "@ai-governance/api-client";

const gateway = createGatewayClient({
  gatewayUrl: process.env.AIGW_GATEWAY_URL,
  apiKey: process.env.AIGW_API_KEY,
});

const { text, meta } = await gateway.chatCompletion({
  model: "gemini-2.5-flash",
  messages: [{ role: "user", content: "Summarise this ticket." }],
});`,
  },
  {
    id: "cli",
    label: "CLI",
    code: `# An admin issues a gateway key for a service
aigw credential create --name payments-service \\
  --subject-type service_account --subject-id svc_payments --expires-in 90d

# Whoever holds the key signs in once, then calls any model they are allowed
aigw auth login --gateway-url https://gateway.your-company.internal --credential aigw_...
aigw models list
aigw request -m claude-sonnet-4-5 -p "Write a retry helper in Go"`,
  },
  {
    id: "curl",
    label: "cURL",
    code: `curl $AIGW_GATEWAY_URL/v1/chat/completions \\
  -H "Authorization: Bearer $AIGW_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gemini-2.5-flash",
    "messages": [{ "role": "user", "content": "Hello" }]
  }'`,
  },
];

export function CodeTabs() {
  const [active, setActive] = useState(SAMPLES[0]!.id);
  const [copied, setCopied] = useState(false);
  const sample = SAMPLES.find((s) => s.id === active) ?? SAMPLES[0]!;

  // Arrow keys move between tabs, as a tablist should.
  const onTabKey = (e: React.KeyboardEvent<HTMLButtonElement>) => {
    if (e.key !== "ArrowRight" && e.key !== "ArrowLeft") return;
    e.preventDefault();
    const index = SAMPLES.findIndex((s) => s.id === active);
    const step = e.key === "ArrowRight" ? 1 : -1;
    const next = SAMPLES[(index + step + SAMPLES.length) % SAMPLES.length]!;
    setActive(next.id);
    setCopied(false);
    document.getElementById(`code-tab-${next.id}`)?.focus();
  };

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(sample.code);
      setCopied(true);
      setTimeout(() => setCopied(false), 1600);
    } catch {
      // Clipboard can be blocked; the code stays selectable.
    }
  };

  return (
    <div className="overflow-hidden rounded-2xl border border-[rgba(132,162,240,0.18)] bg-(--terminal-bg) text-(--terminal-text) shadow-(--shadow-dialog)">
      <div className="flex items-center justify-between gap-3 border-b border-white/8 px-3 py-2.5">
        <div role="tablist" aria-label="Client" className="flex gap-1">
          {SAMPLES.map((s) => (
            <button
              key={s.id}
              id={`code-tab-${s.id}`}
              type="button"
              role="tab"
              aria-selected={s.id === active}
              aria-controls="code-panel"
              tabIndex={s.id === active ? 0 : -1}
              onKeyDown={onTabKey}
              onClick={() => {
                setActive(s.id);
                setCopied(false);
              }}
              className={cn(
                "h-8 rounded-full px-3 text-xs font-medium transition-[background-color,color] duration-150 focus-visible:ring-2 focus-visible:ring-[#4fc7bd] focus-visible:outline-none",
                s.id === active ? "bg-white/10 text-[#e8edf7]" : "text-(--terminal-dim) hover:text-[#e8edf7]",
              )}
            >
              {s.label}
            </button>
          ))}
        </div>
        <button
          type="button"
          onClick={copy}
          aria-label={copied ? "Copied" : "Copy code"}
          className="flex size-8 items-center justify-center rounded-full text-(--terminal-dim) transition-colors duration-150 hover:bg-white/10 hover:text-[#e8edf7] focus-visible:ring-2 focus-visible:ring-[#4fc7bd] focus-visible:outline-none"
        >
          {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
        </button>
      </div>
      <span aria-live="polite" className="sr-only">{copied ? "Code copied" : ""}</span>
      <pre
        id="code-panel"
        role="tabpanel"
        aria-labelledby={`code-tab-${active}`}
        tabIndex={0}
        translate="no"
        className="overflow-x-auto p-5 text-[0.8125rem] leading-relaxed focus-visible:ring-2 focus-visible:ring-[#4fc7bd] focus-visible:outline-none focus-visible:ring-inset"
      >
        <code>{sample.code}</code>
      </pre>
    </div>
  );
}
