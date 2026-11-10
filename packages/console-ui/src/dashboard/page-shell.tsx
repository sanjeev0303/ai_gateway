import { cn } from "../lib/utils";

interface PageShellProps {
  title: string;
  description?: string;
  /** Right-aligned header slot: range picker, export button, primary action. */
  actions?: React.ReactNode;
  /** Optional breadcrumb or status line rendered above the title. */
  eyebrow?: React.ReactNode;
  className?: string;
  children: React.ReactNode;
}

/**
 * Standard frame for every `/app` page: heading block, action slot, then
 * content. Pages supply only their own body so layout stays consistent.
 */
export function PageShell({
  title,
  description,
  actions,
  eyebrow,
  className,
  children,
}: PageShellProps) {
  return (
    <div className={cn("flex min-w-0 flex-col gap-6", className)}>
      <header className="flex flex-wrap items-end justify-between gap-x-6 gap-y-4 pt-1 pb-1">
        <div className="flex min-w-0 flex-col gap-1.5">
          {eyebrow}
          <h1 className="truncate text-[1.625rem] leading-tight font-semibold tracking-[-0.025em] text-foreground">
            {title}
          </h1>
          {description && (
            <p className="max-w-[68ch] text-sm leading-relaxed text-muted-foreground">
              {description}
            </p>
          )}
        </div>
        {actions && (
          <div className="flex max-w-full shrink-0 flex-wrap items-center gap-2">
            {actions}
          </div>
        )}
      </header>
      {children}
    </div>
  );
}
