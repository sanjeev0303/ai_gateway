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
      <Card className="w-full max-w-sm border-none bg-transparent shadow-none ring-0">
        <CardHeader>
          <CardTitle className="text-2xl tracking-[-0.025em]">Sign in</CardTitle>
          <CardDescription>Welcome back. Sign in to the Tollseal console.</CardDescription>
        </CardHeader>

        <CardContent>
          <form action={submitCredentials} className="grid gap-4">
            <label className="grid gap-1.5 text-sm font-medium" htmlFor="email">
              Email
              <Input
                id="email"
                name="email"
                type="email"
                autoComplete="username"
                required
              />
            </label>

            <label className="grid gap-1.5 text-sm font-medium" htmlFor="password">
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

          <div className="mt-5 rounded-xl border border-border bg-muted/50 p-3 text-xs text-muted-foreground">
            <span className="font-semibold text-foreground">Demo credentials:</span>
            <div className="mt-1 flex flex-col gap-0.5 font-mono text-xs">
              <div>Email: <span className="text-foreground">admin@acme.example</span></div>
              <div>Password: <span className="text-foreground">SentinelPassword123!</span></div>
            </div>
          </div>

          <p className="mt-4 text-sm text-muted-foreground">
            Don&apos;t have an account?{" "}
            <Link href="/signup" className="font-medium text-primary underline-offset-4 hover:underline">
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