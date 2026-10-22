import { cn } from "@/lib/utils";

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
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-border/30 pb-5">
        <div className="min-w-0 space-y-1.5">
          {eyebrow}
          <h1 className="truncate text-2xl font-bold tracking-tight text-foreground animate-in fade-in slide-in-from-bottom-1 duration-300">
            {title}
          </h1>
          {description && (
            <p className="max-w-2xl text-sm leading-relaxed text-muted-foreground">
              {description}
            </p>
          )}
        </div>
        {actions && (
          <div className="flex shrink-0 flex-wrap items-center gap-2">
            {actions}
          </div>
        )}
      </header>
      {children}
    </div>
  );
}
