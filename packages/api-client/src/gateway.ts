import { ApiError } from "./errors";
import type { FetchLike } from "./client";

export interface GatewayClientOptions {
  gatewayUrl: string;
  apiKey: string;
  projectId?: string | undefined;
  fetch?: FetchLike | undefined;
  timeoutMs?: number | undefined;
  maxRetries?: number | undefined;
  retryDelayMs?: number | undefined;
}

export interface GatewayModel {
  id: string;
  object: string;
  provider: string;
  display_name?: string | undefined;
}

export interface ChatMessage {
  role: "system" | "user" | "assistant";
  content: string;
}

export interface ChatCompletionRequest {
  model: string;
  messages: ChatMessage[];
  stream?: boolean | undefined;
  max_tokens?: number | undefined;
  temperature?: number | undefined;
  project_id?: string | undefined;
  idempotencyKey?: string | undefined;
  signal?: AbortSignal | undefined;
}

export interface AnthropicMessage {
  role: "user" | "assistant";
  content: string;
}

export interface AnthropicMessageRequest {
  model: string;
  messages: AnthropicMessage[];
  max_tokens?: number | undefined;
  stream?: boolean | undefined;
  system?: string | undefined;
  project_id?: string | undefined;
  idempotencyKey?: string | undefined;
  signal?: AbortSignal | undefined;
}

export interface GatewayExecutionMetadata {
  requestId?: string | undefined;
  statusCode: number;
  model: string;
  inputTokens?: number | undefined;
  outputTokens?: number | undefined;
  totalTokens?: number | undefined;
  durationMs: number;
  isEstimated?: boolean | undefined;
  usageSource?: "reported" | "estimated" | "none" | undefined;
}

export interface GatewayClient {
  listModels(options?: { signal?: AbortSignal }): Promise<GatewayModel[]>;
  chatCompletion(req: ChatCompletionRequest): Promise<{ text: string; raw: any; meta: GatewayExecutionMetadata }>;
  streamChatCompletion(
    req: ChatCompletionRequest,
    onDelta: (chunk: string) => void
  ): Promise<GatewayExecutionMetadata>;
  sendMessage(req: AnthropicMessageRequest): Promise<{ text: string; raw: any; meta: GatewayExecutionMetadata }>;
  streamMessage(
    req: AnthropicMessageRequest,
    onDelta: (chunk: string) => void
  ): Promise<GatewayExecutionMetadata>;
}

