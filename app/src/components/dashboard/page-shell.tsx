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
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div className="min-w-0 space-y-1">
          {eyebrow}
          <h1 className="truncate text-2xl font-semibold tracking-tight">
            {title}
          </h1>
          {description && (
            <p className="max-w-2xl text-sm text-muted-foreground">
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
