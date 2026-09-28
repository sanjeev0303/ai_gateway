import { UnavailablePanel } from "@/lib/metrics";

export interface SpecGroup {
  readonly title: string;
  readonly items: readonly string[];
}

export function SpecView({
  blurb,
  needs,
  groups,
  note,
}: {
  readonly blurb: string;
  readonly needs: readonly string[];
  readonly groups: readonly SpecGroup[];
  readonly note?: string;
}) {
  return (
    <>
      <p className="dash-note">{blurb}</p>
      <UnavailablePanel what="This view" needs={needs} />
      <div className="spec">
        {groups.map((group) => (
          <div key={group.title} className="spec-group">
            <p className="spec-title">{group.title}</p>
            <ul className="spec-items">
              {group.items.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </div>
        ))}
      </div>
      {note !== undefined && <p className="dash-note spec-note">{note}</p>}
    </>
  );
}

export function PendingColumns({
  subject,
  pending,
  needs,
}: {
  readonly subject: string;
  readonly pending: readonly string[];
  readonly needs: readonly string[];
}) {
  return (
    <div className="gap">
      <p className="gap-title">{subject} metrics are not available</p>
      <p className="gap-body">
        The control API serves the identity and configuration columns above. The usage columns below
        have no endpoint, so they are omitted from the table rather than rendered as zero.
      </p>
      <p className="gap-needs">
        Needs{" "}
        <span className="gap-endpoints">
          {needs.map((endpoint) => (
            <code key={endpoint} className="mono">
              {endpoint}
            </code>
          ))}
        </span>
      </p>
      <p className="dash-note spec-note">Pending columns: {pending.join(" · ")}</p>
    </div>
  );
}

export function DetailTile({ label, value, mono }: {
  readonly label: string;
  readonly value: string;
  readonly mono?: boolean;
}) {
  return (
    <div className="dash-tile">
      <p className="dash-tile-label">{label}</p>
      <p className={mono === true ? "dash-tile-value mono" : "dash-tile-value"}>{value}</p>
    </div>
  );
}
