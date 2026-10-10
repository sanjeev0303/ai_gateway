"use client";

import * as React from "react";
import { WifiOff, RefreshCw, Lock, FileX, Inbox, Search, Plus, AlertCircle, Home, ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { useRouter } from "next/navigation";

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
    <Card className={`w-full max-w-md mx-auto ${className || ""}`}>
      <CardContent className="pt-6 pb-8 text-center">
        <WifiOff className="size-16 mx-auto text-destructive/50 mb-4" aria-hidden />
        <CardTitle className="text-lg mb-2">Connection Problem</CardTitle>
        <CardDescription className="mb-6">{message}</CardDescription>
        <div className="flex flex-col sm:flex-row gap-3 justify-center">
          <Button
            onClick={onRetry ?? (() => window.location.reload())}
            className="w-full sm:w-auto"
          >
            <RefreshCw className="size-4 mr-2" aria-hidden />
            {retryLabel}
          </Button>
          {showGoHome && (
            <Button variant="outline" onClick={() => router.push("/")} className="w-full sm:w-auto">
              <Home className="size-4 mr-2" aria-hidden />
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
    <Card className={`w-full max-w-md mx-auto ${className || ""}`}>
      <CardContent className="pt-6 pb-8 text-center">
        <Lock className="size-16 mx-auto text-destructive/50 mb-4" aria-hidden />
        <CardTitle className="text-lg mb-2">Access Denied</CardTitle>
        <CardDescription className="mb-2">{message}</CardDescription>
        {requiredPermission && (
          <CardDescription className="mb-6 text-sm font-mono text-muted-foreground">
            Required: {requiredPermission}
          </CardDescription>
        )}
        <div className="flex flex-col sm:flex-row gap-3 justify-center">
          <Button
            variant="outline"
            onClick={onGoBack ?? (() => router.back())}
            className="w-full sm:w-auto"
          >
            <ArrowLeft className="size-4 mr-2" aria-hidden />
            {goBackLabel}
          </Button>
          {showGoHome && (
            <Button onClick={() => router.push("/")} className="w-full sm:w-auto">
              <Home className="size-4 mr-2" aria-hidden />
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
    <Card className={`w-full max-w-md mx-auto ${className || ""}`}>
      <CardContent className="pt-6 pb-8 text-center">
        <FileX className="size-16 mx-auto text-destructive/50 mb-4" aria-hidden />
        <CardTitle className="text-lg mb-2">Not Found</CardTitle>
        <CardDescription className="mb-2">{message}</CardDescription>
        {resourceType && (
          <CardDescription className="mb-6 text-sm text-muted-foreground">
            {resourceType} not found
          </CardDescription>
        )}
        <div className="flex flex-col sm:flex-row gap-3 justify-center">
          <Button
            variant="outline"
            onClick={onGoBack ?? (() => router.back())}
            className="w-full sm:w-auto"
          >
            <ArrowLeft className="size-4 mr-2" aria-hidden />
            {goBackLabel}
          </Button>
          {showGoHome && (
            <Button onClick={() => router.push("/")} className="w-full sm:w-auto">
              <Home className="size-4 mr-2" aria-hidden />
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
    <Card className={`w-full ${className || ""}`}>
      <CardContent className="pt-8 pb-10 text-center">
        {illustration || icon || (
          <Inbox className="size-16 mx-auto text-muted-foreground/50 mb-4" aria-hidden />
        )}
        <CardTitle className="text-lg mb-2">{title}</CardTitle>
        {description && <CardDescription className="mb-6">{description}</CardDescription>}
        {(action || secondaryAction) && (
          <div className="flex flex-col sm:flex-row gap-3 justify-center">
            {action && (
              <Button
                onClick={action.onClick}
                variant={action.variant || "default"}
                className="w-full sm:w-auto"
              >
                {action.label}
              </Button>
            )}
            {secondaryAction && (
              <Button
                variant="outline"
                onClick={secondaryAction.onClick}
                className="w-full sm:w-auto"
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
      icon={icon || <Inbox className="size-16 mx-auto text-muted-foreground/50 mb-4" aria-hidden />}
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
      icon={<Search className="size-16 mx-auto text-muted-foreground/50 mb-4" aria-hidden />}
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
      icon={<AlertCircle className="size-16 mx-auto text-muted-foreground/50 mb-4" aria-hidden />}
      action={onAction ? { label: actionLabel || "Refresh", onClick: onAction, variant: "outline" } : undefined}
      className={className}
    />
  );
}