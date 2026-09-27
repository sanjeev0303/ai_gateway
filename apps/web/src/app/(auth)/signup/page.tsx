import Link from "next/link";
import type { Metadata } from "next";

import AuthForm from "../auth-form";

export const metadata: Metadata = {
  title: "Create account",
};

export default function SignupPage() {
  return (
    <>
      <div className="auth-head">
        <p className="auth-eyebrow">Control plane</p>
        <h1 className="auth-title">Create your workspace</h1>
        <p className="auth-lead">
          Set up an organization to start governing provider access and AI spend.
        </p>
      </div>
      <AuthForm mode="signup" />
      <p className="auth-note">
        You become the first administrator of the new organization.{" "}
        <Link href="/docs/api">Read the API reference</Link> for the endpoints behind
        this page.
      </p>
      <p className="auth-alt">
        Already have an account? <Link href="/login">Sign in</Link>
      </p>
    </>
  );
}