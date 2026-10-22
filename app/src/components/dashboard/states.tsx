"use client";

import * as React from "react";
import { WifiOff, RefreshCw, Lock, FileX, Inbox, Search, AlertCircle, Home, ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardTitle, CardDescription } from "@/components/ui/card";
import { useRouter } from "next/navigation";
import { cn } from "@/lib/utils";

interface NetworkFailureProps {
  message?: string;
  onRetry?: () => void;
  retryLabel?: string;
  showGoHome?: boolean;
  className?: string;
}

export function NetworkFailure({
  message = "Unable to connect to the server. Please check your internet connection and try again.",
  onRetry,
  retryLabel = "Try Again",
  showGoHome = true,
  className,
}: NetworkFailureProps) {
  const router = useRouter();

  return (
    <Card className={cn("w-full max-w-md mx-auto rounded-2xl border border-border/50 bg-card/80 backdrop-blur-sm shadow-sm", className)}>
      <CardContent className="pt-8 pb-8 text-center">
        <div className="mx-auto mb-4 flex size-14 items-center justify-center rounded-2xl bg-destructive/10 text-destructive border border-destructive/20">
          <WifiOff className="size-7" aria-hidden />
        </div>
        <CardTitle className="text-base font-bold mb-1.5 text-foreground">Connection Problem</CardTitle>
        <CardDescription className="mb-6 text-xs leading-relaxed max-w-xs mx-auto">{message}</CardDescription>
        <div className="flex flex-col sm:flex-row gap-2.5 justify-center">
          <Button
            size="sm"
            onClick={onRetry ?? (() => window.location.reload())}
            className="w-full sm:w-auto font-medium"
          >
            <RefreshCw className="size-3.5 mr-2" aria-hidden />
            {retryLabel}
          </Button>
          {showGoHome && (
            <Button size="sm" variant="outline" onClick={() => router.push("/")} className="w-full sm:w-auto">
              <Home className="size-3.5 mr-2" aria-hidden />
              Go Home
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

interface PermissionDeniedProps {
  message?: string;
  requiredPermission?: string;
  onGoBack?: () => void;
  goBackLabel?: string;
  showGoHome?: boolean;
  className?: string;
}

export function PermissionDenied({
  message = "You don't have permission to access this resource.",
  requiredPermission,
  onGoBack,
  goBackLabel = "Go Back",
  showGoHome = true,
  className,
}: PermissionDeniedProps) {
  const router = useRouter();

  return (
    <Card className={cn("w-full max-w-md mx-auto rounded-2xl border border-border/50 bg-card/80 backdrop-blur-sm shadow-sm", className)}>
      <CardContent className="pt-8 pb-8 text-center">
        <div className="mx-auto mb-4 flex size-14 items-center justify-center rounded-2xl bg-destructive/10 text-destructive border border-destructive/20">
          <Lock className="size-7" aria-hidden />
        </div>
        <CardTitle className="text-base font-bold mb-1.5 text-foreground">Access Denied</CardTitle>
        <CardDescription className="mb-2 text-xs leading-relaxed max-w-xs mx-auto">{message}</CardDescription>
        {requiredPermission && (
          <div className="mb-6 inline-block rounded-md bg-muted px-2.5 py-1 text-xs font-mono text-muted-foreground border">
            Required: {requiredPermission}
          </div>
        )}
        <div className="flex flex-col sm:flex-row gap-2.5 justify-center">
          <Button
            size="sm"
            variant="outline"
            onClick={onGoBack ?? (() => router.back())}
            className="w-full sm:w-auto"
          >
            <ArrowLeft className="size-3.5 mr-2" aria-hidden />
            {goBackLabel}
          </Button>
          {showGoHome && (
            <Button size="sm" onClick={() => router.push("/")} className="w-full sm:w-auto">
              <Home className="size-3.5 mr-2" aria-hidden />
              Dashboard
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

interface NotFoundProps {
  message?: string;
  resourceType?: string;
  onGoBack?: () => void;
  goBackLabel?: string;
  showGoHome?: boolean;
  className?: string;
}

export function NotFound({
  message = "The requested resource could not be found.",
  resourceType,
  onGoBack,
  goBackLabel = "Go Back",
  showGoHome = true,
  className,
}: NotFoundProps) {
  const router = useRouter();

  return (
    <Card className={cn("w-full max-w-md mx-auto rounded-2xl border border-border/50 bg-card/80 backdrop-blur-sm shadow-sm", className)}>
      <CardContent className="pt-8 pb-8 text-center">
        <div className="mx-auto mb-4 flex size-14 items-center justify-center rounded-2xl bg-muted text-muted-foreground border">
          <FileX className="size-7" aria-hidden />
        </div>
        <CardTitle className="text-base font-bold mb-1.5 text-foreground">Not Found</CardTitle>
        <CardDescription className="mb-2 text-xs leading-relaxed max-w-xs mx-auto">{message}</CardDescription>
        {resourceType && (
          <p className="mb-6 text-xs text-muted-foreground">
            {resourceType} not found
          </p>
        )}
        <div className="flex flex-col sm:flex-row gap-2.5 justify-center">
          <Button
            size="sm"
            variant="outline"
            onClick={onGoBack ?? (() => router.back())}
            className="w-full sm:w-auto"
          >
            <ArrowLeft className="size-3.5 mr-2" aria-hidden />
            {goBackLabel}
          </Button>
          {showGoHome && (
            <Button size="sm" onClick={() => router.push("/")} className="w-full sm:w-auto">
              <Home className="size-3.5 mr-2" aria-hidden />
              Dashboard
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

interface EmptyStateProps {
  title: string;
  description?: string;
  icon?: React.ReactNode;
  action?: {
    label: string;
    onClick: () => void;
    variant?: "default" | "outline" | "secondary";
  };
  secondaryAction?: {
    label: string;
    onClick: () => void;
  };
  className?: string;
  illustration?: React.ReactNode;
}

export function EmptyState({
  title,
  description,
  icon,
  action,
  secondaryAction,
  className,
  illustration,
}: EmptyStateProps) {
  return (
    <Card className={cn("w-full rounded-2xl border border-border/50 bg-card/60 backdrop-blur-sm shadow-xs", className)}>
      <CardContent className="pt-10 pb-12 text-center">
        {illustration || (
          <div className="mx-auto mb-4 flex size-14 items-center justify-center rounded-2xl bg-muted/60 text-muted-foreground border">
            {icon || <Inbox className="size-7 opacity-60" aria-hidden />}
          </div>
        )}
        <CardTitle className="text-base font-bold mb-1 text-foreground">{title}</CardTitle>
        {description && <CardDescription className="mb-6 text-xs leading-relaxed max-w-md mx-auto">{description}</CardDescription>}
        {(action || secondaryAction) && (
          <div className="flex flex-col sm:flex-row gap-2.5 justify-center">
            {action && (
              <Button
                size="sm"
                onClick={action.onClick}
                variant={action.variant || "default"}
                className="w-full sm:w-auto font-medium"
              >
                {action.label}
              </Button>
            )}
            {secondaryAction && (
              <Button
                size="sm"
                variant="outline"
                onClick={secondaryAction.onClick}
                className="w-full sm:w-auto font-medium"
              >
                {secondaryAction.label}
              </Button>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

export function EmptyListState({
  title = "No items yet",
  description = "Get started by creating your first item.",
  actionLabel = "Create Item",
  onAction,
  icon,
  className,
}: {
  title?: string;
  description?: string;
  actionLabel?: string;
  onAction?: () => void;
  icon?: React.ReactNode;
  className?: string;
}) {
  return (
    <EmptyState
      title={title}
      description={description}
      icon={icon || <Inbox className="size-7 opacity-60" aria-hidden />}
      action={onAction ? { label: actionLabel, onClick: onAction } : undefined}
      className={className}
    />
  );
}

export function EmptySearchState({
  query,
  title = "No results found",
  description,
  actionLabel,
  onAction,
  className,
}: {
  query?: string;
  title?: string;
  description?: string;
  actionLabel?: string;
  onAction?: () => void;
  className?: string;
}) {
  return (
    <EmptyState
      title={title}
      description={description || (query ? `No results for "${query}"` : "Try adjusting your search or filters.")}
      icon={<Search className="size-7 opacity-60" aria-hidden />}
      action={onAction ? { label: actionLabel || "Clear Search", onClick: onAction, variant: "outline" } : undefined}
      className={className}
    />
  );
}

export function EmptyDataState({
  title = "No data available",
  description = "There's no data to display for the selected time range.",
  actionLabel,
  onAction,
  className,
}: {
  title?: string;
  description?: string;
  actionLabel?: string;
  onAction?: () => void;
  className?: string;
}) {
  return (
    <EmptyState
      title={title}
      description={description}
      icon={<AlertCircle className="size-7 opacity-60" aria-hidden />}
      action={onAction ? { label: actionLabel || "Refresh", onClick: onAction, variant: "outline" } : undefined}
      className={className}
    />
  );
}