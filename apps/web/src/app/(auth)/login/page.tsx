import Link from "next/link";
import type { Metadata } from "next";

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
      <form className="auth-form" method="post">
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
            autoComplete="current-password"
            required
          />
        </div>
        <button className="auth-submit" type="submit" disabled>
          Sign in
        </button>
      </form>
      <p className="auth-note">
        Sign-in is not connected yet. The control API exposes dev login for local use.{" "}
        <Link href="/docs/api">API reference</Link>
      </p>
      <p className="auth-alt">
        No account yet? <Link href="/signup">Create a workspace</Link>
      </p>
    </>
  );
}