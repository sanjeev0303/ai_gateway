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
  });
});
