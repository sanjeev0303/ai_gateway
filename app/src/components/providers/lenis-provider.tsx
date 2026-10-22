"use client";

import { createContext, useContext, useEffect, useRef, useState, useCallback } from "react";
import Lenis from "lenis";
import gsap from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";

gsap.registerPlugin(ScrollTrigger);

interface LenisContextValue {
  lenis: Lenis | null;
  scrollTo: (target: string | number | HTMLElement, options?: { offset?: number; duration?: number; immediate?: boolean }) => void;
}

const LenisContext = createContext<LenisContextValue>({
  lenis: null,
  scrollTo: () => {},
});

export const useLenis = () => useContext(LenisContext);

/**
 * Lenis smooth scroll scoped to the marketing layout only.
 * Exposes Lenis instance and scrollTo via LenisContext for header navigation.
 */
export function LenisProvider({ children }: { children: React.ReactNode }) {
  const [lenisInstance, setLenisInstance] = useState<Lenis | null>(null);
  const lenisRef = useRef<Lenis | null>(null);

  useEffect(() => {
    const lenis = new Lenis({
      duration: 1.2,
      easing: (t) => Math.min(1, 1.001 - Math.pow(2, -10 * t)),
      smoothWheel: true,
      touchMultiplier: 1.5,
      prevent: (node) => {
        return (
          node.id === "dashboard-main" ||
          node.closest("[data-lenis-prevent]") !== null
        );
      },
    });

    lenisRef.current = lenis;
    setLenisInstance(lenis);

    // Sync GSAP ScrollTrigger with Lenis scroll position
    lenis.on("scroll", ScrollTrigger.update);

    // Drive Lenis via GSAP ticker for frame sync
    const tick = (time: number) => lenis.raf(time * 1000);
    gsap.ticker.add(tick);
    gsap.ticker.lagSmoothing(0);

    // Handle initial hash navigation if present
    if (typeof window !== "undefined" && window.location.hash) {
      const hash = window.location.hash;
      const target = document.querySelector(hash);
      if (target) {
        setTimeout(() => {
          lenis.scrollTo(target as HTMLElement, { offset: -70 });
        }, 300);
      }
    }

    return () => {
      lenis.destroy();
      gsap.ticker.remove(tick);
      lenisRef.current = null;
      setLenisInstance(null);
    };
  }, []);

  const scrollTo = useCallback((target: string | number | HTMLElement, options?: { offset?: number; duration?: number; immediate?: boolean }) => {
    const opts = { offset: -70, ...options };
    if (lenisRef.current) {
      lenisRef.current.scrollTo(target, opts);
    } else if (typeof window !== "undefined") {
      if (typeof target === "number") {
        window.scrollTo({ top: target, behavior: opts.immediate ? "auto" : "smooth" });
      } else {
        const el = typeof target === "string" ? document.querySelector(target) : target;
        if (el) {
          const top = el.getBoundingClientRect().top + window.scrollY + (opts.offset ?? 0);
          window.scrollTo({ top, behavior: opts.immediate ? "auto" : "smooth" });
        }
      }
    }
  }, []);

  return (
    <LenisContext.Provider value={{ lenis: lenisInstance, scrollTo }}>
      {children}
    </LenisContext.Provider>
  );
}
