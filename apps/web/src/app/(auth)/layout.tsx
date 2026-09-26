import Link from "next/link";
import type { ReactNode } from "react";

import { BrandMark } from "@/components/icons";
import { site } from "@/lib/site";

import "./auth.css";

export default function AuthLayout({ children }: { readonly children: ReactNode }) {
  return (
    <div className="auth-shell">
      <div className="auth-panel">
        <Link href="/" className="auth-brand">
          <BrandMark width={18} height={18} />
          {site.name}
        </Link>
        <main className="auth-card">{children}</main>
      </div>
    </div>
  );
}