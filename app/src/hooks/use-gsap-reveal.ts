"use client";

import { useEffect, useRef } from "react";
import gsap from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";

gsap.registerPlugin(ScrollTrigger);

/**
 * GSAP-based scroll reveal hook.
 * Elements with [data-reveal] are animated on scroll.
 * Call once per section container ref.
 */
export function useGsapReveal() {
  const containerRef = useRef<HTMLElement>(null);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const targets = container.querySelectorAll<HTMLElement>("[data-reveal]");
    if (!targets.length) return;

    const ctx = gsap.context(() => {
      targets.forEach((el) => {
        const dir = el.dataset.reveal ?? "up";
        const delay = parseFloat(el.dataset.delay ?? "0");

        const from: gsap.TweenVars = {
          opacity: 0,
          duration: 0.85,
          ease: "expo.out",
          delay,
        };

        if (dir === "up" || dir === "") from.y = 36;
        if (dir === "right") from.x = -28;
        if (dir === "left") from.x = 28;
        if (dir === "scale") { from.scale = 0.96; }

        gsap.from(el, {
          ...from,
          scrollTrigger: {
            trigger: el,
            start: "top 88%",
            toggleActions: "play none none none",
          },
        });
      });
    }, container);

    return () => ctx.revert();
  }, []);

  return containerRef;
}
