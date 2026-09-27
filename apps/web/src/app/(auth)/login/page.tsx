import Link from "next/link";
import type { Metadata } from "next";

import AuthForm from "../auth-form";

export const metadata: Metadata = {
  title: "Sign in",
};

export default function LoginPage() {
  return (
    <>
      <div className="auth-head">
        <p className="auth-eyebrow">Control plane</p>
        <h1 className="auth-title">Sign in</h1>
        <p className="auth-lead">Use your work email and password to open your control plane.</p>
      </div>
      <AuthForm mode="signin" />
      <p className="auth-note">
        Sessions use an HttpOnly cookie, so signing in on one tab covers the rest
        of the workspace. <Link href="/docs/api">API reference</Link>
      </p>
      <p className="auth-alt">
        No account yet? <Link href="/signup">Create a workspace</Link>
      </p>
    </>
  );
}