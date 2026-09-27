const CONTROL_API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

export interface SessionClaims {
  readonly subject: string;
  readonly email: string;
  readonly display_name: string;
  readonly organization_id: string;
  readonly role: string;
  readonly kind: string;
  readonly issued_at: string;
  readonly expires_at: string;
}

export interface Session {
  readonly claims: SessionClaims;
  readonly csrf_token?: string;
}

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${CONTROL_API}${path}`, {
    ...init,
    credentials: "include",
    headers: {
      ...(init.body === undefined ? {} : { "Content-Type": "application/json" }),
      ...init.headers,
    },
  });
  if (!response.ok) {
    throw await toApiError(response);
  }
  // Logout answers 204 with an empty body, so there is nothing to decode.
  if (response.status === 204) {
    return undefined as T;
  }
  return (await response.json()) as T;
}

async function toApiError(response: Response): Promise<ApiError> {
  let code = "request_failed";
  let message = `Request failed with status ${response.status}`;
  try {
    const body = (await response.json()) as { error?: { code?: string; message?: string } };
    if (body.error) {
      code = body.error.code ?? code;
      message = body.error.message ?? message;
    }
  } catch {
    // A proxy or crash can return a non-JSON body; the status is all we have.
  }
  return new ApiError(response.status, code, message);
}

function post<T>(path: string, body: unknown, csrfToken?: string): Promise<T> {
  return request<T>(path, {
    method: "POST",
    body: JSON.stringify(body),
    ...(csrfToken === undefined ? {} : { headers: { "X-CSRF-Token": csrfToken } }),
  });
}

export const signIn = (email: string, password: string): Promise<Session> =>
  post<Session>("/v1/auth/signin", { email, password });

export const signUp = (input: {
  readonly email: string;
  readonly password: string;
  readonly organization_name: string;
  readonly display_name?: string;
}): Promise<Session> => post<Session>("/v1/auth/signup", input);

export const getSession = (): Promise<Session> => request<Session>("/v1/auth/session");

export const refreshSession = (csrfToken: string): Promise<Session> =>
  post<Session>("/v1/auth/refresh", {}, csrfToken);

export const signOut = (csrfToken: string): Promise<void> =>
  request<void>("/v1/auth/logout", {
    method: "POST",
    headers: { "X-CSRF-Token": csrfToken },
  });
