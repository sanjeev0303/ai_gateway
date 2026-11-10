import { cn } from "@ai-governance/console-ui/lib/utils";

const APP_NAME = "Tollseal";

interface LogoProps {
  /** Hide the wordmark and render the mark alone. */
  markOnly?: boolean;
  className?: string;
}

/**
 * The Tollseal mark: a seal (an outer ring with a fine inner guilloche ring)
 * around a T drawn as a toll barrier, a bar across a single post. Every request
 * passes the barrier and leaves sealed. Drawn in the action green so it reads
 * the same on security paper and on night intaglio.
 */
export function LogoMark({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 32 32"
      className={cn("size-7 shrink-0 text-primary", className)}
      role="img"
      aria-label={`${APP_NAME} logo`}
      fill="none"
    >
      <circle cx="16" cy="16" r="13.4" stroke="currentColor" strokeWidth="2.2" />
      <circle cx="16" cy="16" r="10.2" stroke="currentColor" strokeWidth="0.9" strokeDasharray="1.6 1.4" opacity="0.6" />
      <path d="M10.4 12.4H21.6M16 12.4V22.2" stroke="currentColor" strokeWidth="2.6" strokeLinecap="round" />
    </svg>
  );
}

export function Logo({ markOnly = false, className }: LogoProps) {
  if (markOnly) return <LogoMark />;
  return (
    <span className={cn("inline-flex items-center gap-2.5", className)}>
      <LogoMark />
      <span translate="no" className="font-display text-[1.0625rem] tracking-[-0.01em]">{APP_NAME}</span>
    </span>
  );
}
