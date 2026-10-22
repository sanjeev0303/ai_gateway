import { Variants } from "motion/react";

// ─── Motion/React Variants ──────────────────────────────────────────────────

/** Fade up for entry animations — GPU-composited only */
export const fadeUp: Variants = {
  hidden: { opacity: 0, y: 28 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { duration: 0.7, ease: [0.16, 1, 0.3, 1] },
  },
};

/** Fade in without movement */
export const fadeIn: Variants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: { duration: 0.6, ease: [0.16, 1, 0.3, 1] },
  },
};

/** Scale in from slightly smaller */
export const scaleIn: Variants = {
  hidden: { opacity: 0, scale: 0.96 },
  visible: {
    opacity: 1,
    scale: 1,
    transition: { duration: 0.6, ease: [0.16, 1, 0.3, 1] },
  },
};

/** Stagger container */
export const staggerContainer = (stagger = 0.09): Variants => ({
  hidden: {},
  visible: {
    transition: { staggerChildren: stagger, delayChildren: 0.05 },
  },
});

/** Slide in from left */
export const slideInLeft: Variants = {
  hidden: { opacity: 0, x: -28 },
  visible: {
    opacity: 1,
    x: 0,
    transition: { duration: 0.7, ease: [0.16, 1, 0.3, 1] },
  },
};

/** Slide in from right */
export const slideInRight: Variants = {
  hidden: { opacity: 0, x: 28 },
  visible: {
    opacity: 1,
    x: 0,
    transition: { duration: 0.7, ease: [0.16, 1, 0.3, 1] },
  },
};

/** Editorial line reveal — clip path wipe */
export const lineReveal: Variants = {
  hidden: { clipPath: "inset(0 100% 0 0)" },
  visible: {
    clipPath: "inset(0 0% 0 0)",
    transition: { duration: 0.8, ease: [0.76, 0, 0.24, 1] },
  },
};

/** Number count-up placeholder (orchestrated via GSAP) */
export const countUp: Variants = {
  hidden: { opacity: 0, y: 16 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { duration: 0.5, ease: [0.16, 1, 0.3, 1] },
  },
};

// ─── GSAP Easing Strings ────────────────────────────────────────────────────

/** Use with gsap.to / gsap.from's `ease` option */
export const EASE = {
  /** Expo out — fast entry, soft land */
  expoOut: "expo.out",
  /** Power4 out — slightly snappier */
  power4Out: "power4.out",
  /** Smooth sine */
  sineOut: "sine.out",
  /** Spring-like */
  back: "back.out(1.2)",
} as const;

// ─── Scroll Trigger Defaults ─────────────────────────────────────────────────

export const SCROLL_DEFAULTS = {
  start: "top 88%",
  end: "bottom 20%",
  toggleActions: "play none none none",
} as const;
