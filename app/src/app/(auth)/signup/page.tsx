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
import { signUp, type AuthState } from "../signin/actions";

const initial: AuthState = {};

export default function SignUpPage() {
  const [state, submit, pending] = useActionState(signUp, initial);

  return (
    <main className="flex flex-1 items-center justify-center p-6">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>Create account</CardTitle>
          <CardDescription>Set up your organization on the AI Governance console.</CardDescription>
        </CardHeader>

        <CardContent>
          <form action={submit} className="grid gap-3">
            <label className="grid gap-1.5 text-sm" htmlFor="display_name">
              Name
              <Input
                id="display_name"
                name="display_name"
                autoComplete="name"
                required
              />
            </label>

            <label className="grid gap-1.5 text-sm" htmlFor="organization_name">
              Organization
              <Input
                id="organization_name"
                name="organization_name"
                autoComplete="organization"
                required
              />
            </label>

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
                autoComplete="new-password"
                required
              />
            </label>

            {state.error ? (
              <p role="alert" className="text-sm text-destructive">
                {state.error}
              </p>
            ) : null}

            <Button type="submit" disabled={pending}>
              {pending ? "Creating account…" : "Create account"}
            </Button>
          </form>

          <p className="mt-4 text-sm text-muted-foreground">
            Already have an account?{" "}
            <Link href="/signin" className="underline-offset-4 hover:underline">
              Sign in
            </Link>
          </p>
        </CardContent>
      </Card>
    </main>
  );
}
