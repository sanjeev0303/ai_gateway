"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { MenuIcon } from "@/components/icons";
import { getSession, type Session } from "@/lib/control-api";

interface NavLink {
  readonly href: string;
  readonly label: string;
}

/**
 * SessionNav swaps the signed-out CTAs for the signed-in ones. The session
 * cookie is HttpOnly and the API is a separate origin, so the server cannot
 * read it; the client asks instead.
 */
export function SessionNav({ links }: { readonly links: readonly NavLink[] }) {
  // ponytail: signed-out is the server-rendered default, so a signed-in
  // visitor sees one swap after hydration rather than a layout shift. Promote
  // to a server component that forwards the cookie if that flash ever matters.
  const [session, setSession] = useState<Session | null>(null);

  useEffect(() => {
    let active = true;
    // A 401 is the signed-out answer, not a failure, and it arrives as a
    // thrown ApiError just like a real outage would. Both mean "no session".
    getSession()
      .then((s) => {
        if (active) setSession(s);
      })
      .catch(() => {
        if (active) setSession(null);
      });
    return () => {
      active = false;
    };
  }, []);

  const identity = session ? labelFor(session) : null;

  return (
    <div className="nav-actions">
      {identity === null ? (
        <>
          <Link href="/login" className="nav-signin">
            Sign in
          </Link>
          <Link href="/signup" className="btn btn-primary btn-sm">
            Get started
          </Link>
        </>
      ) : (
        <>
          <span className="nav-user" title={identity.email}>
            <span className="nav-user-avatar" aria-hidden="true">
              {identity.initials}
            </span>
            <span className="nav-user-name">{identity.name}</span>
          </span>
          <Link href="/dashboard" className="btn btn-primary btn-sm">
            Dashboard
          </Link>
        </>
      )}
      <details className="nav-more">
        <summary className="nav-more-btn" aria-label="Open site navigation">
          <MenuIcon width={20} height={20} />
        </summary>
        <nav aria-label="Site" className="nav-more-panel">
          {links.map((link) => (
            <Link key={link.href} href={link.href} className="nav-more-link">
              {link.label}
            </Link>
          ))}
          <Link href={session ? "/dashboard" : "/login"} className="nav-more-link">
            {session ? "Dashboard" : "Sign in"}
          </Link>
        </nav>
      </details>
    </div>
  );
}

function labelFor(session: Session) {
  const name = session.claims.display_name.trim() || session.claims.email;
  const initials = name
    .split(/\s+/)
    .map((word) => word.charAt(0))
    .join("")
    .slice(0, 2)
    .toUpperCase();
  return { name, initials, email: session.claims.email };
}
