"use client";

import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";

import { ApiError, signIn, signUp } from "../../lib/control-api";

const MIN_PASSWORD_LENGTH = 12;

type Mode = "signin" | "signup";

export default function AuthForm({ mode }: { readonly mode: Mode }) {
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    const email = String(data.get("email") ?? "").trim();
    const password = String(data.get("password") ?? "");

    setPending(true);
    setError(null);
    try {
      if (mode === "signup") {
        const organization = String(data.get("organization") ?? "").trim();
        await signUp({ email, password, organization_name: organization });
      } else {
        await signIn(email, password);
      }
      router.replace("/dashboard");
      router.refresh();
    } catch (cause) {
      setPending(false);
      setError(messageFor(cause));
    }
  }

  return (
    <form className="auth-form" onSubmit={onSubmit}>
      {mode === "signup" ? (
        <div className="auth-field">
          <label htmlFor="organization">Organization name</label>
          <input
            id="organization"
            name="organization"
            type="text"
            autoComplete="organization"
            required
          />
        </div>
      ) : null}

      <div className="auth-field">
        <label htmlFor="email">Work email</label>
        <input id="email" name="email" type="email" autoComplete="email" required />
      </div>

      <div className="auth-field">
        <label htmlFor="password">Password</label>
        <input
          id="password"
          name="password"
          type="password"
          autoComplete={mode === "signup" ? "new-password" : "current-password"}
          minLength={MIN_PASSWORD_LENGTH}
          required
        />
        {mode === "signup" ? (
          <p className="auth-hint">Use at least {MIN_PASSWORD_LENGTH} characters.</p>
        ) : null}
      </div>

      {error !== null ? (
        <p className="auth-error" role="alert">
          {error}
        </p>
      ) : null}

      <button className="auth-submit" type="submit" disabled={pending}>
        {pending ? "Working" : mode === "signup" ? "Create workspace" : "Sign in"}
      </button>
    </form>
  );
}

function messageFor(cause: unknown): string {
  if (cause instanceof ApiError) {
    if (cause.status === 401) {
      return "That email and password combination did not match an account.";
    }
    if (cause.status === 429) {
      return "Too many attempts. Wait a minute and try again.";
    }
    if (cause.status === 409) {
      return "An account already exists for that email.";
    }
    return cause.message;
  }
  return "Could not reach the control plane. Check your connection and try again.";
}
