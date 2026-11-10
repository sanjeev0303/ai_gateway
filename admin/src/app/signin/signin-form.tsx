"use client";

import * as React from "react";
import { useRouter } from "next/navigation";

import { Spinner } from "@ai-governance/console-ui/dashboard/state";
import { Button } from "@ai-governance/console-ui/ui/button";
import { Input } from "@ai-governance/console-ui/ui/input";
import { Label } from "@ai-governance/console-ui/ui/label";

/**
 * Posts to the control API through this app's /api/v1 proxy, so the session
 * cookie the backend sets lands on the console's own origin.
 */
export function SignInForm() {
  const router = useRouter();
  const [error, setError] = React.useState<string | null>(null);
  const [pending, startTransition] = React.useTransition();

  const submit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const email = String(form.get("email") ?? "").trim();
    const password = String(form.get("password") ?? "");
    if (!email || !password) {
      setError("Enter your email and password.");
      return;
    }
    setError(null);
    startTransition(async () => {
      let response: Response;
      try {
        response = await fetch("/api/v1/auth/signin", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ email, password }),
          credentials: "include",
        });
      } catch {
        setError("Cannot reach the control API. Check that it is running.");
        return;
      }
      if (!response.ok) {
        let message = "Sign-in failed. Check your email and password.";
        try {
          const body = (await response.json()) as { error?: { message?: string } };
          if (body.error?.message) message = body.error.message;
        } catch {
          // keep the generic message
        }
        setError(message);
        return;
      }
      router.replace("/");
      router.refresh();
    });
  };

  return (
    <form onSubmit={submit} className="flex flex-col gap-4" noValidate>
      <div className="flex flex-col gap-2">
        <Label htmlFor="email">Email</Label>
        <Input id="email" name="email" type="email" autoComplete="username" spellCheck={false} required />
      </div>
      <div className="flex flex-col gap-2">
        <Label htmlFor="password">Password</Label>
        <Input id="password" name="password" type="password" autoComplete="current-password" required />
      </div>
      {error && (
        <p role="alert" className="text-sm text-rose-700 dark:text-rose-400">
          {error}
        </p>
      )}
      <Button type="submit" disabled={pending} className="w-full">
        {pending && <Spinner />}
        Sign in
      </Button>
    </form>
  );
}
