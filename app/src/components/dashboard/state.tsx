import type { LucideIcon } from "lucide-react";
import { Inbox, Loader2, TriangleAlert } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

interface EmptyStateProps {
  title: string;
  description?: string;
  icon?: LucideIcon;
  /** Filters a list down to nothing, so it offers the way back. */
  action?: { label: string; onClick: () => void };
  className?: string;
}

/** No rows to show. Kept icon-on-top and unboxed so it drops into tables and cards alike. */
export function EmptyState({
  title,
  description,
  icon: Icon = Inbox,
  action,
  className,
}: EmptyStateProps) {
  return (
    <div className={cn("flex flex-col items-center gap-3 px-6 py-12 text-center", className)}>
      <span className="flex size-10 items-center justify-center rounded-full bg-muted text-muted-foreground">
        <Icon className="size-5" aria-hidden />
      </span>
      <div className="space-y-1">
        <p className="text-sm font-medium">{title}</p>
        {description && (
          <p className="mx-auto max-w-sm text-sm text-muted-foreground text-pretty">
            {description}
          </p>
        )}
      </div>
      {action && (
        <Button variant="outline" size="sm" onClick={action.onClick}>
          {action.label}
        </Button>
      )}
    </div>
  );
}

interface ErrorStateProps {
  title?: string;
  description?: string;
  onRetry?: () => void;
  className?: string;
}

/** Recoverable failure with a retry affordance; the copy names the action, not the code. */
export function ErrorState({
  title = "Something went wrong",
  description = "We could not load this data. Try again, and check the event log if it keeps failing.",
  onRetry,
  className,
}: ErrorStateProps) {
  return (
    <div
      role="alert"
      className={cn("flex flex-col items-center gap-3 px-6 py-12 text-center", className)}
    >
      <span className="flex size-10 items-center justify-center rounded-full bg-destructive/10 text-destructive">
        <TriangleAlert className="size-5" aria-hidden />
      </span>
      <div className="space-y-1">
        <p className="text-sm font-medium">{title}</p>
        <p className="mx-auto max-w-sm text-sm text-muted-foreground text-pretty">
          {description}
        </p>
      </div>
      {onRetry && (
        <Button variant="outline" size="sm" onClick={onRetry}>
          Try again
        </Button>
      )}
    </div>
  );
}

interface LoadingStateProps {
  /** Row count for skeleton lists; ignored when `label` is set. */
  rows?: number;
  label?: string;
  className?: string;
}

/**
 * Placeholder shown while a client-side panel loads. `label` is announced to
 * screen readers; bare skeletons announce nothing, so prefer it on real waits.
 */
export function LoadingState({ rows = 3, label = "Loading", className }: LoadingStateProps) {
  return (
    <div
      role="status"
      aria-live="polite"
      aria-busy="true"
      className={cn("space-y-2", className)}
    >
      <span className="sr-only">{label}</span>
      {Array.from({ length: Math.max(rows, 1) }, (_, index) => (
        <Skeleton key={index} className="h-12 w-full" />
      ))}
    </div>
  );
}

/** Inline spinner for a control that waits without unmounting, e.g. a submit button. */
export function Spinner({ className }: { className?: string }) {
  return <Loader2 className={cn("size-4 animate-spin", className)} aria-hidden />;
}
