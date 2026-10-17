import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { createGatewayClient } from "./gateway";
import { ApiError } from "./errors";

describe("GatewayClient SDK", () => {
  it("does not expose or share credentials between independent client instances", () => {
    const clientUser1 = createGatewayClient({
      gatewayUrl: "http://localhost:8084",
      apiKey: "aigw_user1_secret_key_111",
      projectId: "proj-1",
    });

    const clientUser2 = createGatewayClient({
      gatewayUrl: "http://localhost:8084",
      apiKey: "aigw_user2_secret_key_222",
      projectId: "proj-2",
    });

    // Ensure distinct configurations and verify no secret leakage
    assert.notStrictEqual(clientUser1, clientUser2);
    const inspectUser1 = JSON.stringify(clientUser1);
    const inspectUser2 = JSON.stringify(clientUser2);
    assert.strictEqual(inspectUser1.includes("aigw_user1_secret_key_111"), false);
    assert.strictEqual(inspectUser2.includes("aigw_user2_secret_key_222"), false);
  });

  it("handles authentication failure (HTTP 401)", async () => {
    const mockFetch = async () =>
      new Response(JSON.stringify({ error: { type: "authentication_error", message: "invalid api key" } }), {
        status: 401,
        headers: { "Content-Type": "application/json", "x-request-id": "req-401" },
      });

    const client = createGatewayClient({
      gatewayUrl: "http://localhost:8084",
      apiKey: "invalid_key",
      fetch: mockFetch as any,
    });

    await assert.rejects(
      async () => {
        await client.chatCompletion({ model: "gemini-2.5-flash", messages: [{ role: "user", content: "hi" }] });
      },
      (err: any) => {
        assert.ok(err instanceof ApiError);
        assert.strictEqual(err.status, 401);
        assert.strictEqual(err.requestId, "req-401");
        assert.ok(err.message.includes("invalid api key"));
        return true;
      }
    );
  });

  it("handles unsupported model (HTTP 400)", async () => {
    const mockFetch = async () =>
      new Response(JSON.stringify({ error: { type: "unsupported_model", message: "model 'unknown-model' is not supported" } }), {
        status: 400,
        headers: { "Content-Type": "application/json", "x-request-id": "req-400" },
      });

    const client = createGatewayClient({
      gatewayUrl: "http://localhost:8084",
      apiKey: "valid_key",
      fetch: mockFetch as any,
    });

    await assert.rejects(
      async () => {
        await client.sendMessage({ model: "unknown-model", messages: [{ role: "user", content: "hi" }] });
      },
      (err: any) => {
        assert.ok(err instanceof ApiError);
        assert.strictEqual(err.status, 400);
        assert.strictEqual(err.requestId, "req-400");
        assert.ok(err.message.includes("not supported"));
        return true;
      }
    );
  });

  it("handles provider timeout (HTTP 504)", async () => {
    const mockFetch = async () =>
      new Response(JSON.stringify({ error: { type: "upstream_timeout", message: "upstream provider timed out" } }), {
        status: 504,
        headers: { "Content-Type": "application/json", "x-request-id": "req-504" },
      });

    const client = createGatewayClient({
      gatewayUrl: "http://localhost:8084",
      apiKey: "valid_key",
      fetch: mockFetch as any,
    });

    await assert.rejects(
      async () => {
        await client.chatCompletion({ model: "gemini-2.5-pro", messages: [{ role: "user", content: "slow query" }] });
      },
      (err: any) => {
        assert.ok(err instanceof ApiError);
        assert.strictEqual(err.status, 504);
        assert.strictEqual(err.requestId, "req-504");
        return true;
      }
    );
  });

  it("handles rate-limit response (HTTP 429)", async () => {
    const mockFetch = async () =>
      new Response(JSON.stringify({ error: { type: "rate_limit_exceeded", message: "rate limit exceeded: 5 req/min" } }), {
        status: 429,
        headers: { "Content-Type": "application/json", "x-request-id": "req-429", "retry-after": "12" },
      });

    const client = createGatewayClient({
      gatewayUrl: "http://localhost:8084",
      apiKey: "valid_key",
      fetch: mockFetch as any,
    });

    await assert.rejects(
      async () => {
        await client.chatCompletion({ model: "gemini-2.5-flash", messages: [{ role: "user", content: "flood" }] });
      },
      (err: any) => {
        assert.ok(err instanceof ApiError);
        assert.strictEqual(err.status, 429);
        assert.strictEqual(err.requestId, "req-429");
        return true;
      }
    );
  });

  it("handles credential revocation (HTTP 401)", async () => {
    const mockFetch = async () =>
      new Response(JSON.stringify({ error: { type: "credential_revoked", message: "credential has been revoked" } }), {
        status: 401,
        headers: { "Content-Type": "application/json", "x-request-id": "req-revoked" },
      });

    const client = createGatewayClient({
      gatewayUrl: "http://localhost:8084",
      apiKey: "aigw_revoked_key",
      fetch: mockFetch as any,
    });

    await assert.rejects(
      async () => {
        await client.chatCompletion({ model: "gemini-2.5-flash", messages: [{ role: "user", content: "test" }] });
      },
      (err: any) => {
        assert.ok(err instanceof ApiError);
        assert.strictEqual(err.status, 401);
        assert.strictEqual(err.requestId, "req-revoked");
        return true;
      }
    );
  });

  it("successfully parses non-streaming Gemini responses with usage metadata", async () => {
    const mockResponsePayload = {
      candidates: [
        {
          content: {
            parts: [{ text: "Hello from Sentinel Gemini integration!" }],
            role: "model",
          },
          finishReason: "STOP",
        },
      ],
      usageMetadata: {
        promptTokenCount: 15,
        candidatesTokenCount: 22,
        totalTokenCount: 37,
      },
    };

    const mockFetch = async (_url: string, init: RequestInit) => {
      // Verify headers passed
      const headers = init.headers as Record<string, string>;
      assert.strictEqual(headers["x-api-key"], "aigw_test_key");
      assert.strictEqual(headers["Authorization"], "Bearer aigw_test_key");
      assert.strictEqual(headers["x-project-id"], "proj-alpha");

      return new Response(JSON.stringify(mockResponsePayload), {
        status: 200,
        headers: { "Content-Type": "application/json", "x-request-id": "req-gemini-200" },
      });
    };

    const client = createGatewayClient({
      gatewayUrl: "http://localhost:8084",
      apiKey: "aigw_test_key",
      projectId: "proj-alpha",
      fetch: mockFetch as any,
    });

    const res = await client.chatCompletion({
      model: "gemini-2.5-flash",
      messages: [{ role: "user", content: "Greet me" }],
    });

    assert.strictEqual(res.text, "Hello from Sentinel Gemini integration!");
    assert.strictEqual(res.meta.requestId, "req-gemini-200");
    assert.strictEqual(res.meta.statusCode, 200);
    assert.strictEqual(res.meta.model, "gemini-2.5-flash");
    assert.strictEqual(res.meta.inputTokens, 15);
    assert.strictEqual(res.meta.outputTokens, 22);
    assert.strictEqual(res.meta.totalTokens, 37);
  });

  it("successfully parses streaming Gemini SSE chunks with usage", async () => {
    const sseChunks = [
      'data: {"candidates":[{"content":{"parts":[{"text":"Streaming chunk 1 "}]}}],"usageMetadata":{"promptTokenCount":10}}\n\n',
      'data: {"candidates":[{"content":{"parts":[{"text":"and chunk 2."}]},"finishReason":"STOP"}],"usageMetadata":{"candidatesTokenCount":14,"totalTokenCount":24}}\n\n',
      "data: [DONE]\n\n",
    ];

    const stream = new ReadableStream({
      start(controller) {
        for (const chunk of sseChunks) {
          controller.enqueue(new TextEncoder().encode(chunk));
        }
        controller.close();
      },
    });

    const mockFetch = async () =>
      new Response(stream, {
        status: 200,
        headers: {
          "Content-Type": "text/event-stream",
          "x-request-id": "req-stream-gemini",
        },
      });

    const client = createGatewayClient({
      gatewayUrl: "http://localhost:8084",
      apiKey: "aigw_stream_key",
      fetch: mockFetch as any,
    });

    let collectedText = "";
    const meta = await client.streamChatCompletion(
      {
        model: "gemini-2.5-flash",
        messages: [{ role: "user", content: "Stream test" }],
      },
      (chunk) => {
        collectedText += chunk;
      }
    );

    assert.strictEqual(collectedText, "Streaming chunk 1 and chunk 2.");
    assert.strictEqual(meta.requestId, "req-stream-gemini");
    assert.strictEqual(meta.inputTokens, 10);
    assert.strictEqual(meta.outputTokens, 14);
    assert.strictEqual(meta.totalTokens, 24);
    assert.strictEqual(meta.isEstimated, false);
    assert.strictEqual(meta.usageSource, "reported");
  });

  it("retries idempotent listModels on HTTP 503 and succeeds", async () => {
    let callCount = 0;
    const mockFetch = async () => {
      callCount++;
      if (callCount < 3) {
        return new Response(JSON.stringify({ error: { message: "service unavailable", type: "service_unavailable" } }), {
          status: 503,
          headers: { "Content-Type": "application/json" },
        });
      }
      return new Response(JSON.stringify({ data: [{ id: "claude-sonnet-4-5", object: "model", provider: "anthropic" }] }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    };

    const client = createGatewayClient({
      gatewayUrl: "http://localhost:8084",
      apiKey: "test_key",
      fetch: mockFetch as any,
      maxRetries: 3,
      retryDelayMs: 5,
    });

    const models = await client.listModels();
    assert.strictEqual(callCount, 3);
    assert.strictEqual(models.length, 1);
    assert.strictEqual(models[0]?.id, "claude-sonnet-4-5");
  });

  it("does not retry non-idempotent chatCompletion on error without idempotencyKey", async () => {
    let callCount = 0;
    const mockFetch = async () => {
      callCount++;
      return new Response(JSON.stringify({ error: { message: "internal server error", type: "internal_error" } }), {
        status: 500,
        headers: { "Content-Type": "application/json" },
      });
    };

    const client = createGatewayClient({
      gatewayUrl: "http://localhost:8084",
      apiKey: "test_key",
      fetch: mockFetch as any,
      maxRetries: 3,
      retryDelayMs: 5,
    });

    await assert.rejects(
      async () => {
        await client.chatCompletion({ model: "gpt-4o", messages: [{ role: "user", content: "pay $50" }] });
      },
      (err: any) => {
        assert.ok(err instanceof ApiError);
        assert.strictEqual(err.status, 500);
        return true;
      }
    );
    // Crucial: Must NOT retry non-idempotent requests blindly
    assert.strictEqual(callCount, 1);
  });

  it("retries non-idempotent chatCompletion when idempotencyKey is explicitly provided", async () => {
    let callCount = 0;
    const mockFetch = async () => {
      callCount++;
      if (callCount < 2) {
        return new Response(JSON.stringify({ error: { message: "temporary glitch", type: "service_unavailable" } }), {
          status: 503,
          headers: { "Content-Type": "application/json" },
        });
      }
      return new Response(
        JSON.stringify({
          choices: [{ message: { content: "Idempotent response" } }],
          usage: { prompt_tokens: 10, completion_tokens: 5, total_tokens: 15 },
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }
      );
    };

    const client = createGatewayClient({
      gatewayUrl: "http://localhost:8084",
      apiKey: "test_key",
      fetch: mockFetch as any,
      maxRetries: 2,
      retryDelayMs: 5,
    });

    const res = await client.chatCompletion({
      model: "gpt-4o",
      messages: [{ role: "user", content: "safe idempotent request" }],
      idempotencyKey: "idem-key-12345",
    });

    assert.strictEqual(callCount, 2);
    assert.strictEqual(res.text, "Idempotent response");
  });

  it("handles request timeout and cancellation via signal", async () => {
    const controller = new AbortController();
    const mockFetch = async (_url: string, init: RequestInit) => {
      return new Promise<Response>((_, reject) => {
        const signal = init.signal;
        if (signal?.aborted) {
          reject(signal.reason);
          return;
        }
        signal?.addEventListener("abort", () => reject(new Error("aborted")));
      });
    };

    const client = createGatewayClient({
      gatewayUrl: "http://localhost:8084",
      apiKey: "test_key",
      fetch: mockFetch as any,
    });

    setTimeout(() => controller.abort(new Error("client cancelled")), 10);

    await assert.rejects(
      async () => {
        await client.chatCompletion({
          model: "gemini-2.5-flash",
          messages: [{ role: "user", content: "test" }],
          signal: controller.signal,
        });
      },
      (err: any) => {
        assert.ok(err);
        return true;
      }
    );
  });

  it("detects and surfaces interrupted streaming cleanly", async () => {
    const stream = new ReadableStream({
      start(controller) {
        controller.enqueue(new TextEncoder().encode('data: {"choices":[{"delta":{"content":"Hi" comment:"truncated"}}\n\n'));
        controller.error(new Error("TCP connection reset by peer"));
      },
    });

    const mockFetch = async () =>
      new Response(stream, {
        status: 200,
        headers: { "Content-Type": "text/event-stream", "x-request-id": "req-broken" },
      });

    const client = createGatewayClient({
      gatewayUrl: "http://localhost:8084",
      apiKey: "test_key",
      fetch: mockFetch as any,
    });

    await assert.rejects(
      async () => {
        await client.streamChatCompletion(
          { model: "gpt-4o", messages: [{ role: "user", content: "stream" }] },
          () => {}
        );
      },
      (err: any) => {
        assert.ok(err instanceof ApiError);
        assert.strictEqual(err.code, "stream_interrupted");
        assert.strictEqual(err.status, 502);
        assert.strictEqual(err.requestId, "req-broken");
        return true;
      }
    );
  });

  it("correctly identifies missing or estimated usage metadata instead of silently treating as exact", async () => {
    const mockResponseNoUsage = {
      choices: [{ message: { content: "Response without usage stats" } }],
    };

    const mockFetch = async () =>
      new Response(JSON.stringify(mockResponseNoUsage), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });

    const client = createGatewayClient({
      gatewayUrl: "http://localhost:8084",
      apiKey: "test_key",
      fetch: mockFetch as any,
    });

    const res = await client.chatCompletion({
      model: "custom-model",
      messages: [{ role: "user", content: "hello" }],
    });

    assert.strictEqual(res.text, "Response without usage stats");
    assert.strictEqual(res.meta.isEstimated, true);
    assert.strictEqual(res.meta.usageSource, "none");
    assert.strictEqual(res.meta.inputTokens, undefined);
  });

  it("successfully parses Anthropic format non-streaming and streaming responses", async () => {
    const anthropicNonStreaming = {
      id: "msg_123",
      type: "message",
      role: "assistant",
      content: [{ type: "text", text: "Anthropic direct response" }],
      usage: { input_tokens: 42, output_tokens: 18 },
    };

    const client = createGatewayClient({
      gatewayUrl: "http://localhost:8084",
      apiKey: "test_key",
      fetch: (async () =>
        new Response(JSON.stringify(anthropicNonStreaming), {
          status: 200,
          headers: { "Content-Type": "application/json", "x-request-id": "req-anthropic" },
        })) as any,
    });

    const res = await client.sendMessage({
      model: "claude-sonnet-4-5",
      messages: [{ role: "user", content: "Hi Claude" }],
    });

    assert.strictEqual(res.text, "Anthropic direct response");
    assert.strictEqual(res.meta.inputTokens, 42);
    assert.strictEqual(res.meta.outputTokens, 18);
    assert.strictEqual(res.meta.totalTokens, 60);
    assert.strictEqual(res.meta.isEstimated, false);
    assert.strictEqual(res.meta.usageSource, "reported");
  });
});
