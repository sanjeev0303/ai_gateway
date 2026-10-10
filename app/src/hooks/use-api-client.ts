"use client";

import * as React from "react";
import { createApiClient, type ApiClient } from "@ai-governance/api-client";
import { useSession } from "@/lib/session-context";

let apiClientInstance: ApiClient | null = null;

export function useApiClient(): ApiClient {
  const session = useSession();

  return React.useMemo(() => {
    if (session.status !== "authenticated") {
      return createApiClient({
        baseUrl: process.env.NEXT_PUBLIC_CONTROL_API_URL || "http://localhost:8081",
      });
    }

    if (!apiClientInstance) {
      apiClientInstance = createApiClient({
        baseUrl: process.env.NEXT_PUBLIC_CONTROL_API_URL || "http://localhost:8081",
        credentials: "include",
      });
    }
    return apiClientInstance;
  }, [session.status]);
}

export function getApiClient(): ApiClient {
  if (!apiClientInstance) {
    apiClientInstance = createApiClient({
      baseUrl: process.env.NEXT_PUBLIC_CONTROL_API_URL || "http://localhost:8081",
      credentials: "include",
    });
  }
  return apiClientInstance;
}

export function resetApiClient() {
  apiClientInstance = null;
}