import { LogoMark } from "@/components/logo";
import { cn } from "@ai-governance/console-ui/lib/utils";

/**
 * The gateway as an object: a stack of foil plates (authenticate, guard,
 * route, record) under an intaglio cap that carries the seal. Pure CSS 3D, so it
 * stays sharp at any size and themes with the tokens. Decorative only.
 */
export function GatewayPrism({ className, size = 168 }: { className?: string; size?: number }) {
  return (
    <div
      aria-hidden
      className={cn("gateway-prism", className)}
      style={{ "--prism-size": `${size}px` } as React.CSSProperties}
    >
      <div className="gateway-prism__glow" />
      <div className="gateway-prism__float">
        <div className="gateway-prism__stack">
          <span className="gateway-prism__plate" style={{ "--z": 0 } as React.CSSProperties} />
          <span className="gateway-prism__plate" style={{ "--z": 1 } as React.CSSProperties} />
          <span className="gateway-prism__plate" style={{ "--z": 2 } as React.CSSProperties} />
          <span className="gateway-prism__plate gateway-prism__cap" style={{ "--z": 3 } as React.CSSProperties}>
            <LogoMark className="size-[46%] text-[#e2c15f]" />
          </span>
        </div>
      </div>
    </div>
  );
}
