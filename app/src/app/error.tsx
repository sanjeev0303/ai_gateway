"use client";

import * as React from "react";
import { AlertCircle, RefreshCw, Home } from "lucide-react";
import { Button } from "@/components/ui/button";

interface ErrorProps {
  error: Error & { digest?: string };
  reset: () => void;
}

export default function GlobalError({ error, reset }: ErrorProps) {
  console.error("Global error:", error);

  return (
    <html lang="en">
      <body className="min-h-screen flex items-center justify-center bg-background p-4">
        <div className="max-w-md w-full text-center space-y-4">
          <AlertCircle className="size-16 mx-auto text-destructive/50" aria-hidden />
          <div className="space-y-2">
            <h1 className="text-2xl font-bold text-foreground">Something went wrong</h1>
            <p className="text-muted-foreground">
              We encountered an unexpected error. Please try again or go back to the homepage.
            </p>
          </div>
          <div className="flex flex-col sm:flex-row gap-3 justify-center">
            <Button onClick={reset} className="w-full sm:w-auto">
              <RefreshCw className="size-4 mr-2" aria-hidden />
              Try again
            </Button>
            <Button variant="outline" onClick={() => window.location.href = "/"} className="w-full sm:w-auto">
              <Home className="size-4 mr-2" aria-hidden />
              Go home
            </Button>
          </div>
          {process.env.NODE_ENV === "development" && (
            <details className="text-left text-xs text-muted-foreground mt-4 p-2 rounded bg-muted">
              <summary className="cursor-pointer font-medium mb-1">Error details (development)</summary>
              <pre className="whitespace-pre-wrap break-all">{error.message}</pre>
              {error.digest && <p className="mt-1">Digest: {error.digest}</p>}
            </details>
          )}
        </div>
      </body>
    </html>
  );
}