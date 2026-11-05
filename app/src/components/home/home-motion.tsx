"use client";

import { useEffect } from "react";
import gsap from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";

gsap.registerPlugin(ScrollTrigger);

/**
 * The homepage's motion. Hidden starting states are set before first paint
 * (MOTION_BOOT in the root layout), so a reload never shows things complete and
 * then hides them. Without motion, everything is simply complete.
 *
 * - The hero's guilloche is engraved once and the sample request prints its checks.
 * - Each workflow diagram builds step by step with the scroll and unbuilds on the
 *   way back: its timeline is scrubbed by the figure's journey up the screen.
 * - The hairline curves behind sections draw in as their section arrives.
 * Scroll position comes from Lenis, which already drives ScrollTrigger.update.
 */
export function HomeMotion() {
  useEffect(() => {
    document.documentElement.classList.add("motion-live");
    const mm = gsap.matchMedia();

    mm.add("(prefers-reduced-motion: no-preference)", () => {
      // The paths are measured as 1 (pathLength), so a dash of 1 hides or shows a whole line.
      const lines = gsap.utils.toArray<SVGPathElement>("section[aria-labelledby='hero-heading'] [data-guilloche] path");
      const stamps = gsap.utils.toArray<HTMLElement>("[data-ledger] [data-stamp]");
      const hero = gsap.timeline({ defaults: { ease: "expo.out" } });
      hero.fromTo(lines, { strokeDasharray: 1, strokeDashoffset: 1 }, { strokeDashoffset: 0, duration: 1.6, ease: "power2.inOut", stagger: 0.03 }, 0);
      hero.fromTo(stamps, { autoAlpha: 0, y: 6, scale: 0.97 }, { autoAlpha: 1, y: 0, scale: 1, duration: 0.5, stagger: 0.16 }, 0.5);

      gsap.utils.toArray<HTMLElement>("[data-scroll-diagram]").forEach((figure) => {
        const steps = gsap.utils.toArray<SVGElement | HTMLElement>("[data-scroll-step]", figure);
        const order = [...new Set(steps.map((el) => Number(el.getAttribute("data-scroll-step"))))].sort((a, b) => a - b);
        const tl = gsap.timeline({
          defaults: { ease: "none" },
          scrollTrigger: { trigger: figure, start: "top 82%", end: "bottom 62%", scrub: 0.5 },
        });
        order.forEach((step, i) => {
          const group = steps.filter((el) => Number(el.getAttribute("data-scroll-step")) === step);
          tl.fromTo(group, { opacity: 0.12, y: 8 }, { opacity: 1, y: 0, duration: 1 }, i);
          const connectors = group.flatMap((el) => gsap.utils.toArray<SVGPathElement>("[data-connector]", el));
          if (connectors.length) {
            tl.fromTo(connectors, { strokeDasharray: 1, strokeDashoffset: 1 }, { strokeDashoffset: 0, duration: 1 }, i);
          }
        });
      });

      gsap.utils.toArray<SVGSVGElement>("[data-flow-curves]").forEach((svg) => {
        const section = svg.closest("section") ?? svg.parentElement ?? svg;
        const lines = gsap.utils.toArray<SVGPathElement>("[data-flow-lines] path", svg);
        const pulses = gsap.utils.toArray<SVGPathElement>("[data-flow-pulses] path", svg);
        const inHero = section.matches("[aria-labelledby='hero-heading']");

        // The lines draw in: on load in the hero, which is already on screen; elsewhere as the section arrives.
        if (inHero) {
          hero.fromTo(lines, { strokeDasharray: 1, strokeDashoffset: 1 }, { strokeDashoffset: 0, duration: 1.8, ease: "power2.inOut", stagger: 0.12 }, 0.2);
        } else {
          gsap.fromTo(
            lines,
            { strokeDasharray: 1, strokeDashoffset: 1 },
            { strokeDashoffset: 0, ease: "none", stagger: 0.08, scrollTrigger: { trigger: section, start: "top 90%", end: "center 45%", scrub: 0.6 } },
          );
        }

        // A pulse runs along each line while the section scrolls past, each a little behind the last.
        // The hero is on screen from the start, so its pulses begin with the first scroll.
        const flow = gsap.timeline({ scrollTrigger: { trigger: section, start: inHero ? "top top" : "top bottom", end: "bottom top", scrub: 0.8 } });
        pulses.forEach((pulse, i) => {
          flow.fromTo(pulse, { strokeDashoffset: 0.07 }, { strokeDashoffset: -1, ease: "none", duration: 1 }, i * 0.12);
        });

        // A little depth: the lines drift slower than the page.
        gsap.fromTo(svg, { y: 28 }, { y: -28, ease: "none", scrollTrigger: { trigger: section, start: "top bottom", end: "bottom top", scrub: true } });
      });

      ScrollTrigger.refresh();
      return () => hero.kill();
    });

    return () => mm.revert();
  }, []);

  return null;
}
