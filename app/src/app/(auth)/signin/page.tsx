"use client";

import { useActionState } from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { devSignIn, signIn, type AuthState } from "./actions";

const initial: AuthState = {};

export default function SignInPage() {
  const [credentials, submitCredentials, credentialsPending] = useActionState(signIn, initial);
  const [dev, submitDev, devPending] = useActionState(devSignIn, initial);
  const pending = credentialsPending || devPending;

  return (
    <main className="flex flex-1 items-center justify-center p-6">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>Sign in</CardTitle>
          <CardDescription>Access the AI Governance console.</CardDescription>
        </CardHeader>

        <CardContent>
          <form action={submitCredentials} className="grid gap-3">
            <label className="grid gap-1.5 text-sm" htmlFor="email">
              Email
              <Input
                id="email"
                name="email"
                type="email"
                autoComplete="username"
                required
              />
            </label>

            <label className="grid gap-1.5 text-sm" htmlFor="password">
              Password
              <Input
                id="password"
                name="password"
                type="password"
                autoComplete="current-password"
                required
              />
            </label>

            {credentials.error ? (
              <p role="alert" className="text-sm text-destructive">
                {credentials.error}
              </p>
            ) : null}

            <Button type="submit" disabled={pending}>
              {credentialsPending ? "Signing in…" : "Sign in"}
            </Button>
          </form>

          {/* Local development escape hatch: the backend provisions the account
              on first use, so a fresh environment needs no seeded credentials. */}
          <form action={submitDev} className="mt-3">
            <input type="hidden" name="email" value="developer@localhost" />
            <Button
              type="submit"
              variant="outline"
              className="w-full"
              disabled={pending}
            >
              {devPending ? "Starting session…" : "Continue as developer"}
            </Button>
          </form>

          {dev.error ? (
            <p role="alert" className="mt-3 text-sm text-destructive">
              {dev.error}
            </p>
          ) : null}

          <div className="mt-4 rounded-md border border-border/60 bg-muted/30 p-2.5 text-xs text-muted-foreground">
            <span className="font-semibold text-foreground">Demo credentials:</span>
            <div className="mt-1 flex flex-col gap-0.5 font-mono text-[11px]">
              <div>Email: <span className="text-foreground">admin@acme.example</span></div>
              <div>Password: <span className="text-foreground">SentinelPassword123!</span></div>
            </div>
          </div>

          <p className="mt-4 text-sm text-muted-foreground">
            Don&apos;t have an account?{" "}
            <Link href="/signup" className="underline-offset-4 hover:underline">
              Create account
            </Link>
          </p>

          <p className="mt-2 text-xs text-muted-foreground">
            <Link href="/" className="underline-offset-4 hover:underline">
              Back to site
            </Link>
          </p>
        </CardContent>
      </Card>
    </main>
  );
}