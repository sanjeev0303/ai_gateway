

export interface CreateApiClientOptions {
  readonly baseUrl: string;
}

export function createApiClient(options: CreateApiClientOptions) {
  const requestJson = async (path: string, method: string = "GET", body?: any) => {
    const init: RequestInit = {
      method,
    };
    if (body) {
      init.headers = { "Content-Type": "application/json" };
      init.body = JSON.stringify(body);
    }
    const response = await fetch(new URL(path, options.baseUrl), init);
    if (!response.ok) {
      throw new Error(`API ${method} ${path} failed with HTTP ${response.status}`);
    }
    if (response.status === 204) {
        return undefined;
    }
    return await response.json();
  };

  return {
    livez: () => requestJson("/livez"),
    readyz: () => requestJson("/readyz"),
    healthz: () => requestJson("/healthz"),
    request: requestJson,
  };
}