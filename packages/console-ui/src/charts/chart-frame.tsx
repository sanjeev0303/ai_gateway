"use client";

import { useEffect, useRef } from "react";

const LABEL_PX = 12;

/**
 * Charts draw in a fixed 720-unit coordinate space and scale to their card, which
 * would scale their labels too: huge on a wide screen, unreadable on a phone. This
 * keeps labels at a true 12px by feeding the current scale into their font size.
 * Below 520px the plot scrolls sideways rather than clipping its axis labels.
 */
export function ChartFrame({ viewWidth = 720, children }: { viewWidth?: number; children: React.ReactNode }) {
  const outer = useRef<HTMLDivElement>(null);
  const inner = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const scroller = outer.current;
    const plot = inner.current;
    if (!scroller || !plot) return;
    // Time runs left to right, so a phone opens on the latest days.
    scroller.scrollLeft = scroller.scrollWidth;
    const observer = new ResizeObserver(([entry]) => {
      const width = entry?.contentRect.width;
      if (width) plot.style.setProperty("--chart-text", `${(LABEL_PX * viewWidth) / width}px`);
      // Only a plot that actually scrolls earns a stop in the tab order.
      if (scroller.scrollWidth > scroller.clientWidth) scroller.tabIndex = 0;
      else scroller.removeAttribute("tabindex");
    });
    observer.observe(plot);
    return () => observer.disconnect();
  }, [viewWidth]);

  return (
    <div ref={outer} className="overflow-x-auto overscroll-x-contain rounded-lg pr-4 focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:outline-none">
      <div ref={inner} className="mx-auto w-full max-w-[920px] min-w-[520px] [&_svg]:overflow-visible [&_text]:[font-size:var(--chart-text,12px)]">
        {children}
      </div>
    </div>
  );
}