export function createGatewayClient(options: GatewayClientOptions): GatewayClient {
  const baseUrl = options.gatewayUrl.replace(/\/+$/, "");
  const fetchImpl = options.fetch ?? (globalThis.fetch as FetchLike);
  const maxRetries = options.maxRetries ?? 0;
  const retryDelayMs = options.retryDelayMs ?? 200;

  function getHeaders(customHeaders?: Record<string, string>): Record<string, string> {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      "x-api-key": options.apiKey,
      Authorization: `Bearer ${options.apiKey}`,
      ...customHeaders,
    };
    if (options.projectId && !headers["x-project-id"]) {
      headers["x-project-id"] = options.projectId;
    }
    return headers;
  }

  function combineSignal(userSignal?: AbortSignal): AbortSignal | undefined {
    if (!options.timeoutMs && !userSignal) {
      return undefined;
    }
    const controller = new AbortController();
    let timeoutId: any = undefined;

    if (options.timeoutMs && options.timeoutMs > 0) {
      timeoutId = setTimeout(() => {
        controller.abort(new Error(`Gateway request timed out after ${options.timeoutMs}ms`));
      }, options.timeoutMs);
    }

    if (userSignal) {
      if (userSignal.aborted) {
        if (timeoutId) clearTimeout(timeoutId);
        controller.abort(userSignal.reason);
        return controller.signal;
      }
      userSignal.addEventListener(
        "abort",
        () => {
          if (timeoutId) clearTimeout(timeoutId);
          controller.abort(userSignal.reason);
        },
        { once: true }
      );
    }

    return controller.signal;
  }

  async function parseError(res: Response, path: string, method: string): Promise<ApiError> {
    const text = await res.text();
    let body: unknown = undefined;
    let message = text || res.statusText;
    let code = `http_${res.status}`;
    try {
      const json = JSON.parse(text);
      body = json;
      if (json.error?.message) {
        message = json.error.message;
      }
      if (json.error?.type || json.error?.code || json.error?.status) {
        code = json.error.type || json.error.code || json.error.status;
      }
    } catch {
      // not JSON
    }
    return new ApiError(
      message,
      {
        status: res.status,
        method,
        path,
        body,
        requestId: res.headers.get("x-request-id") ?? undefined,
      },
      code
    );
  }

  async function sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  return {
    async listModels(listOpts?: { signal?: AbortSignal }): Promise<GatewayModel[]> {
      const path = "/v1/models";
      let attempt = 0;
      while (true) {
        const signal = combineSignal(listOpts?.signal);
        try {
          const res = await fetchImpl(`${baseUrl}${path}`, {
            method: "GET",
            headers: getHeaders(),
            ...(signal ? { signal } : {}),
          });
          if (!res.ok) {
            // Idempotent GET request: retry on rate limit (429) or temporary server error (503)
            if (attempt < maxRetries && (res.status === 429 || res.status === 503)) {
              attempt++;
              await sleep(retryDelayMs * Math.pow(2, attempt - 1));
              continue;
            }
            throw await parseError(res, path, "GET");
          }
          const data = await res.json();
          if (Array.isArray(data.data)) {
            return data.data;
          }
          if (Array.isArray(data.items)) {
            return data.items;
          }
          return [];
        } catch (err: any) {
          if (err instanceof ApiError) {
            throw err;
          }
          if (attempt < maxRetries && !listOpts?.signal?.aborted) {
            attempt++;
            await sleep(retryDelayMs * Math.pow(2, attempt - 1));
            continue;
          }
          throw err;
        }
      }
    },

    async chatCompletion(req: ChatCompletionRequest): Promise<{ text: string; raw: any; meta: GatewayExecutionMetadata }> {
      const start = Date.now();
      const path = "/v1/chat/completions";
      const body = {
        ...req,
        stream: false,
        project_id: req.project_id ?? options.projectId,
      };

      const extraHeaders: Record<string, string> = {};
      if (req.idempotencyKey) {
        extraHeaders["Idempotency-Key"] = req.idempotencyKey;
      }

      let attempt = 0;
      while (true) {
        const signal = combineSignal(req.signal);
        try {
          const res = await fetchImpl(`${baseUrl}${path}`, {
            method: "POST",
            headers: getHeaders(extraHeaders),
            body: JSON.stringify(body),
            ...(signal ? { signal } : {}),
          });

          if (!res.ok) {
            // Non-idempotent operations: ONLY retry if idempotency key is present or on safe 429/503
            const canRetry = Boolean(req.idempotencyKey) || res.status === 429 || res.status === 503;
            if (attempt < maxRetries && canRetry) {
              attempt++;
              await sleep(retryDelayMs * Math.pow(2, attempt - 1));
              continue;
            }
            throw await parseError(res, path, "POST");
          }

          const json = await res.json();
          let text = json.choices?.[0]?.message?.content ?? "";
          if (!text && Array.isArray(json.candidates)) {
            for (const candidate of json.candidates) {
              if (Array.isArray(candidate.content?.parts)) {
                for (const part of candidate.content.parts) {
                  if (part.text) {
                    text += part.text;
                  }
                }
              }
            }
          }
          if (!text && Array.isArray(json.content)) {
            for (const item of json.content) {
              if (item.type === "text" && item.text) {
                text += item.text;
              }
            }
          }

          const hasOpenAIUsage = json.usage?.prompt_tokens !== undefined;
          const hasGeminiUsage = json.usageMetadata?.promptTokenCount !== undefined;
          const hasAnthropicUsage = json.usage?.input_tokens !== undefined;
          const tokensReported = hasOpenAIUsage || hasGeminiUsage || hasAnthropicUsage;

          const inputTokens = json.usage?.prompt_tokens ?? json.usageMetadata?.promptTokenCount ?? json.usage?.input_tokens;
          const outputTokens = json.usage?.completion_tokens ?? json.usageMetadata?.candidatesTokenCount ?? json.usage?.output_tokens;
          const totalTokens = json.usage?.total_tokens ?? json.usageMetadata?.totalTokenCount ?? (
            (inputTokens !== undefined || outputTokens !== undefined) ? ((inputTokens ?? 0) + (outputTokens ?? 0)) : undefined
          );

          const meta: GatewayExecutionMetadata = {
            requestId: res.headers.get("x-request-id") ?? undefined,
            statusCode: res.status,
            model: req.model,
            inputTokens,
            outputTokens,
            totalTokens,
            durationMs: Date.now() - start,
            isEstimated: !tokensReported,
            usageSource: tokensReported ? "reported" : (inputTokens !== undefined ? "estimated" : "none"),
          };

          return { text, raw: json, meta };
        } catch (err: any) {
          if (err instanceof ApiError) {
            throw err;
          }
          if (attempt < maxRetries && Boolean(req.idempotencyKey) && !req.signal?.aborted) {
            attempt++;
            await sleep(retryDelayMs * Math.pow(2, attempt - 1));
            continue;
          }
          throw err;
        }
      }
    },

    async streamChatCompletion(
      req: ChatCompletionRequest,
      onDelta: (chunk: string) => void
    ): Promise<GatewayExecutionMetadata> {
      const start = Date.now();
      const path = "/v1/chat/completions";
      const body = {
        ...req,
        stream: true,
        project_id: req.project_id ?? options.projectId,
      };

      const extraHeaders: Record<string, string> = {
        Accept: "text/event-stream",
      };
      if (req.idempotencyKey) {
        extraHeaders["Idempotency-Key"] = req.idempotencyKey;
      }

      const signal = combineSignal(req.signal);
      const res = await fetchImpl(`${baseUrl}${path}`, {
        method: "POST",
        headers: getHeaders(extraHeaders),
        body: JSON.stringify(body),
        ...(signal ? { signal } : {}),
      });

      if (!res.ok) {
        throw await parseError(res, path, "POST");
      }

      const meta: GatewayExecutionMetadata = {
        requestId: res.headers.get("x-request-id") ?? undefined,
        statusCode: res.status,
        model: req.model,
        durationMs: 0,
        isEstimated: true,
        usageSource: "none",
      };

      if (!res.body) {
        meta.durationMs = Date.now() - start;
        return meta;
      }

      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";
      let tokensReported = false;

      try {
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });
          const lines = buffer.split("\n");
          buffer = lines.pop() ?? "";

          for (const line of lines) {
            const trimmed = line.trim();
            if (!trimmed.startsWith("data:")) continue;
            const payload = trimmed.slice(5).trim();
            if (payload === "" || payload === "[DONE]") continue;

            try {
              const parsed = JSON.parse(payload);
              const content = parsed.choices?.[0]?.delta?.content;
              if (content) {
                onDelta(content);
              } else if (Array.isArray(parsed.candidates)) {
                for (const candidate of parsed.candidates) {
                  if (Array.isArray(candidate.content?.parts)) {
                    for (const part of candidate.content.parts) {
                      if (part.text) {
                        onDelta(part.text);
                      }
                    }
                  }
                }
              }

              if (parsed.usage) {
                tokensReported = true;
                meta.inputTokens = parsed.usage.prompt_tokens ?? parsed.usage.input_tokens;
                meta.outputTokens = parsed.usage.completion_tokens ?? parsed.usage.output_tokens;
                meta.totalTokens = parsed.usage.total_tokens ?? ((meta.inputTokens ?? 0) + (meta.outputTokens ?? 0));
              } else if (parsed.usageMetadata) {
                tokensReported = true;
                if (parsed.usageMetadata.promptTokenCount !== undefined) meta.inputTokens = parsed.usageMetadata.promptTokenCount;
                if (parsed.usageMetadata.candidatesTokenCount !== undefined) meta.outputTokens = parsed.usageMetadata.candidatesTokenCount;
                if (parsed.usageMetadata.totalTokenCount !== undefined) meta.totalTokens = parsed.usageMetadata.totalTokenCount;
              }
            } catch {
              // ignore unparseable SSE frame
            }
          }
        }
      } catch (err: any) {
        if (req.signal?.aborted) {
          throw err;
        }
        throw new ApiError(
          `Stream interrupted: ${err?.message || "connection closed prematurely"}`,
          {
            status: 502,
            method: "POST",
            path,
            requestId: meta.requestId,
          },
          "stream_interrupted"
        );
      }

      meta.durationMs = Date.now() - start;
      if (meta.totalTokens === undefined && meta.inputTokens !== undefined && meta.outputTokens !== undefined) {
        meta.totalTokens = meta.inputTokens + meta.outputTokens;
      }
      meta.isEstimated = !tokensReported;
      meta.usageSource = tokensReported ? "reported" : (meta.inputTokens !== undefined ? "estimated" : "none");
      return meta;
    },

    async sendMessage(req: AnthropicMessageRequest): Promise<{ text: string; raw: any; meta: GatewayExecutionMetadata }> {
      const start = Date.now();
      const path = "/v1/messages";
      const body = {
        ...req,
        stream: false,
        max_tokens: req.max_tokens ?? 1024,
        project_id: req.project_id ?? options.projectId,
      };

      const extraHeaders: Record<string, string> = {};
      if (req.idempotencyKey) {
        extraHeaders["Idempotency-Key"] = req.idempotencyKey;
      }

      let attempt = 0;
      while (true) {
        const signal = combineSignal(req.signal);
        try {
          const res = await fetchImpl(`${baseUrl}${path}`, {
            method: "POST",
            headers: getHeaders(extraHeaders),
            body: JSON.stringify(body),
            ...(signal ? { signal } : {}),
          });

          if (!res.ok) {
            const canRetry = Boolean(req.idempotencyKey) || res.status === 429 || res.status === 503;
            if (attempt < maxRetries && canRetry) {
              attempt++;
              await sleep(retryDelayMs * Math.pow(2, attempt - 1));
              continue;
            }
            throw await parseError(res, path, "POST");
          }

          const json = await res.json();
          let text = "";
          if (Array.isArray(json.content)) {
            for (const item of json.content) {
              if (item.type === "text" && item.text) {
                text += item.text;
              }
            }
          } else if (Array.isArray(json.candidates)) {
            for (const candidate of json.candidates) {
              if (Array.isArray(candidate.content?.parts)) {
                for (const part of candidate.content.parts) {
                  if (part.text) {
                    text += part.text;
                  }
                }
              }
            }
          } else if (json.choices?.[0]?.message?.content) {
            text = json.choices[0].message.content;
          }

          const hasAnthropicUsage = json.usage?.input_tokens !== undefined;
          const hasGeminiUsage = json.usageMetadata?.promptTokenCount !== undefined;
          const hasOpenAIUsage = json.usage?.prompt_tokens !== undefined;
          const tokensReported = hasAnthropicUsage || hasGeminiUsage || hasOpenAIUsage;

          const inputTokens = json.usage?.input_tokens ?? json.usageMetadata?.promptTokenCount ?? json.usage?.prompt_tokens;
          const outputTokens = json.usage?.output_tokens ?? json.usageMetadata?.candidatesTokenCount ?? json.usage?.completion_tokens;
          const totalTokens = json.usage?.total_tokens ?? json.usageMetadata?.totalTokenCount ?? (
            (inputTokens !== undefined || outputTokens !== undefined) ? ((inputTokens ?? 0) + (outputTokens ?? 0)) : undefined
          );

          const meta: GatewayExecutionMetadata = {
            requestId: res.headers.get("x-request-id") ?? undefined,
            statusCode: res.status,
            model: req.model,
            inputTokens,
            outputTokens,
            totalTokens,
            durationMs: Date.now() - start,
            isEstimated: !tokensReported,
            usageSource: tokensReported ? "reported" : (inputTokens !== undefined ? "estimated" : "none"),
          };

          return { text, raw: json, meta };
        } catch (err: any) {
          if (err instanceof ApiError) {
            throw err;
          }
          if (attempt < maxRetries && Boolean(req.idempotencyKey) && !req.signal?.aborted) {
            attempt++;
            await sleep(retryDelayMs * Math.pow(2, attempt - 1));
            continue;
          }
          throw err;
        }
      }
    },

    async streamMessage(
      req: AnthropicMessageRequest,
      onDelta: (chunk: string) => void
    ): Promise<GatewayExecutionMetadata> {
      const start = Date.now();
      const path = "/v1/messages";
      const body = {
        ...req,
        stream: true,
        max_tokens: req.max_tokens ?? 1024,
        project_id: req.project_id ?? options.projectId,
      };

      const extraHeaders: Record<string, string> = {
        Accept: "text/event-stream",
      };
      if (req.idempotencyKey) {
        extraHeaders["Idempotency-Key"] = req.idempotencyKey;
      }

      const signal = combineSignal(req.signal);
      const res = await fetchImpl(`${baseUrl}${path}`, {
        method: "POST",
        headers: getHeaders(extraHeaders),
        body: JSON.stringify(body),
        ...(signal ? { signal } : {}),
      });

      if (!res.ok) {
        throw await parseError(res, path, "POST");
      }

      const meta: GatewayExecutionMetadata = {
        requestId: res.headers.get("x-request-id") ?? undefined,
        statusCode: res.status,
        model: req.model,
        durationMs: 0,
        isEstimated: true,
        usageSource: "none",
      };

      if (!res.body) {
        meta.durationMs = Date.now() - start;
        return meta;
      }

      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";
      let tokensReported = false;

      try {
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });
          const lines = buffer.split("\n");
          buffer = lines.pop() ?? "";

          for (const line of lines) {
            const trimmed = line.trim();
            if (!trimmed.startsWith("data:")) continue;
            const payload = trimmed.slice(5).trim();
            if (payload === "" || payload === "[DONE]") continue;

            try {
              const parsed = JSON.parse(payload);
              if (parsed.type === "content_block_delta" && parsed.delta?.text) {
                onDelta(parsed.delta.text);
              } else if (Array.isArray(parsed.candidates)) {
                for (const candidate of parsed.candidates) {
                  if (Array.isArray(candidate.content?.parts)) {
                    for (const part of candidate.content.parts) {
                      if (part.text) {
                        onDelta(part.text);
                      }
                    }
                  }
                }
              }

              if (parsed.type === "message_start" && parsed.message?.usage?.input_tokens !== undefined) {
                tokensReported = true;
                meta.inputTokens = parsed.message.usage.input_tokens;
              }
              if (parsed.type === "message_delta" && parsed.usage?.output_tokens !== undefined) {
                tokensReported = true;
                meta.outputTokens = parsed.usage.output_tokens;
              }
              if (parsed.usageMetadata) {
                tokensReported = true;
                if (parsed.usageMetadata.promptTokenCount !== undefined) meta.inputTokens = parsed.usageMetadata.promptTokenCount;
                if (parsed.usageMetadata.candidatesTokenCount !== undefined) meta.outputTokens = parsed.usageMetadata.candidatesTokenCount;
                if (parsed.usageMetadata.totalTokenCount !== undefined) meta.totalTokens = parsed.usageMetadata.totalTokenCount;
              }
            } catch {
              // ignore non-json SSE lines
            }
          }
        }
      } catch (err: any) {
        if (req.signal?.aborted) {
          throw err;
        }
        throw new ApiError(
          `Stream interrupted: ${err?.message || "connection closed prematurely"}`,
          {
            status: 502,
            method: "POST",
            path,
            requestId: meta.requestId,
          },
          "stream_interrupted"
        );
      }

      meta.durationMs = Date.now() - start;
      if (meta.totalTokens === undefined && meta.inputTokens !== undefined && meta.outputTokens !== undefined) {
        meta.totalTokens = meta.inputTokens + meta.outputTokens;
      }
      meta.isEstimated = !tokensReported;
      meta.usageSource = tokensReported ? "reported" : (meta.inputTokens !== undefined ? "estimated" : "none");
      return meta;
    },
  };
}
