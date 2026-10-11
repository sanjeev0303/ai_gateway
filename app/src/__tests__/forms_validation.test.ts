import test from "node:test";
import assert from "node:assert/strict";

interface CredentialFormInput {
  name: string;
  subject_type: "user" | "service_account";
  subject_id: string;
  project_id?: string;
  team_id?: string;
  scopes?: string[];
}

function validateCredentialForm(input: CredentialFormInput): string | null {
  if (!input.name || input.name.trim().length === 0) {
    return "Credential name is required";
  }
  if (!input.subject_type || !["user", "service_account"].includes(input.subject_type)) {
    return "Invalid subject type";
  }
  if (!input.subject_id || input.subject_id.trim().length === 0) {
    return "Subject ID is required";
  }
  return null;
}

interface BudgetFormInput {
  amount: number;
  period: "monthly" | "daily";
  currency?: string;
  alert_threshold?: number;
}

function validateBudgetForm(input: BudgetFormInput): string | null {
  if (input.amount <= 0) {
    return "Budget amount must be greater than zero";
  }
  if (!["monthly", "daily"].includes(input.period)) {
    return "Invalid budget period";
  }
  if (input.alert_threshold !== undefined && (input.alert_threshold < 0 || input.alert_threshold > 100)) {
    return "Alert threshold must be between 0 and 100 percent";
  }
  return null;
}

interface InviteFormInput {
  email: string;
  role: "admin" | "member" | "billing_admin" | "auditor";
}

function validateInviteForm(input: InviteFormInput): string | null {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  if (!input.email || !emailRegex.test(input.email.trim())) {
    return "Valid email address is required";
  }
  if (!["admin", "member", "billing_admin", "auditor"].includes(input.role)) {
    return "Invalid role assigned";
  }
  return null;
}

test("Forms & Validation: credential creation validates inputs", () => {
  assert.equal(validateCredentialForm({ name: "", subject_type: "service_account", subject_id: "sa_1" }), "Credential name is required");
  assert.equal(validateCredentialForm({ name: "API Key", subject_type: "invalid" as any, subject_id: "sa_1" }), "Invalid subject type");
  assert.equal(validateCredentialForm({ name: "API Key", subject_type: "service_account", subject_id: "" }), "Subject ID is required");
  assert.equal(validateCredentialForm({ name: "API Key", subject_type: "service_account", subject_id: "sa_1" }), null);
});

test("Forms & Validation: budget creation validates amounts and thresholds", () => {
  assert.equal(validateBudgetForm({ amount: 0, period: "monthly" }), "Budget amount must be greater than zero");
  assert.equal(validateBudgetForm({ amount: -50, period: "monthly" }), "Budget amount must be greater than zero");
  assert.equal(validateBudgetForm({ amount: 100, period: "yearly" as any }), "Invalid budget period");
  assert.equal(validateBudgetForm({ amount: 100, period: "monthly", alert_threshold: 120 }), "Alert threshold must be between 0 and 100 percent");
  assert.equal(validateBudgetForm({ amount: 100, period: "monthly", alert_threshold: 80 }), null);
});

test("Forms & Validation: member invitation validates email and role", () => {
  assert.equal(validateInviteForm({ email: "invalid-email", role: "member" }), "Valid email address is required");
  assert.equal(validateInviteForm({ email: "user@example.com", role: "superadmin" as any }), "Invalid role assigned");
  assert.equal(validateInviteForm({ email: "user@example.com", role: "admin" }), null);
  assert.equal(validateInviteForm({ email: "billing@example.com", role: "billing_admin" }), null);
});
