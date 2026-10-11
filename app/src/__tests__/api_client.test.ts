import test from "node:test";
import assert from "node:assert/strict";
import { createTransport, ApiError } from "@ai-governance/api-client";

test("API Client: constructs requests with proper baseUrl and headers", async () => {
  let capturedUrl = "";
  let capturedHeaders: Headers | undefined;

  const mockFetch = async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    capturedUrl = input.toString();
    capturedHeaders = new Headers(init?.headers);
    return new Response(JSON.stringify({ ok: true, data: [] }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });
  };

  const transport = createTransport({
    baseUrl: "http://localhost:8081",
    fetch: mockFetch as any,
    cookie: "session=xyz123",
    csrfToken: "csrf-token-abc",
  });

  const res = await transport.request("/v1/projects" as any, {
    method: "POST",
    body: { name: "Test Project" },
  });

  assert.equal(capturedUrl, "http://localhost:8081/v1/projects");
  assert.equal(capturedHeaders?.get("Cookie"), "session=xyz123");
  assert.equal(capturedHeaders?.get("X-CSRF-Token"), "csrf-token-abc");
  assert.equal((res as any).ok, true);
});

test("API Client: formats query parameters correctly", async () => {
  let capturedUrl = "";

  const mockFetch = async (input: RequestInfo | URL): Promise<Response> => {
    capturedUrl = input.toString();
    return new Response(JSON.stringify({ items: [] }), { status: 200 });
  };

  const transport = createTransport({
    baseUrl: "http://localhost:8081",
    fetch: mockFetch as any,
  });

  await transport.request("/v1/usage/summary" as any, {
    method: "GET",
    query: {
      range: "7d",
      limit: 50,
    },
  });

  assert.ok(capturedUrl.includes("range=7d"));
  assert.ok(capturedUrl.includes("limit=50"));
});

test("API Client: maps 401 Unauthorized with correct ApiError", async () => {
  const mockFetch = async (): Promise<Response> => {
    return new Response(JSON.stringify({ error: { message: "Invalid session or token", code: "unauthorized" } }), {
      status: 401,
      headers: { "Content-Type": "application/json" },
    });
  };

  const transport = createTransport({
    baseUrl: "http://localhost:8081",
    fetch: mockFetch as any,
  });

  await assert.rejects(
    async () => {
      await transport.request("/v1/organization" as any, {
        method: "GET",
      });
    },
    (err: any) => {
      assert.ok(err instanceof ApiError);
      assert.equal(err.status, 401);
      assert.equal(err.code, "unauthorized");
      return true;
    }
  );
});

test("API Client: maps 403 Forbidden with policy rejection details", async () => {
  const mockFetch = async (): Promise<Response> => {
    return new Response(
      JSON.stringify({
        error: {
          code: "policy_violation",
          message: "Model access blocked by organization governance policy",
        },
      }),
      { status: 403, headers: { "Content-Type": "application/json" } }
    );
  };

  const transport = createTransport({
    baseUrl: "http://localhost:8081",
    fetch: mockFetch as any,
  });

  await assert.rejects(
    async () => {
      await transport.request("/v1/messages" as any, {
        method: "POST",
      });
    },
    (err: any) => {
      assert.ok(err instanceof ApiError);
      assert.equal(err.status, 403);
      assert.equal(err.code, "policy_violation");
      assert.ok(err.message.includes("Model access blocked"));
      return true;
    }
  );
});

test("API Client: maps 404 Not Found error", async () => {
  const mockFetch = async (): Promise<Response> => {
    return new Response(
      JSON.stringify({ error: { code: "not_found", message: "Resource not found" } }),
      { status: 404, headers: { "Content-Type": "application/json" } }
    );
  };

  const transport = createTransport({
    baseUrl: "http://localhost:8081",
    fetch: mockFetch as any,
  });

  await assert.rejects(
    async () => {
      await transport.request("/v1/projects/nonexistent" as any, {
        method: "GET",
      });
    },
    (err: any) => {
      assert.ok(err instanceof ApiError);
      assert.equal(err.status, 404);
      assert.equal(err.code, "not_found");
      return true;
    }
  );
});
