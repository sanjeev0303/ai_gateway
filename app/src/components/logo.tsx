import { APP_NAME } from "@/lib/constants";
import { cn } from "@/lib/utils";

interface LogoProps {
  /** Hide the wordmark and render the mark alone. */
  markOnly?: boolean;
  className?: string;
}

/** Shield-and-beam mark; inherits text colour so it themes automatically. */
export function LogoMark({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      className={cn("size-6 shrink-0", className)}
      role="img"
      aria-label={`${APP_NAME} logo`}
      fill="none"
    >
      <path
        d="M12 2.5 4.75 5.6v6.15c0 4.53 3.02 8.44 7.25 9.75 4.23-1.31 7.25-5.22 7.25-9.75V5.6L12 2.5Z"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinejoin="round"
      />
      <path
        d="M12 7.4v9.2M8.9 10.2h6.2"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinecap="round"
      />
    </svg>
  );
}

export function Logo({ markOnly = false, className }: LogoProps) {
  if (markOnly) return <LogoMark className={className} />;
  return (
    <span className={cn("inline-flex items-center gap-2", className)}>
      <LogoMark />
      <span className="text-base font-semibold tracking-tight">{APP_NAME}</span>
    </span>
  );
}
