import type { components } from "@ai-governance/contracts";

export type HealthStatus = components["schemas"]["HealthStatus"];

export interface ApiClient {
  readonly livez: () => Promise<HealthStatus>;
  readonly readyz: () => Promise<HealthStatus>;
  readonly healthz: () => Promise<HealthStatus>;
}

export interface CreateApiClientOptions {
  readonly baseUrl: string;
}

const HEALTH_PATHS = ["/livez", "/readyz", "/healthz"] as const;

export function createApiClient(options: CreateApiClientOptions): ApiClient {
  const requestJson = async (path: (typeof HEALTH_PATHS)[number]): Promise<HealthStatus> => {
    const response = await fetch(new URL(path, options.baseUrl));
    if (!response.ok) {
      throw new Error(`Health probe ${path} failed with HTTP ${response.status}`);
    }
    // Trusted boundary: the Go backend is the only producer of this JSON,
    // and its shape is enforced by the generated OpenAPI types above.
    return (await response.json()) as HealthStatus;
  };

  return {
    livez: () => requestJson("/livez"),
    readyz: () => requestJson("/readyz"),
    healthz: () => requestJson("/healthz"),
  };
}