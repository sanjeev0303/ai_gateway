"use client";

import * as React from "react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ShieldCheck, ArrowRight, UserCheck, KeyRound, AlertCircle } from "lucide-react";
import { acceptInvitationOnly, authAndAcceptInvitation, type AcceptInviteState } from "./actions";

interface AcceptInviteClientProps {
  token: string;
  user: {
    name: string;
    email: string;
    role: string;
  } | null;
}

export function AcceptInviteClient({ token, user }: AcceptInviteClientProps) {
  const [error, setError] = React.useState<string | null>(null);
  const [isPending, setIsPending] = React.useState(false);

  if (!token) {
    return (
      <Card className="w-full max-w-md border-destructive/30">
        <CardHeader>
          <div className="flex items-center gap-2 text-destructive">
            <AlertCircle className="size-5" />
            <CardTitle>Missing Invitation Token</CardTitle>
          </div>
          <CardDescription>
            No invitation token was provided in the URL. Please verify the invitation link sent by your administrator.
          </CardDescription>
        </CardHeader>
      </Card>
    );
  }

  // If user is already authenticated
  if (user) {
    const handleDirectAccept = async () => {
      try {
        setIsPending(true);
        setError(null);
        const res = await acceptInvitationOnly(token);
        if (res?.error) setError(res.error);
      } catch {
        setError("Failed to accept invitation.");
      } finally {
        setIsPending(false);
      }
    };

    return (
      <Card className="w-full max-w-md border-border/80 shadow-lg">
        <CardHeader>
          <div className="size-10 rounded-xl bg-primary/10 text-primary flex items-center justify-center mb-2">
            <ShieldCheck className="size-6" />
          </div>
          <CardTitle>Accept Organization Invitation</CardTitle>
          <CardDescription>
            You have been invited to join an organization on Sentinel AI Gateway.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="rounded-lg border bg-muted/30 p-3 text-sm space-y-1">
            <p className="text-xs text-muted-foreground uppercase tracking-wider font-semibold">
              Signed in as
            </p>
            <p className="font-medium text-foreground">{user.email}</p>
            <p className="text-xs text-muted-foreground">Current role: {user.role}</p>
          </div>

          {error && (
            <p role="alert" className="text-sm text-destructive bg-destructive/10 p-2.5 rounded-md border border-destructive/20">
              {error}
            </p>
          )}
        </CardContent>
        <CardFooter className="flex justify-end gap-2">
          <Button onClick={handleDirectAccept} disabled={isPending} className="w-full">
            {isPending ? "Joining organization…" : "Accept & Go to Onboarding"}
            <ArrowRight className="ml-2 size-4" />
          </Button>
        </CardFooter>
      </Card>
    );
  }

  // User is not authenticated: offer Sign In or Create Account
  return (
    <Card className="w-full max-w-md border-border/80 shadow-lg">
      <CardHeader>
        <div className="size-10 rounded-xl bg-primary/10 text-primary flex items-center justify-center mb-2">
          <KeyRound className="size-6" />
        </div>
        <CardTitle>Join Sentinel AI Gateway</CardTitle>
        <CardDescription>
          Sign in or create an account with your invited email to accept the invitation.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Tabs defaultValue="signin" className="w-full">
          <TabsList className="grid w-full grid-cols-2 mb-4">
            <TabsTrigger value="signin">Sign In & Join</TabsTrigger>
            <TabsTrigger value="signup">Create Account</TabsTrigger>
          </TabsList>

          <TabsContent value="signin">
            <form
              onSubmit={async (e) => {
                e.preventDefault();
                setIsPending(true);
                setError(null);
                const formData = new FormData(e.currentTarget);
                const res = await authAndAcceptInvitation(token, "signin", formData);
                if (res?.error) setError(res.error);
                setIsPending(false);
              }}
              className="space-y-3"
            >
              <div className="space-y-1">
                <Label htmlFor="signin-email">Email</Label>
                <Input id="signin-email" name="email" type="email" placeholder="you@company.com" required />
              </div>
              <div className="space-y-1">
                <Label htmlFor="signin-password">Password</Label>
                <Input id="signin-password" name="password" type="password" required />
              </div>

              {error && (
                <p role="alert" className="text-sm text-destructive bg-destructive/10 p-2.5 rounded-md border border-destructive/20">
                  {error}
                </p>
              )}

              <Button type="submit" disabled={isPending} className="w-full mt-2">
                {isPending ? "Authenticating…" : "Sign In & Accept Invitation"}
              </Button>
            </form>
          </TabsContent>

          <TabsContent value="signup">
            <form
              onSubmit={async (e) => {
                e.preventDefault();
                setIsPending(true);
                setError(null);
                const formData = new FormData(e.currentTarget);
                const res = await authAndAcceptInvitation(token, "signup", formData);
                if (res?.error) setError(res.error);
                setIsPending(false);
              }}
              className="space-y-3"
            >
              <div className="space-y-1">
                <Label htmlFor="signup-name">Full Name</Label>
                <Input id="signup-name" name="display_name" placeholder="Alex Chen" required />
              </div>
              <div className="space-y-1">
                <Label htmlFor="signup-email">Work Email</Label>
                <Input id="signup-email" name="email" type="email" placeholder="you@company.com" required />
              </div>
              <div className="space-y-1">
                <Label htmlFor="signup-password">Password</Label>
                <Input id="signup-password" name="password" type="password" required />
              </div>

              {error && (
                <p role="alert" className="text-sm text-destructive bg-destructive/10 p-2.5 rounded-md border border-destructive/20">
                  {error}
                </p>
              )}

              <Button type="submit" disabled={isPending} className="w-full mt-2">
                {isPending ? "Creating account…" : "Create Account & Accept"}
              </Button>
            </form>
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  );
}
