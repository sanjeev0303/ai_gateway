"use client";

import * as React from "react";
import { AlertTriangle, RefreshCw, WifiOff, Lock, FileX, ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
  errorType: "network" | "permission" | "not-found" | "unknown";
  retryCount: number;
}

interface ErrorBoundaryProps {
  children: React.ReactNode;
  fallback?: React.ReactNode;
  onError?: (error: Error, errorInfo: React.ErrorInfo) => void;
  maxRetries?: number;
  retryDelay?: number;
}

type ErrorType = "network" | "permission" | "not-found" | "unknown";

const MAX_RETRIES = 3;
const RETRY_DELAY = 1000;

function classifyError(error: Error): ErrorType {
  const message = error.message.toLowerCase();
  if (
    message.includes("network") ||
    message.includes("fetch") ||
    message.includes("connection") ||
    message.includes("timeout") ||
    message.includes("econnrefused") ||
    message.includes("enotfound")
  ) {
    return "network";
  }
  if (
    message.includes("permission") ||
    message.includes("403") ||
    message.includes("forbidden") ||
    message.includes("unauthorized") ||
    message.includes("401")
  ) {
    return "permission";
  }
  if (
    message.includes("404") ||
    message.includes("not found") ||
    message.includes("not_found")
  ) {
    return "not-found";
  }
  return "unknown";
}

function getErrorConfig(type: ErrorType) {
  switch (type) {
    case "network":
      return {
        icon: WifiOff,
        title: "Connection Problem",
        description: "Unable to reach the server. Please check your internet connection.",
        actionLabel: "Retry",
      };
    case "permission":
      return {
        icon: Lock,
        title: "Access Denied",
        description: "You don't have permission to view this content.",
        actionLabel: "Go Back",
      };
    case "not-found":
      return {
        icon: FileX,
        title: "Not Found",
        description: "The requested resource could not be found.",
        actionLabel: "Go Back",
      };
    default:
      return {
        icon: AlertTriangle,
        title: "Something Went Wrong",
        description: "An unexpected error occurred. Please try again.",
        actionLabel: "Retry",
      };
  }
}

export class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = {
    hasError: false,
    error: null,
    errorType: "unknown",
    retryCount: 0,
  };

  static getDerivedStateFromError(error: Error): Partial<ErrorBoundaryState> {
    return {
      hasError: true,
      error,
      errorType: classifyError(error),
    };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error("ErrorBoundary caught:", error, errorInfo);
    this.props.onError?.(error, errorInfo);
  }

  handleRetry = () => {
    const { retryCount } = this.state;
    const maxRetries = this.props.maxRetries ?? MAX_RETRIES;

    if (retryCount >= maxRetries) {
      return;
    }

    this.setState({ retryCount: retryCount + 1 });

    setTimeout(() => {
      this.setState({
        hasError: false,
        error: null,
      });
    }, this.props.retryDelay ?? RETRY_DELAY);
  };

  handleGoBack = () => {
    if (window.history.length > 1) {
      window.history.back();
    } else {
      window.location.href = "/";
    }
  };

  render() {
    const { hasError, error, errorType, retryCount } = this.state;
    const { children, fallback } = this.props;

    if (hasError) {
      if (fallback) {
        return fallback;
      }

      const config = getErrorConfig(errorType);
      const Icon = config.icon;
      const canRetry = errorType !== "permission" && errorType !== "not-found" && retryCount < (this.props.maxRetries ?? MAX_RETRIES);

      return (
        <Card className="w-full">
          <CardHeader className="text-center py-6">
            <Icon className="size-12 mx-auto text-destructive/50 mb-3" aria-hidden />
            <CardTitle className="text-lg">{config.title}</CardTitle>
            <CardDescription>{config.description}</CardDescription>
            {error && process.env.NODE_ENV === "development" && (
              <details className="mt-4 text-left text-xs text-muted-foreground">
                <summary className="cursor-pointer font-medium mb-1">Error details</summary>
                <pre className="whitespace-pre-wrap break-all p-2 rounded bg-muted">{error.message}</pre>
              </details>
            )}
          </CardHeader>
          <CardContent className="pt-0 pb-6">
            <div className="flex flex-col sm:flex-row gap-3 justify-center">
              {canRetry && (
                <Button onClick={this.handleRetry} className="w-full sm:w-auto" disabled={retryCount > 0}>
                  {retryCount > 0 ? (
                    <>
                      <RefreshCw className="size-4 mr-2 animate-spin" aria-hidden />
                      Retrying... ({retryCount}/{(this.props.maxRetries ?? MAX_RETRIES)})
                    </>
                  ) : (
                    <>
                      <RefreshCw className="size-4 mr-2" aria-hidden />
                      {config.actionLabel}
                    </>
                  )}
                </Button>
              )}
              <Button
                variant="outline"
                onClick={this.handleGoBack}
                className="w-full sm:w-auto"
              >
                <ArrowLeft className="size-4 mr-2" aria-hidden />
                Go Back
              </Button>
            </div>
          </CardContent>
        </Card>
      );
    }

    return <>{children}</>;
  }
}

interface WithErrorBoundaryProps {
  children: React.ReactNode;
  errorType?: ErrorType;
  fallback?: React.ReactNode;
  maxRetries?: number;
  retryDelay?: number;
}

export function withErrorBoundary<P extends object>(
  Component: React.ComponentType<P>,
  errorBoundaryProps?: Omit<ErrorBoundaryProps, "children">
) {
  return function WithErrorBoundary(props: P) {
    return (
      <ErrorBoundary {...errorBoundaryProps}>
        <Component {...props} />
      </ErrorBoundary>
    );
  };
}

export function AsyncErrorBoundary({
  children,
  fallback,
  onError,
  maxRetries = MAX_RETRIES,
  retryDelay = RETRY_DELAY,
}: ErrorBoundaryProps) {
  return (
    <ErrorBoundary
      fallback={fallback}
      onError={onError}
      maxRetries={maxRetries}
      retryDelay={retryDelay}
    >
      {children}
    </ErrorBoundary>
  );
}

export { classifyError, getErrorConfig };
export type { ErrorType };