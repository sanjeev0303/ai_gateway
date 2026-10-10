import type { paths } from "@ai-governance/contracts";
import { ApiError, parseErrorEnvelope } from "./errors";

/** Any route literal declared in the OpenAPI contract. */
export type ContractPath = keyof paths & string;

export type HttpMethod = "GET" | "POST" | "PATCH" | "PUT" | "DELETE";

export type QueryValue =
  | string
  | number
  | boolean
  | Date
  | null
  | undefined
  | ReadonlyArray<string | number>;

export type Query = Record<string, QueryValue>;

export type FetchLike = (input: string, init: RequestInit) => Promise<Response>;

/** Safe methods are the only ones retried by default. */
const SAFE_METHODS: ReadonlySet<HttpMethod> = new Set<HttpMethod>(["GET"]);
const RETRYABLE_STATUS: ReadonlySet<number> = new Set([408, 429, 502, 503, 504]);
const DEFAULT_TIMEOUT_MS = 30_000;
const DEFAULT_RETRIES = 2;
const BASE_BACKOFF_MS = 200;
const MAX_BACKOFF_MS = 2_000;

export interface ApiClientOptions {
  /** Absolute origin, optionally with a path prefix (e.g. `https://host/api`). */
  baseUrl: string;
  /** Server-side session cookie to forward; browsers rely on the cookie jar. */
  cookie?: string | undefined;
  /** Double-submit CSRF token, echoed on unsafe methods. */
  csrfToken?: string | undefined;
  /** Defaults to `include` so browser calls send the session cookie. */
  credentials?: RequestCredentials | undefined;
  timeoutMs?: number | undefined;
  /** Retries for safe methods; unsafe methods need `retryable`. */
  retries?: number | undefined;
  fetch?: FetchLike | undefined;
  headers?: Record<string, string> | undefined;
  /** Generates the `X-Request-Id` sent on every request. */
  generateRequestId?: (() => string) | undefined;
  sleep?: ((ms: number) => Promise<void>) | undefined;
}

export interface RequestOptions {
  method?: HttpMethod | undefined;
  body?: unknown;
  query?: Query | undefined;
  headers?: Record<string, string> | undefined;
  signal?: AbortSignal | undefined;
  timeoutMs?: number | undefined;
  retries?: number | undefined;
  /** Opts an unsafe method into retrying; safe methods always may retry. */
  retryable?: boolean | undefined;
  /** Sends an `Authorization: Bearer` or `X-API-Key` header for this call. */
  apiKey?: string | undefined;
}

/** The slice of client behaviour that domain modules depend on. */
export interface Transport {
  /** Static routes are checked against the OpenAPI contract at compile time. */
  request<TResponse>(path: ContractPath, options?: RequestOptions): Promise<TResponse>;
  /** Interpolated routes (those carrying path params) escape static checking. */
  requestAt<TResponse>(path: string, options?: RequestOptions): Promise<TResponse>;
  /** Raw escape hatch for non-JSON responses such as the activity stream. */
  fetchRaw(path: string, options?: RequestOptions): Promise<Response>;
}

function isArrayValue(value: QueryValue): value is ReadonlyArray<string | number> {
  return Array.isArray(value);
}

function buildUrl(baseUrl: string, path: string, query?: Query | undefined): string {
  const base = baseUrl.replace(/\/+$/, "");
  const suffix = path.startsWith("/") ? path : `/${path}`;
  const url = new URL(`${base}${suffix}`);
  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value === undefined || value === null) continue;
      if (isArrayValue(value)) {
        for (const item of value) url.searchParams.append(key, String(item));
        continue;
      }
      url.searchParams.append(key, value instanceof Date ? value.toISOString() : String(value));
    }
  }
  return url.toString();
}

function parseRetryAfter(value: string | null): number | undefined {
  if (!value) return undefined;
  const seconds = Number(value);
  if (Number.isFinite(seconds) && seconds >= 0) return seconds * 1_000;
  const at = Date.parse(value);
  if (Number.isNaN(at)) return undefined;
  return Math.max(0, at - Date.now());
}

function backoffMs(attempt: number, retryAfterMs: number | undefined): number {
  if (retryAfterMs !== undefined) return Math.min(retryAfterMs, MAX_BACKOFF_MS);
  return Math.min(BASE_BACKOFF_MS * 2 ** attempt, MAX_BACKOFF_MS);
}

function resolveAttempts(method: HttpMethod, options: RequestOptions, defaults: ApiClientOptions): number {
  if (!SAFE_METHODS.has(method) && options.retryable !== true) return 0;
  return options.retries ?? defaults.retries ?? DEFAULT_RETRIES;
}

function defaultRequestId(): string {
  const cryptoRef: Crypto | undefined = globalThis.crypto;
  if (cryptoRef && typeof cryptoRef.randomUUID === "function") return cryptoRef.randomUUID();
  return Math.random().toString(36).slice(2);
}

function defaultSleep(ms: number): Promise<void> {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}

interface AttemptContext {
  url: string;
  method: HttpMethod;
  options: RequestOptions;
  settings: ApiClientOptions;
  requestId: string;
}

