import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = {
  title: "Govern enterprise AI usage",
};

export default function MarketingHome() {
  return (
    <section className="hero">
      <h1>One gateway for governed AI access</h1>
      <p>
        Route every LLM call through a gateway that enforces scoped credentials,
        budgets, and policies — then keep audit-grade usage records of what
        happened.
      </p>
      <div className="hero-actions">
        <Link href="/signup" className="btn">
          Start free
        </Link>
        <Link href="/login" className="btn btn-ghost">
          Sign in
        </Link>
      </div>
    </section>
  );
}