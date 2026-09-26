import Link from "next/link";
import type { ReactNode } from "react";

export default function AuthLayout({ children }: { readonly children: ReactNode }) {
  return (
    <div className="auth-shell">
      <main className="card auth-card">{children}</main>
      <p className="auth-back">
        <Link href="/">← Back to home</Link>
      </p>
    </div>
  );
}