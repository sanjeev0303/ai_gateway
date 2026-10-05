/**
 * Structured errors for the control API.
 *
 * Every control-api failure is reported as
 * `{ "error": { "code", "message", "request_id"?, "details"? } }`
 * (see backend/internal/platform/httpx/respond.go). `ApiError` keeps that
 * envelope intact so callers can branch on `code` and surface `requestId` in
 * bug reports instead of pattern-matching on message text.
 */

/** The `error` object of the API's structured error envelope. */
export interface ApiErrorDetail {
  code: string;
  message: string;
  request_id?: string | undefined;
  details?: unknown;
}

/** Pulls the error detail out of a decoded body, or null when it is not one. */
export function parseErrorEnvelope(body: unknown): ApiErrorDetail | null {
  if (typeof body !== "object" || body === null) return null;
  const error = (body as { error?: unknown }).error;
  if (typeof error !== "object" || error === null) return null;
  const detail = error as Partial<ApiErrorDetail>;
  if (typeof detail.code !== "string" || typeof detail.message !== "string") return null;
  return {
    code: detail.code,
    message: detail.message,
    ...(typeof detail.request_id === "string" ? { request_id: detail.request_id } : {}),
    ...(detail.details !== undefined ? { details: detail.details } : {}),
  };
}

/** Coarse code for failures that never reached an error envelope. */
function fallbackCode(status: number): string {
  if (status === 0) return "network_error";
  return `http_${status}`;
}

export interface ApiErrorInit {
  /** HTTP status, or 0 when the request never produced a response. */
  status: number;
  method: string;
  path: string;
  /** Decoded response body, used to recover the structured envelope. */
  body?: unknown;
  /** Overrides the envelope's request_id, e.g. from the `X-Request-Id` header. */
  requestId?: string | undefined;
  /** Parsed `Retry-After`, in milliseconds. */
  retryAfterMs?: number | undefined;
  cause?: unknown;
}

export class ApiError extends Error {
  override readonly name = "ApiError";
  readonly status: number;
  readonly code: string;
  readonly method: string;
  readonly path: string;
  readonly requestId?: string | undefined;
  readonly details?: unknown;
  readonly body?: unknown;
  readonly retryAfterMs?: number | undefined;

  constructor(message: string, init: ApiErrorInit, code?: string) {
    super(message, { cause: init.cause });
    const detail = parseErrorEnvelope(init.body);
    this.status = init.status;
    this.method = init.method;
    this.path = init.path;
    this.code = code ?? detail?.code ?? fallbackCode(init.status);
    this.requestId = init.requestId ?? detail?.request_id;
    this.details = detail?.details;
    this.body = init.body;
    this.retryAfterMs = init.retryAfterMs;
  }

  /** True for 5xx responses, which are the ones worth retrying. */
  get isServerError(): boolean {
    return this.status >= 500;
  }

  /** True when the caller needs to sign in again. */
  get isUnauthenticated(): boolean {
    return this.status === 401;
  }
}

export function isApiError(value: unknown): value is ApiError {
  return value instanceof ApiError;
}