import Link from "next/link";
import type { Metadata } from "next";

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
      <form className="auth-form" method="post">
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
            autoComplete="new-password"
            required
          />
        </div>
        <button className="auth-submit" type="submit" disabled>
          Create workspace
        </button>
      </form>
      <p className="auth-note">
        Workspace creation is not connected yet.{" "}
        <Link href="/docs/api">Read the API reference</Link> for the endpoints that exist
        today.
      </p>
      <p className="auth-alt">
        Already have an account? <Link href="/login">Sign in</Link>
      </p>
    </>
  );
}