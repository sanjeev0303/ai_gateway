"use client";

import Link from "next/link";
import { motion } from "motion/react";
import { fadeUp, staggerContainer } from "@/lib/motion";
import { DashboardPreview } from "@/components/landing/dashboard-preview";
import { Container } from "@/components/landing/section";
import { HERO } from "@/lib/landing-data";

const STATS = [
  { label: "Provider kinds", value: "5+" },
  { label: "Policy decisions", value: "Per request" },
  { label: "Overhead", value: "<1ms" },
] as const;

export function Hero() {
  return (
    <section
      className="relative overflow-hidden grain"
      style={{ background: "var(--canvas)" }}
    >
      {/* Vertical rule-of-thirds line — decorative */}
      <div className="thirds-line" aria-hidden />

      {/* Warm canvas-to-lavender ambient wash */}
      <div
        className="pointer-events-none absolute inset-0"
        style={{
          background:
            "radial-gradient(ellipse 60% 50% at 72% 50%, rgba(230,224,240,0.35) 0%, transparent 70%)",
        }}
        aria-hidden
      />

      <Container className="relative z-10 pt-24 pb-0 sm:pt-28 lg:pt-32">
        {/* Left text, right visual centered */}
        <div className="grid gap-8 lg:grid-cols-12 lg:items-center">

          {/* Left: editorial text block */}
          <motion.div
            variants={staggerContainer(0.1)}
            initial="hidden"
            animate="visible"
            className="lg:col-span-6 pb-8 lg:pb-0 pr-0 lg:pr-8"
          >
            {/* Eyebrow: monospaced label */}
            <motion.p
              variants={fadeUp}
              className="font-mono text-xs tracking-label text-[#6F4E37] uppercase mb-8"
              style={{ fontFamily: "var(--font-fira-code)", letterSpacing: "0.14em" }}
            >
              {HERO.eyebrow}
            </motion.p>

            {/* Display headline — editorial serif, tight tracking */}
            <motion.h1
              variants={fadeUp}
              className="font-display text-[clamp(3rem,6.5vw,5.5rem)] font-semibold text-[#1C1412] tracking-display leading-[0.92]"
              style={{
                fontFamily: "var(--font-cormorant)",
                letterSpacing: "-0.04em",
                lineHeight: "0.92",
              }}
            >
              See every AI
              <br />
              <span style={{ color: "#6F4E37" }}>request</span>
              <br />
              your company
              <br />
              makes.
            </motion.h1>

            {/* Body — DM Sans, generous leading */}
            <motion.p
              variants={fadeUp}
              className="mt-8 text-base sm:text-lg leading-relaxed text-[#4A3A30] max-w-[46ch] tracking-tight-body"
              style={{ letterSpacing: "-0.01em" }}
            >
              {HERO.subhead}
            </motion.p>

            {/* CTA row */}
            <motion.div variants={fadeUp} className="mt-10 flex flex-wrap items-center gap-4">
              <Link
                href={HERO.primary.href}
                id="hero-cta-primary"
                className="inline-flex items-center justify-center h-12 px-7 text-sm font-semibold text-[#F5F3F0] rounded-none transition-colors duration-200"
                style={{ background: "#6F4E37" }}
                onMouseEnter={e => (e.currentTarget.style.background = "#4A3224")}
                onMouseLeave={e => (e.currentTarget.style.background = "#6F4E37")}
              >
                {HERO.primary.label}
              </Link>
              <Link
                href={HERO.secondary.href}
                id="hero-cta-secondary"
                className="inline-flex items-center h-12 px-6 text-sm font-medium text-[#6F4E37] border border-[rgba(111,78,55,0.30)] transition-colors duration-200"
                style={{ background: "transparent" }}
                onMouseEnter={e => (e.currentTarget.style.borderColor = "#6F4E37")}
                onMouseLeave={e => (e.currentTarget.style.borderColor = "rgba(111,78,55,0.30)")}
              >
                {HERO.secondary.label}
              </Link>
            </motion.div>

            {/* Microcopy */}
            <motion.p variants={fadeUp} className="mt-5 text-xs text-[#7A6458]" style={{ fontFamily: "var(--font-fira-code)" }}>
              {HERO.microcopy}
            </motion.p>
          </motion.div>

          {/* Right: dashboard visual — centered */}
          <motion.div
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.9, delay: 0.25, ease: [0.16, 1, 0.3, 1] }}
            className="lg:col-span-6 relative flex items-center justify-center w-full"
          >
            <DashboardPreview className="w-full shadow-2xl" />
          </motion.div>
        </div>

        {/* Stats row — spans full width below split */}
        <motion.dl
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.8, delay: 0.55 }}
          className="border-t border-[rgba(111,78,55,0.12)] mt-12 py-7 grid grid-cols-3 gap-4 stat-block"
        >
          {STATS.map((stat) => (
            <div key={stat.label}>
              <dd>{stat.value}</dd>
              <dt className="mt-2">{stat.label}</dt>
            </div>
          ))}
        </motion.dl>
      </Container>
    </section>
  );
}
