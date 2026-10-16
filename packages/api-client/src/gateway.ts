import { ApiError } from "./errors";
import type { FetchLike } from "./client";

export interface GatewayClientOptions {
  gatewayUrl: string;
  apiKey: string;
  projectId?: string | undefined;
  fetch?: FetchLike | undefined;
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
}

export interface GatewayExecutionMetadata {
  requestId?: string | undefined;
  statusCode: number;
  model: string;
  inputTokens?: number | undefined;
  outputTokens?: number | undefined;
  totalTokens?: number | undefined;
  durationMs: number;
}

export interface GatewayClient {
  listModels(): Promise<GatewayModel[]>;
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

  function getHeaders(): Record<string, string> {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      "x-api-key": options.apiKey,
      Authorization: `Bearer ${options.apiKey}`,
    };
    if (options.projectId) {
      headers["x-project-id"] = options.projectId;
    }
    return headers;
  }

  async function parseError(res: Response, path: string): Promise<ApiError> {
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
      if (json.error?.type || json.error?.code) {
        code = json.error.type || json.error.code;
      }
    } catch {
      // not JSON
    }
    return new ApiError(
      message,
      {
        status: res.status,
        method: "POST",
        path,
        body,
        requestId: res.headers.get("x-request-id") ?? undefined,
      },
      code
    );
  }

  return {
    async listModels(): Promise<GatewayModel[]> {
      const path = "/v1/models";
      const res = await fetchImpl(`${baseUrl}${path}`, {
        method: "GET",
        headers: getHeaders(),
      });
      if (!res.ok) {
        throw await parseError(res, path);
      }
      const data = await res.json();
      if (Array.isArray(data.data)) {
        return data.data;
      }
      if (Array.isArray(data.items)) {
        return data.items;
      }
      return [];
    },

    async chatCompletion(req: ChatCompletionRequest): Promise<{ text: string; raw: any; meta: GatewayExecutionMetadata }> {
      const start = Date.now();
      const path = "/v1/chat/completions";
      const body = {
        ...req,
        stream: false,
        project_id: req.project_id ?? options.projectId,
      };

      const res = await fetchImpl(`${baseUrl}${path}`, {
        method: "POST",
        headers: getHeaders(),
        body: JSON.stringify(body),
      });

      if (!res.ok) {
        throw await parseError(res, path);
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
      const inputTokens = json.usage?.prompt_tokens ?? json.usageMetadata?.promptTokenCount;
      const outputTokens = json.usage?.completion_tokens ?? json.usageMetadata?.candidatesTokenCount;
      const totalTokens = json.usage?.total_tokens ?? json.usageMetadata?.totalTokenCount;
      const meta: GatewayExecutionMetadata = {
        requestId: res.headers.get("x-request-id") ?? undefined,
        statusCode: res.status,
        model: req.model,
        inputTokens,
        outputTokens,
        totalTokens,
        durationMs: Date.now() - start,
      };

      return { text, raw: json, meta };
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

      const res = await fetchImpl(`${baseUrl}${path}`, {
        method: "POST",
        headers: {
          ...getHeaders(),
          Accept: "text/event-stream",
        },
        body: JSON.stringify(body),
      });

      if (!res.ok) {
        throw await parseError(res, path);
      }

      const meta: GatewayExecutionMetadata = {
        requestId: res.headers.get("x-request-id") ?? undefined,
        statusCode: res.status,
        model: req.model,
        durationMs: 0,
      };

      if (!res.body) {
        meta.durationMs = Date.now() - start;
        return meta;
      }

      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";

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
              meta.inputTokens = parsed.usage.prompt_tokens;
              meta.outputTokens = parsed.usage.completion_tokens;
              meta.totalTokens = parsed.usage.total_tokens;
            } else if (parsed.usageMetadata) {
              if (parsed.usageMetadata.promptTokenCount) meta.inputTokens = parsed.usageMetadata.promptTokenCount;
              if (parsed.usageMetadata.candidatesTokenCount) meta.outputTokens = parsed.usageMetadata.candidatesTokenCount;
              if (parsed.usageMetadata.totalTokenCount) meta.totalTokens = parsed.usageMetadata.totalTokenCount;
            }
          } catch {
            // ignore non-json SSE lines
          }
        }
      }

      meta.durationMs = Date.now() - start;
      if (meta.totalTokens === undefined && meta.inputTokens !== undefined && meta.outputTokens !== undefined) {
        meta.totalTokens = meta.inputTokens + meta.outputTokens;
      }
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

      const res = await fetchImpl(`${baseUrl}${path}`, {
        method: "POST",
        headers: getHeaders(),
        body: JSON.stringify(body),
      });

      if (!res.ok) {
        throw await parseError(res, path);
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
      }

      const inputTokens = json.usage?.input_tokens ?? json.usageMetadata?.promptTokenCount;
      const outputTokens = json.usage?.output_tokens ?? json.usageMetadata?.candidatesTokenCount;
      const totalTokens = json.usage?.total_tokens ?? json.usageMetadata?.totalTokenCount ?? ((inputTokens ?? 0) + (outputTokens ?? 0));

      const meta: GatewayExecutionMetadata = {
        requestId: res.headers.get("x-request-id") ?? undefined,
        statusCode: res.status,
        model: req.model,
        inputTokens,
        outputTokens,
        totalTokens,
        durationMs: Date.now() - start,
      };

      return { text, raw: json, meta };
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

      const res = await fetchImpl(`${baseUrl}${path}`, {
        method: "POST",
        headers: {
          ...getHeaders(),
          Accept: "text/event-stream",
        },
        body: JSON.stringify(body),
      });

      if (!res.ok) {
        throw await parseError(res, path);
      }

      const meta: GatewayExecutionMetadata = {
        requestId: res.headers.get("x-request-id") ?? undefined,
        statusCode: res.status,
        model: req.model,
        durationMs: 0,
      };

      if (!res.body) {
        meta.durationMs = Date.now() - start;
        return meta;
      }

      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";

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
            if (parsed.type === "message_start" && parsed.message?.usage?.input_tokens) {
              meta.inputTokens = parsed.message.usage.input_tokens;
            }
            if (parsed.type === "message_delta" && parsed.usage?.output_tokens) {
              meta.outputTokens = parsed.usage.output_tokens;
            }
            if (parsed.usageMetadata) {
              if (parsed.usageMetadata.promptTokenCount) meta.inputTokens = parsed.usageMetadata.promptTokenCount;
              if (parsed.usageMetadata.candidatesTokenCount) meta.outputTokens = parsed.usageMetadata.candidatesTokenCount;
              if (parsed.usageMetadata.totalTokenCount) meta.totalTokens = parsed.usageMetadata.totalTokenCount;
            }
          } catch {
            // ignore non-json SSE lines
          }
        }
      }

      meta.durationMs = Date.now() - start;
      if (meta.totalTokens === undefined && meta.inputTokens !== undefined && meta.outputTokens !== undefined) {
        meta.totalTokens = meta.inputTokens + meta.outputTokens;
      }
      return meta;
    },
  };
}