async function attemptOnce(context: AttemptContext): Promise<Response> {
  const { url, method, options, settings, requestId } = context;
  const headers = new Headers(settings.headers);
  for (const [key, value] of Object.entries(options.headers ?? {})) headers.set(key, value);
  headers.set("Accept", "application/json");
  headers.set("X-Request-Id", requestId);
  if (options.body !== undefined) headers.set("Content-Type", "application/json");
  if (settings.cookie) headers.set("Cookie", settings.cookie);
  if (!SAFE_METHODS.has(method) && settings.csrfToken) headers.set("X-CSRF-Token", settings.csrfToken);
  if (options.apiKey) headers.set("X-API-Key", options.apiKey);

  const timeoutMs = options.timeoutMs ?? settings.timeoutMs ?? DEFAULT_TIMEOUT_MS;
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort("timeout"), timeoutMs);
  const external = options.signal;
  const forwardAbort = (): void => controller.abort(external?.reason);
  if (external) {
    if (external.aborted) forwardAbort();
    else external.addEventListener("abort", forwardAbort, { once: true });
  }

  try {
    const fetchImpl = settings.fetch ?? globalThis.fetch;
    return await fetchImpl(url, {
      method,
      headers,
      ...(options.body === undefined ? {} : { body: JSON.stringify(options.body) }),
      credentials: settings.credentials ?? "include",
      signal: controller.signal,
    });
  } catch (cause) {
    if (external?.aborted) {
      throw new ApiError("Request aborted by caller", { status: 0, method, path: url, cause }, "aborted");
    }
    if (controller.signal.aborted) {
      throw new ApiError(`Request timed out after ${timeoutMs}ms`, { status: 0, method, path: url, cause }, "timeout");
    }
    throw new ApiError(
      cause instanceof Error ? cause.message : "Network request failed",
      { status: 0, method, path: url, cause },
      "network_error",
    );
  } finally {
    clearTimeout(timer);
    external?.removeEventListener("abort", forwardAbort);
  }
}

async function readBody(response: Response): Promise<unknown> {
  const text = await response.text();
  if (!text) return undefined;
  try {
    return JSON.parse(text) as unknown;
  } catch {
    return text;
  }
}

async function send<TResponse>(context: AttemptContext): Promise<TResponse> {
  const response = await attemptOnce(context);
  const headerRequestId = response.headers.get("X-Request-Id") ?? undefined;

  if (response.status === 204) return undefined as TResponse;

  const body = await readBody(response);
  if (response.ok) return normaliseListItems(body) as TResponse;

  const detail = parseErrorEnvelope(body);
  const retryAfterMs = parseRetryAfter(response.headers.get("Retry-After"));
  throw new ApiError(
    detail?.message ?? `Request failed with status ${response.status}`,
    {
      status: response.status,
      method: context.method,
      path: context.url,
      ...(body === undefined ? {} : { body }),
      requestId: headerRequestId,
      ...(retryAfterMs === undefined ? {} : { retryAfterMs }),
    },
  );
}

/** Go marshals a nil slice as `null`, so empty aggregation lists arrive as `{ items: null }`. */
function normaliseListItems(body: unknown): unknown {
  if (typeof body !== "object" || body === null || Array.isArray(body)) return body;
  const record = body as Record<string, unknown>;
  if (record.items !== null) return body;
  record.items = [];
  return body;
}

export function createTransport(settings: ApiClientOptions): Transport {
  const sleep = settings.sleep ?? defaultSleep;
  const generateRequestId = settings.generateRequestId ?? defaultRequestId;

  async function run<TResponse>(path: string, options: RequestOptions): Promise<TResponse> {
    const method = options.method ?? "GET";
    const url = buildUrl(settings.baseUrl, path, options.query);
    const attempts = resolveAttempts(method, options, settings);

    let lastError: unknown;
    for (let attempt = 0; attempt <= attempts; attempt += 1) {
      try {
        return await send<TResponse>({
          url,
          method,
          options,
          settings,
          requestId: generateRequestId(),
        });
      } catch (error) {
        lastError = error;
        const retryable =
          error instanceof ApiError &&
          (error.code === "network_error" || RETRYABLE_STATUS.has(error.status));
        if (!retryable || attempt === attempts) throw error;
        await sleep(backoffMs(attempt, error instanceof ApiError ? error.retryAfterMs : undefined));
      }
    }
    throw lastError;
  }

  return {
    request: <TResponse>(path: ContractPath, options?: RequestOptions): Promise<TResponse> =>
      run<TResponse>(path, options ?? {}),
    requestAt: <TResponse>(path: string, options?: RequestOptions): Promise<TResponse> =>
      run<TResponse>(path, options ?? {}),
    fetchRaw: (path: string, options?: RequestOptions): Promise<Response> =>
      attemptOnce({
        url: buildUrl(settings.baseUrl, path, options?.query),
        method: options?.method ?? "GET",
        options: options ?? {},
        settings,
        requestId: generateRequestId(),
      }),
  };
}

/** Envelope shared by every list endpoint. `items` is always an array, never `null`. */
export interface ListResponse<TItem> {
  items: TItem[];
  has_more?: boolean;
  next_cursor?: string;
}

/** One page of client-side pagination. */
export interface Page<TItem> {
  items: TItem[];
  page: number;
  pageSize: number;
  total: number;
  pageCount: number;
}

/**
 * Slices an already-fetched list. The control API exposes no server-side
 * pagination (no limit/offset/cursor parameters), so tables that paginate do
 * it here rather than round-tripping the whole collection per page.
 */
export function paginate<TItem>(
  items: readonly TItem[],
  options: { page?: number; pageSize?: number } = {},
): Page<TItem> {
  const pageSize = Math.max(1, Math.trunc(options.pageSize ?? 25));
  const total = items.length;
  const pageCount = Math.max(1, Math.ceil(total / pageSize));
  const page = Math.min(Math.max(1, Math.trunc(options.page ?? 1)), pageCount);
  const start = (page - 1) * pageSize;
  return { items: items.slice(start, start + pageSize), page, pageSize, total, pageCount };
}

