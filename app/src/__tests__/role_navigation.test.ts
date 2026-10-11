import test from "node:test";
import assert from "node:assert/strict";
import { hasPermission, hasAllPermissions, hasAnyPermission, Role, Permission } from "../lib/permissions.js";

test("Role & Navigation: owner has full administrative and billing permissions", () => {
  const role: Role = "owner";
  assert.ok(hasPermission(role, "org.manage"));
  assert.ok(hasPermission(role, "members.manage"));
  assert.ok(hasPermission(role, "billing.manage"));
  assert.ok(hasPermission(role, "credentials.manage"));
  assert.ok(hasPermission(role, "policies.manage"));
  assert.ok(hasPermission(role, "budgets.manage"));
});

test("Role & Navigation: admin has operational permissions but cannot manage billing", () => {
  const role: Role = "admin";
  assert.ok(hasPermission(role, "org.read"));
  assert.equal(hasPermission(role, "org.update"), false);
  assert.ok(hasPermission(role, "members.manage"));
  assert.ok(hasPermission(role, "credentials.manage"));
  assert.ok(hasPermission(role, "policies.manage"));
  assert.ok(hasPermission(role, "billing.read"));
  assert.equal(hasPermission(role, "billing.manage"), false);
  assert.equal(hasPermission(role, "org.manage"), false);
});

test("Role & Navigation: billing_admin is restricted to billing management", () => {
  const role: Role = "billing_admin";
  assert.ok(hasPermission(role, "billing.read"));
  assert.ok(hasPermission(role, "billing.manage"));
  assert.equal(hasPermission(role, "credentials.manage"), false);
  assert.equal(hasPermission(role, "policies.manage"), false);
  assert.equal(hasPermission(role, "members.manage"), false);
});

test("Role & Navigation: auditor has read-only access to audit logs and policies", () => {
  const role: Role = "auditor";
  assert.ok(hasPermission(role, "audit.read"));
  assert.ok(hasPermission(role, "security_events.read"));
  assert.ok(hasPermission(role, "policies.read"));
  assert.equal(hasPermission(role, "policies.manage"), false);
  assert.equal(hasPermission(role, "credentials.manage"), false);
  assert.equal(hasPermission(role, "members.manage"), false);
});

test("Role & Navigation: member cannot manage credentials, policies, or members", () => {
  const role: Role = "member";
  assert.ok(hasPermission(role, "usage.read"));
  assert.ok(hasPermission(role, "projects.read"));
  assert.equal(hasPermission(role, "credentials.manage"), false);
  assert.equal(hasPermission(role, "credentials.read"), false);
  assert.equal(hasPermission(role, "policies.read"), false);
  assert.equal(hasPermission(role, "policies.manage"), false);
  assert.equal(hasPermission(role, "billing.read"), false);
});

test("Role & Navigation: helper utilities work correctly", () => {
  assert.ok(hasAnyPermission("admin", ["billing.manage", "policies.manage"]));
  assert.equal(hasAnyPermission("member", ["billing.manage", "policies.manage"]), false);
  assert.ok(hasAllPermissions("owner", ["org.manage", "billing.manage", "policies.manage"]));
  assert.equal(hasAllPermissions("admin", ["org.manage", "billing.manage", "policies.manage"]), false);
});
