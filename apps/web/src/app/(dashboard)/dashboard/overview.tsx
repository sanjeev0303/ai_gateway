"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import {
  countCredentials,
  countProjects,
  countServiceAccounts,
  countTeams,
  getSession,
  listMembers,
  type Member,
  type SessionClaims,
} from "@/lib/control-api";

interface Totals {
  readonly members: readonly Member[] | null;
  readonly projects: number | null;
  readonly teams: number | null;
  readonly serviceAccounts: number | null;
  readonly credentials: number | null;
}

const settled = <T,>(r: PromiseSettledResult<T>): T | null => (r.status === "fulfilled" ? r.value : null);

const show = (n: number | null): string => (n === null ? "—" : String(n));

export function Overview() {
  const [claims, setClaims] = useState<SessionClaims | null>(null);
  const [totals, setTotals] = useState<Totals | null>(null);
  const [probed, setProbed] = useState(false);

  useEffect(() => {
    let live = true;
    void (async () => {
      let session;
      try {
        session = await getSession();
      } catch {
        return;
      } finally {
        if (live) setProbed(true);
      }
      if (!live) return;
      setClaims(session.claims);

      const [members, projects, teams, serviceAccounts, credentials] = await Promise.allSettled([
        listMembers(),
        countProjects(),
        countTeams(),
        countServiceAccounts(),
        countCredentials(),
      ]);
      if (live) {
        setTotals({
          members: settled(members)?.items ?? null,
          projects: settled(projects),
          teams: settled(teams),
          serviceAccounts: settled(serviceAccounts),
          credentials: settled(credentials),
        });
      }
    })();
    return () => {
      live = false;
    };
  }, []);

  if (!claims) {
    return (
      <section className="dash-panel">
        <h1>Overview</h1>
        {probed ? (
          <>
            <p className="dash-note">You need to be signed in to view this organization.</p>
            <p className="dash-note">
              <Link href="/login">Sign in</Link>
            </p>
          </>
        ) : (
          <p className="dash-note">Checking your session…</p>
        )}
      </section>
    );
  }

  return (
    <>
      <header className="dash-head">
        <div>
          <h1>Overview</h1>
          <p className="dash-note">
            {claims.display_name || claims.email} &middot; {claims.role} &middot; org{" "}
            <code>{claims.organization_id.slice(0, 8)}</code>
          </p>
        </div>
      </header>

      <section aria-label="Organization totals" className="dash-tiles">
        <Tile label="Members" value={show(totals?.members?.length ?? null)} />
        <Tile label="Projects" value={show(totals?.projects ?? null)} />
        <Tile label="Teams" value={show(totals?.teams ?? null)} />
        <Tile label="API credentials" value={show(totals?.credentials ?? null)} />
        <Tile label="Service accounts" value={show(totals?.serviceAccounts ?? null)} />
      </section>

      <section className="dash-panel dash-panel-muted">
        <h2>Usage, spend, and latency</h2>
        <p className="dash-note">
          Nothing to chart yet, and no numbers stand in for them. Usage accounting is schema-first but
          not wired: neither data plane writes usage events, and the async worker runs health probes
          only. These figures appear once request accounting is connected.
        </p>
      </section>

      <section className="dash-panel">
        <h2>Members</h2>
        <MembersTable totals={totals} />
      </section>
    </>
  );
}

function MembersTable({ totals }: { readonly totals: Totals | null }) {
  if (totals === null) return <p className="dash-note">Loading members…</p>;
  if (totals.members === null) return <p className="dash-note">Members could not be loaded.</p>;
  if (totals.members.length === 0)
    return <p className="dash-note">No members in this organization yet.</p>;

  return (
    <div className="dash-table-wrap">
      <table className="dash-table">
        <thead>
          <tr>
            <th scope="col">Name</th>
            <th scope="col">Email</th>
            <th scope="col">Role</th>
            <th scope="col">Status</th>
          </tr>
        </thead>
        <tbody>
          {totals.members.map((m) => (
            <tr key={m.user.id}>
              <td>{m.user.display_name || "—"}</td>
              <td>{m.user.email}</td>
              <td>{m.role}</td>
              <td>{m.user.status}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function Tile({ label, value }: { readonly label: string; readonly value: string }) {
  return (
    <div className="dash-tile">
      <span className="dash-tile-value">{value}</span>
      <span className="dash-tile-label">{label}</span>
    </div>
  );
}
