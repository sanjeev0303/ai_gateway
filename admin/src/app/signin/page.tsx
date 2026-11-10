import type { Metadata } from "next";

import { GuillocheBand } from "@ai-governance/console-ui/brand/guilloche";
import { Logo } from "@/components/logo";
import { SignInForm } from "./signin-form";
import { signOut } from "./actions";
import { Button } from "@ai-governance/console-ui/ui/button";

export const metadata: Metadata = { title: "Sign in · Platform console" };

export default async function SignInPage({ searchParams }: { searchParams: Promise<{ reason?: string }> }) {
  const { reason } = await searchParams;
  const notAdmin = reason === "not-admin";

  return (
    <main className="flex min-h-screen items-center justify-center bg-background px-4 py-10">
      <div aria-hidden className="ambient pointer-events-none fixed inset-x-0 top-0 h-[520px] opacity-80" />
      <section
        aria-labelledby="signin-heading"
        className="relative w-full max-w-sm overflow-hidden rounded-[20px] border border-border-strong bg-card"
      >
        <GuillocheBand className="opacity-70" />
        <div className="flex flex-col gap-6 px-6 pt-2 pb-6">
          <div className="flex flex-col gap-3">
            <Logo />
            <div className="flex flex-col gap-1">
              <h1 id="signin-heading" className="text-xl font-semibold tracking-[-0.02em] text-foreground">
                Platform console
              </h1>
              <p className="text-sm text-muted-foreground">
                For the people who run this Tollseal deployment. Customers use the dashboard.
              </p>
            </div>
          </div>

          {notAdmin ? (
            <div className="flex flex-col gap-3">
              <p role="alert" className="rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-800 dark:text-amber-300">
                This account is not a platform admin. Only addresses listed in PLATFORM_ADMIN_EMAILS can open the console.
              </p>
              <form action={signOut}>
                <Button type="submit" variant="outline" className="w-full">
                  Sign in with another account
                </Button>
              </form>
            </div>
          ) : (
            <SignInForm />
          )}
        </div>
      </section>
    </main>
  );
}
