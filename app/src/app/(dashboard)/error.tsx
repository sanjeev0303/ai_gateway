"use client";

import * as React from "react";
import { AlertTriangle, RefreshCw, ArrowLeft, Home } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { useRouter, usePathname } from "next/navigation";

interface DashboardErrorProps {
  error: Error & { digest?: string };
  reset: () => void;
}

export default function DashboardError({ error, reset }: DashboardErrorProps) {
  const router = useRouter();
  const pathname = usePathname();

  console.error("Dashboard error:", error);

  const isAuthError = error.message.includes("unauthorized") || error.message.includes("401");
  const isNetworkError = error.message.includes("network") || error.message.includes("fetch");
  const isPermissionError = error.message.includes("permission") || error.message.includes("403");

  const getErrorMessage = () => {
    if (isAuthError) return "Your session has expired. Please sign in again.";
    if (isNetworkError) return "Network error. Please check your connection and try again.";
    if (isPermissionError) return "You don't have permission to access this page.";
    return "Something went wrong. Please try again.";
  };

  const getErrorAction = () => {
    if (isAuthError) {
      return (
        <Button onClick={() => window.location.href = "/signin"} className="w-full sm:w-auto">
          Sign in
        </Button>
      );
    }
    if (isNetworkError || !isPermissionError) {
      return (
        <Button onClick={reset} className="w-full sm:w-auto">
          <RefreshCw className="size-4 mr-2" aria-hidden />
          Try again
        </Button>
      );
    }
    return (
      <Button variant="outline" onClick={() => router.back()} className="w-full sm:w-auto">
        <ArrowLeft className="size-4 mr-2" aria-hidden />
        Go back
      </Button>
    );
  };

  return (
    <div className="flex h-full items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <CardContent className="pt-6">
          <div className="text-center space-y-4">
            <AlertTriangle className="size-16 mx-auto text-destructive/50" aria-hidden />
            <div className="space-y-2">
              <h1 className="text-xl font-semibold text-foreground">Error</h1>
              <p className="text-muted-foreground">{getErrorMessage()}</p>
            </div>
            <div className="flex flex-col sm:flex-row gap-3 justify-center">
              {getErrorAction()}
              <Button variant="outline" onClick={() => router.push("/")} className="w-full sm:w-auto">
                <Home className="size-4 mr-2" aria-hidden />
                Dashboard
              </Button>
            </div>
            {process.env.NODE_ENV === "development" && (
              <details className="text-left text-xs text-muted-foreground mt-4 p-2 rounded bg-muted">
                <summary className="cursor-pointer font-medium mb-1">Error details (development)</summary>
                <pre className="whitespace-pre-wrap break-all">{error.message}</pre>
                {error.digest && <p className="mt-1">Digest: {error.digest}</p>}
                <p className="mt-1 font-mono text-xs">Path: {pathname}</p>
              </details>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}