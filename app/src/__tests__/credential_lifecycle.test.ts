import test from "node:test";
import assert from "node:assert/strict";

interface GatewayCredential {
  id: string;
  name: string;
  key_prefix: string;
  fingerprint: string;
  secret_token?: string; // Only present on create/rotate response
  status: "active" | "revoked";
  created_at: string;
  revoked_at?: string | null;
}

function handleCredentialCreateResponse(rawResponse: { id: string; name: string; key_prefix: string; raw_key: string }): {
  credential: GatewayCredential;
  oneTimeSecret: string;
} {
  return {
    credential: {
      id: rawResponse.id,
      name: rawResponse.name,
      key_prefix: rawResponse.key_prefix,
      fingerprint: `fp_${rawResponse.id.slice(0, 8)}`,
      status: "active",
      created_at: new Date().toISOString(),
    },
    oneTimeSecret: rawResponse.raw_key,
  };
}

function handleCredentialListResponse(items: Array<{ id: string; name: string; key_prefix: string; status: "active" | "revoked" }>): GatewayCredential[] {
  return items.map((item) => ({
    id: item.id,
    name: item.name,
    key_prefix: item.key_prefix,
    fingerprint: `fp_${item.id.slice(0, 8)}`,
    status: item.status,
    created_at: new Date().toISOString(),
    secret_token: undefined, // Never exposed on listing
  }));
}

function revokeCredentialState(creds: GatewayCredential[], targetId: string): GatewayCredential[] {
  return creds.map((c) =>
    c.id === targetId ? { ...c, status: "revoked", revoked_at: new Date().toISOString() } : c
  );
}

test("Credential Lifecycle: one-time raw secret is captured on create", () => {
  const result = handleCredentialCreateResponse({
    id: "cred-1",
    name: "Production Gateway Key",
    key_prefix: "aigw_live_ab12",
    raw_key: "aigw_live_ab12_SECRET_FULL_KEY_987654321",
  });

  assert.equal(result.oneTimeSecret, "aigw_live_ab12_SECRET_FULL_KEY_987654321");
  assert.equal(result.credential.key_prefix, "aigw_live_ab12");
  assert.equal(result.credential.status, "active");
  assert.equal(result.credential.secret_token, undefined);
});

test("Credential Lifecycle: list endpoints strictly mask secrets", () => {
  const list = handleCredentialListResponse([
    { id: "cred-1", name: "Prod Key", key_prefix: "aigw_live_ab12", status: "active" },
    { id: "cred-2", name: "Dev Key", key_prefix: "aigw_test_cd34", status: "active" },
  ]);

  for (const cred of list) {
    assert.equal(cred.secret_token, undefined);
    assert.ok(cred.key_prefix.startsWith("aigw_"));
    assert.ok(cred.fingerprint.startsWith("fp_"));
  }
});

test("Credential Lifecycle: revocation updates state immediately", () => {
  const list = handleCredentialListResponse([
    { id: "cred-1", name: "Prod Key", key_prefix: "aigw_live_ab12", status: "active" },
  ]);

  const updated = revokeCredentialState(list, "cred-1");
  assert.equal(updated[0].status, "revoked");
  assert.ok(updated[0].revoked_at !== null);
});
