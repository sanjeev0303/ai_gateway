"use client";

import { useEffect, useRef } from "react";
import { Container } from "@/components/landing/section";

const PROVIDERS = [
  "OpenAI", "Anthropic", "Google Gemini", "Azure OpenAI", "AWS Bedrock",
  "Mistral", "Cohere", "Meta Llama", "Groq", "Together AI",
  "OpenAI", "Anthropic", "Google Gemini", "Azure OpenAI", "AWS Bedrock",
  "Mistral", "Cohere", "Meta Llama", "Groq", "Together AI",
];

export function ProviderStrip() {
  const trackRef = useRef<HTMLUListElement>(null);

  // Pause marquee on reduced-motion preference
  useEffect(() => {
    const mq = window.matchMedia("(prefers-reduced-motion: reduce)");
    if (mq.matches && trackRef.current) {
      trackRef.current.style.animationPlayState = "paused";
    }
  }, []);

  return (
    <section
      className="relative overflow-hidden border-y border-[rgba(111,78,55,0.10)]"
      style={{ background: "var(--canvas-warm)" }}
      aria-label="Supported AI providers"
    >
      {/* Fade masks on left/right edges */}
      <div
        className="pointer-events-none absolute inset-y-0 left-0 z-10 w-24"
        style={{
          background: "linear-gradient(to right, var(--canvas-warm), transparent)",
        }}
        aria-hidden
      />
      <div
        className="pointer-events-none absolute inset-y-0 right-0 z-10 w-24"
        style={{
          background: "linear-gradient(to left, var(--canvas-warm), transparent)",
        }}
        aria-hidden
      />

      <div className="py-6">
        <p
          className="text-center font-mono text-xs tracking-label text-[#7A6458] font-semibold uppercase mb-5"
          style={{ fontFamily: "var(--font-fira-code)", letterSpacing: "0.14em" }}
        >
          One gateway in front of every major provider
        </p>

        {/* Marquee track */}
        <div className="overflow-hidden" aria-hidden>
          <ul
            ref={trackRef}
            className="flex items-center gap-0 animate-marquee"
            style={{ width: "max-content" }}
          >
            {PROVIDERS.map((provider, i) => (
              <li key={`${provider}-${i}`} className="flex items-center">
                <span
                  className="whitespace-nowrap px-6 text-base font-medium text-[#4A3A30]"
                  style={{ fontFamily: "var(--font-dm-sans)", letterSpacing: "-0.02em" }}
                >
                  {provider}
                </span>
                <span
                  className="h-3 w-px"
                  style={{ background: "rgba(111,78,55,0.18)" }}
                  aria-hidden
                />
              </li>
            ))}
          </ul>
        </div>
      </div>
    </section>
  );
}
