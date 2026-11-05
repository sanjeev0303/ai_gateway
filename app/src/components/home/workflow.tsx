import { FlowCurves } from "@/components/brand/flow-curves";
import { Container, SectionIntro } from "./section";

/*
 * Two diagrams, drawn to the diagram-design skill's grammar in the page's own
 * tokens and faces: orthogonal connectors, masked labels, a 4px grid, and the
 * complete meaning in the markup before any motion runs.
 */

const UI = "var(--font-ui)";
const MONO = "var(--font-code)";
const PAPER = "var(--card)";
const INK = "var(--foreground)";
const MUTED = "var(--muted-foreground)";
const RULE = "var(--border-strong)";
const ACCENT = "var(--primary)";

function Eyebrow({ x, y, width, label }: { x: number; y: number; width: number; label: string }) {
  return (
    <>
      <rect x={x} y={y} width={width} height={12} rx={2} fill={PAPER} />
      <text x={x + width / 2} y={y + 9} fill={MUTED} fontSize={8} fontFamily={MONO} textAnchor="middle" letterSpacing="0.14em">
        {label}
      </text>
    </>
  );
}

function Node({
  x,
  y,
  width,
  height,
  tag,
  name,
  sub,
  focal = false,
  store = false,
}: {
  x: number;
  y: number;
  width: number;
  height: number;
  tag: string;
  name: string;
  sub: string;
  focal?: boolean;
  store?: boolean;
}) {
  const cx = x + width / 2;
  const cy = y + height / 2;
  const stroke = focal ? ACCENT : RULE;
  const fill = focal
    ? "color-mix(in oklab, var(--primary) 12%, var(--card))"
    : store
      ? "color-mix(in oklab, var(--foreground) 4%, var(--card))"
      : PAPER;
  return (
    <g>
      <rect x={x} y={y} width={width} height={height} rx={6} fill={PAPER} />
      <rect x={x} y={y} width={width} height={height} rx={6} style={{ fill }} stroke={stroke} strokeWidth={focal ? 1.4 : 1} />
      <rect x={x + 8} y={y + 6} width={tag.length * 6 + 10} height={12} rx={2} fill="none" stroke={stroke} strokeOpacity={0.45} strokeWidth={0.8} />
      <text x={x + 13} y={y + 15} fill={MUTED} fontSize={7} fontFamily={MONO} letterSpacing="0.08em">
        {tag}
      </text>
      <text x={cx} y={cy + 6} fill={INK} fontSize={13} fontWeight={600} fontFamily={UI} textAnchor="middle">
        {name}
      </text>
      <text x={cx} y={cy + 21} fill={MUTED} fontSize={9.5} fontFamily={MONO} textAnchor="middle">
        {sub}
      </text>
    </g>
  );
}

function ArrowLabel({ cx, y, label }: { cx: number; y: number; label: string }) {
  const width = Math.ceil(label.length * 8 * 0.62) + 10;
  return (
    <>
      <rect x={cx - width / 2} y={y} width={width} height={12} rx={2} fill={PAPER} />
      <text x={cx} y={y + 9} fill={MUTED} fontSize={8} fontFamily={MONO} textAnchor="middle" letterSpacing="0.08em">
        {label}
      </text>
    </>
  );
}

/* -------------------------------------------------------------------------- */
/* Where Tollseal sits: one snapshot, a quiet flow token on the headline path */
/* -------------------------------------------------------------------------- */

function WhereItSits() {
  return (
    <svg
      viewBox="0 0 960 452"
      role="img"
      aria-labelledby="where-it-sits-title where-it-sits-desc"
      className="block h-auto w-full min-w-[960px]"
    >
      <title id="where-it-sits-title">Where Tollseal sits</title>
      <desc id="where-it-sits-desc">
        Your services, scripts and tools call one Tollseal gateway that runs on your own infrastructure; it checks the key,
        policy, guardrails and budget, sends the request to OpenAI, Anthropic or another provider, and writes every request
        to a record of tokens, cost, latency and outcome.
      </desc>
      <defs>
        <marker id="wis-arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
          <path d="M1 1L9 5L1 9" fill="none" stroke={MUTED} strokeWidth={1.4} strokeLinecap="round" strokeLinejoin="round" />
        </marker>
      </defs>

      {/* Built in the order a request travels, one step per group, so the player can
          reveal it: apps, one key, the gateway, routing to providers, the sealed record. */}
      <g data-scroll-step={1} aria-label="Your apps: services with the TypeScript SDK, scripts with the aigw CLI, tools over REST">
        <rect x={24} y={56} width={232} height={296} rx={8} style={{ fill: "color-mix(in oklab, var(--foreground) 2%, transparent)" }} stroke={RULE} strokeOpacity={0.5} strokeWidth={0.8} />
        <Eyebrow x={40} y={60} width={76} label="YOUR APPS" />
        <Node x={48} y={96} width={184} height={56} tag="SDK" name="Services" sub="TypeScript SDK" />
        <Node x={48} y={176} width={184} height={56} tag="CLI" name="Scripts" sub="aigw CLI" />
        <Node x={48} y={256} width={184} height={56} tag="REST" name="Tools" sub="OpenAI / Anthropic style" />
      </g>

      <g data-scroll-step={2} aria-label="Every app calls the gateway with one Tollseal key">
        <g fill="none" stroke={MUTED} strokeWidth={1.2} markerEnd="url(#wis-arrow)" data-connectors>
          <path d="M232 124H300Q308 124 308 132V176Q308 184 316 184H384" pathLength={1} data-connector />
          <path d="M232 204H384" pathLength={1} data-connector />
          <path d="M232 284H300Q308 284 308 276V232Q308 224 316 224H384" pathLength={1} data-connector />
        </g>
        <ArrowLabel cx={266} y={184} label="ONE KEY" />
      </g>

      <g data-scroll-step={3} aria-label="The Tollseal gateway, on your own infrastructure, checks keys, rules and budgets">
        <rect x={352} y={128} width={256} height={244} rx={8} fill="none" stroke={RULE} strokeWidth={0.8} strokeDasharray="5 4" />
        <Eyebrow x={368} y={132} width={132} label="YOUR INFRASTRUCTURE" />
        <Node x={384} y={164} width={192} height={80} tag="GATEWAY" name="Tollseal" sub="keys · rules · budgets" focal />
      </g>

      <g data-scroll-step={4} aria-label="Routed to the provider that serves the model: OpenAI, Anthropic, Gemini, Hugging Face or any OpenAI-compatible endpoint">
        <rect x={704} y={56} width={232} height={296} rx={8} style={{ fill: "color-mix(in oklab, var(--foreground) 2%, transparent)" }} stroke={RULE} strokeOpacity={0.5} strokeWidth={0.8} />
        <Eyebrow x={720} y={60} width={76} label="PROVIDERS" />
        <g fill="none" stroke={MUTED} strokeWidth={1.2} markerEnd="url(#wis-arrow)" data-connectors>
          <path d="M576 184H644Q652 184 652 176V132Q652 124 660 124H728" pathLength={1} data-connector />
          <path d="M576 204H728" pathLength={1} data-connector />
          <path d="M576 224H644Q652 224 652 232V276Q652 284 660 284H728" pathLength={1} data-connector />
        </g>
        <ArrowLabel cx={690} y={184} label="ROUTED" />
        <Node x={728} y={96} width={184} height={56} tag="API" name="OpenAI" sub="and Azure OpenAI" />
        <Node x={728} y={176} width={184} height={56} tag="API" name="Anthropic" sub="messages format" />
        <Node x={728} y={256} width={184} height={56} tag="API" name="Gemini · Hugging Face" sub="and any OpenAI-compatible" />
      </g>

      <g data-scroll-step={5} aria-label="Every request is sealed into the record: tokens, cost and outcome">
        <path d="M480 244V300" fill="none" stroke={MUTED} strokeWidth={1.2} markerEnd="url(#wis-arrow)" pathLength={1} data-connector />
        <ArrowLabel cx={516} y={266} label="SEALED" />
        <Node x={384} y={300} width={192} height={56} tag="POSTGRES" name="Request record" sub="tokens · cost · outcome" store />
      </g>

      {/* Legend: a horizontal strip at the bottom */}
      <g fontFamily={MONO} fontSize={8.5} fill={MUTED}>
        <rect x={24} y={396} width={912} height={40} rx={6} fill="none" stroke={RULE} strokeOpacity={0.5} strokeWidth={0.8} />
        <rect x={44} y={410} width={18} height={12} rx={3} style={{ fill: "color-mix(in oklab, var(--primary) 12%, var(--card))" }} stroke={ACCENT} strokeWidth={1.2} />
        <text x={70} y={419}>THE GATEWAY</text>
        <rect x={194} y={410} width={18} height={12} rx={3} fill={PAPER} stroke={RULE} />
        <text x={220} y={419}>A CALLER OR A PROVIDER</text>
        <rect x={392} y={410} width={18} height={12} rx={3} style={{ fill: "color-mix(in oklab, var(--foreground) 4%, var(--card))" }} stroke={RULE} />
        <text x={418} y={419}>A STORE</text>
        <rect x={510} y={410} width={18} height={12} rx={3} fill="none" stroke={RULE} strokeDasharray="4 3" />
        <text x={536} y={419}>RUNS ON YOUR MACHINES</text>
        <path d="M712 416H744" stroke={MUTED} strokeWidth={1.2} markerEnd="url(#wis-arrow)" />
        <text x={754} y={419}>A REQUEST</text>
      </g>
    </svg>
  );
}

/* -------------------------------------------------------------------------- */
/* Two requests, the same checks: paired policy-evaluation traces             */
/* -------------------------------------------------------------------------- */

type Status = "PASS" | "FAIL" | "NOT REACHED";

const RULES: { name: string; sub: string; a: [Status, string]; b: [Status, string] }[] = [
  { name: "Key", sub: "gateway key", a: ["PASS", "verified · team Payments"], b: ["PASS", "verified · team Payments"] },
  { name: "Policy", sub: "model access", a: ["PASS", "gpt-4o-mini allowed"], b: ["PASS", "gpt-4o-mini allowed"] },
  { name: "Guardrails", sub: "secrets: block", a: ["PASS", "nothing found"], b: ["FAIL", "API key found · blocked"] },
  { name: "Budget", sub: "monthly spend cap", a: ["PASS", "$41.20 of $500"], b: ["NOT REACHED", "stopped at guardrails"] },
  { name: "Route", sub: "provider", a: ["PASS", "sent to OpenAI"], b: ["NOT REACHED", "no provider saw it"] },
  { name: "Record", sub: "request log", a: ["PASS", "1,284 tokens · 812 ms"], b: ["PASS", "recorded as blocked"] },
];

const ROW_Y = (i: number) => 104 + i * 72;

/** The drawn mark for a status: a check, a cross, or a dash. Box is 11 x 10. */
function markPath(status: Status, x: number, y: number): string {
  if (status === "PASS") return `M${x} ${y + 5.5}L${x + 3.8} ${y + 9.2}L${x + 11} ${y + 1}`;
  if (status === "FAIL") return `M${x + 1} ${y + 1}L${x + 10} ${y + 10}M${x + 10} ${y + 1}L${x + 1} ${y + 10}`;
  return `M${x} ${y + 5.5}H${x + 10}`;
}

function Mark({ status, x, y, tone }: { status: Status; x: number; y: number; tone: string }) {
  return <path d={markPath(status, x, y)} fill="none" stroke={tone} strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round" />;
}

function StatusCell({ x, y, status, detail }: { x: number; y: number; status: Status; detail: string }) {
  const tone = status === "PASS" ? "var(--success)" : status === "FAIL" ? "var(--danger)" : MUTED;
  const reached = status !== "NOT REACHED";
  return (
    <g>
      <rect
        x={x}
        y={y}
        width={312}
        height={56}
        rx={6}
        style={{ fill: reached ? `color-mix(in oklab, ${tone} 9%, var(--card))` : PAPER }}
        stroke={tone}
        strokeOpacity={reached ? 0.55 : 0.6}
        strokeWidth={1}
        strokeDasharray={reached ? undefined : "4 3"}
      />
      <Mark status={status} x={x + 16} y={y + 14} tone={tone} />
      <text x={x + 34} y={y + 23} fill={tone} fontSize={9} fontFamily={MONO} letterSpacing="0.12em">
        {status}
      </text>
      <text x={x + 34} y={y + 41} fill={reached ? INK : MUTED} fontSize={12} fontFamily={UI}>
        {detail}
      </text>
    </g>
  );
}

function PolicyTraces() {
  return (
    <svg
      viewBox="0 0 960 664"
      role="img"
      aria-labelledby="policy-traces-title policy-traces-desc"
      className="block h-auto w-full min-w-[960px]"
    >
      <title id="policy-traces-title">Two requests, the same six checks</title>
      <desc id="policy-traces-desc">
        Request A and request B are the same prompt, but B carries a pasted API key. Both pass the key and policy checks. At
        guardrails they first diverge: A passes, B is blocked, so its budget and route checks are not reached. A is answered
        by OpenAI with status 200; B ends with 403 before any provider sees it. Both are recorded.
      </desc>

      {/* Column heads */}
      <text x={24} y={52} fill={MUTED} fontSize={9} fontFamily={MONO} letterSpacing="0.14em">
        CHECK, IN ORDER
      </text>
      {[
        { x: 288, name: "Request A", sub: "“Summarise ticket #4821”" },
        { x: 624, name: "Request B", sub: "same prompt + a pasted API key" },
      ].map((head) => (
        <g key={head.name}>
          <rect x={head.x} y={24} width={312} height={56} rx={6} fill={PAPER} stroke={RULE} />
          <text x={head.x + 16} y={48} fill={INK} fontSize={13} fontWeight={600} fontFamily={UI}>
            {head.name}
          </text>
          <text x={head.x + 16} y={66} fill={MUTED} fontSize={10} fontFamily={MONO}>
            {head.sub}
          </text>
        </g>
      ))}

      {RULES.map((rule, i) => {
        const y = ROW_Y(i);
        const [aStatus, aDetail] = rule.a;
        const [bStatus, bDetail] = rule.b;
        return (
          <g
            key={rule.name}
            data-scroll-step={i + 1}
            aria-label={`${rule.name}: request A ${aStatus.toLowerCase()}, ${aDetail}; request B ${bStatus.toLowerCase()}, ${bDetail}`}
          >
            <rect x={24} y={y} width={240} height={56} rx={6} fill={PAPER} stroke={RULE} />
            <text x={40} y={y + 25} fill={INK} fontSize={13} fontWeight={600} fontFamily={UI}>
              {rule.name}
            </text>
            <text x={40} y={y + 42} fill={MUTED} fontSize={10} fontFamily={MONO}>
              {rule.sub}
            </text>
            <StatusCell x={288} y={y} status={aStatus} detail={aDetail} />
            <StatusCell x={624} y={y} status={bStatus} detail={bDetail} />
          </g>
        );
      })}

      {/* The first divergence, at guardrails */}
      <g data-scroll-step={3} aria-label="First divergence at guardrails">
        <rect x={12} y={240} width={936} height={72} rx={8} fill="none" stroke={ACCENT} strokeWidth={1.4} />
        <rect x={32} y={234} width={96} height={12} rx={2} fill={PAPER} />
        <text x={80} y={243} fill={ACCENT} fontSize={8} fontFamily={MONO} textAnchor="middle" letterSpacing="0.1em">
          FIRST DIVERGENCE
        </text>
      </g>

      {/* Outcomes */}
      <g data-scroll-step={7} aria-label="Outcomes: A answered with 200 and sealed; B blocked with 403 before any provider saw it">
        <text x={24} y={574} fill={MUTED} fontSize={9} fontFamily={MONO} letterSpacing="0.14em">
          OUTCOME
        </text>
        <rect x={288} y={544} width={312} height={48} rx={6} fill="none" stroke="var(--success)" strokeWidth={1.4} />
        <text x={304} y={573} fill={INK} fontSize={12.5} fontWeight={600} fontFamily={UI}>
          200 · answered by OpenAI, sealed
        </text>
        <rect x={624} y={544} width={312} height={48} rx={6} fill="none" stroke="var(--danger)" strokeWidth={1.4} />
        <text x={640} y={573} fill={INK} fontSize={12.5} fontWeight={600} fontFamily={UI}>
          403 · blocked before any provider
        </text>
      </g>

      {/* Legend: every status the pattern distinguishes */}
      <g fontFamily={MONO} fontSize={8.5} fill={MUTED}>
        <rect x={24} y={612} width={912} height={36} rx={6} fill="none" stroke={RULE} strokeOpacity={0.5} strokeWidth={0.8} />
        <Mark status="PASS" x={44} y={625} tone="var(--success)" />
        <text x={62} y={634} fill="var(--success)">PASS</text>
        <text x={124} y={634}>the check ran and let it through</text>
        <Mark status="FAIL" x={344} y={625} tone="var(--danger)" />
        <text x={362} y={634} fill="var(--danger)">FAIL</text>
        <text x={420} y={634}>the check stopped it</text>
        <Mark status="NOT REACHED" x={562} y={625} tone={MUTED} />
        <text x={580} y={634}>NOT REACHED</text>
        <text x={672} y={634}>never evaluated</text>
        <text x={776} y={634}>SKIPPED: none here</text>
      </g>
    </svg>
  );
}

/* -------------------------------------------------------------------------- */
/* Phone layouts: the same facts, stacked, so nothing sits off-screen          */
/* -------------------------------------------------------------------------- */

function StackStep({ tag, title, sub, focal = false }: { tag: string; title: string; sub: string; focal?: boolean }) {
  return (
    <div
      className={
        focal
          ? "rounded-lg border-[1.4px] border-primary bg-primary/10 px-4 py-3"
          : "rounded-lg border border-border-strong bg-card px-4 py-3"
      }
    >
      <span className="font-mono text-[0.625rem] tracking-[0.12em] text-muted-foreground">{tag}</span>
      <p className="text-sm font-semibold">{title}</p>
      <p className="font-mono text-xs text-muted-foreground">{sub}</p>
    </div>
  );
}

function StackArrow({ label }: { label: string }) {
  return (
    <div className="flex items-center gap-3 py-1.5 pl-6" aria-hidden>
      <span className="h-6 w-px bg-muted-foreground/60" />
      <span className="font-mono text-[0.625rem] tracking-[0.12em] text-muted-foreground">{label}</span>
    </div>
  );
}

function WhereItSitsStacked() {
  return (
    <div role="img" aria-label="Where Tollseal sits: your services, scripts and tools call one Tollseal gateway on your own infrastructure with one key; it checks keys, rules and budgets, routes to OpenAI, Anthropic, Gemini, Hugging Face or any OpenAI-compatible endpoint, and seals every request into a record of tokens, cost and outcome.">
      <div data-scroll-step={1}>
        <StackStep tag="YOUR APPS" title="Services · Scripts · Tools" sub="TypeScript SDK · aigw CLI · REST" />
      </div>
      <div data-scroll-step={2}>
        <StackArrow label="ONE KEY" />
        <StackStep tag="YOUR INFRASTRUCTURE" title="Tollseal gateway" sub="keys · rules · budgets" focal />
      </div>
      <div data-scroll-step={3}>
        <StackArrow label="ROUTED" />
        <StackStep tag="PROVIDERS" title="OpenAI · Anthropic · Gemini" sub="Azure, Hugging Face, any OpenAI-compatible" />
      </div>
      <div data-scroll-step={4}>
        <StackArrow label="SEALED" />
        <StackStep tag="POSTGRES" title="Request record" sub="tokens · cost · outcome" />
      </div>
    </div>
  );
}

function MarkIcon({ status }: { status: Status }) {
  const tone = status === "PASS" ? "var(--success)" : status === "FAIL" ? "var(--danger)" : MUTED;
  return (
    <svg viewBox="0 0 13 12" className="size-3.5 shrink-0" aria-hidden focusable="false">
      <Mark status={status} x={1} y={1} tone={tone} />
    </svg>
  );
}

function TraceCell({ who, status, detail }: { who: string; status: Status; detail: string }) {
  const reached = status !== "NOT REACHED";
  return (
    <div className={reached ? "rounded-md border border-border-strong bg-card px-2.5 py-2" : "rounded-md border border-dashed border-border-strong px-2.5 py-2"}>
      <span className="flex items-center gap-1.5 font-mono text-[0.625rem] tracking-[0.1em] text-muted-foreground">
        {who} <MarkIcon status={status} /> {status}
      </span>
      <p className={reached ? "text-xs text-foreground" : "text-xs text-muted-foreground"}>{detail}</p>
    </div>
  );
}

function PolicyTracesStacked() {
  return (
    <ol aria-label="Two requests, the same six checks" className="flex flex-col gap-2.5">
      {RULES.map((rule, i) => {
        const diverges = rule.name === "Guardrails";
        return (
          <li key={rule.name} data-scroll-step={i + 1} className={diverges ? "rounded-lg border-[1.4px] border-primary p-2.5" : "p-0.5"}>
            {diverges && <p className="mb-1.5 font-mono text-[0.625rem] tracking-[0.12em] text-primary">FIRST DIVERGENCE</p>}
            <p className="mb-1.5 text-sm font-semibold">
              {rule.name} <span className="font-mono text-xs font-normal text-muted-foreground">{rule.sub}</span>
            </p>
            <div className="grid grid-cols-2 gap-2">
              <TraceCell who="A" status={rule.a[0]} detail={rule.a[1]} />
              <TraceCell who="B" status={rule.b[0]} detail={rule.b[1]} />
            </div>
          </li>
        );
      })}
      <li data-scroll-step={7} className="grid grid-cols-2 gap-2 pt-1">
        <p className="rounded-md border-[1.4px] border-(--success) px-2.5 py-2 text-xs font-semibold">A · 200 · answered by OpenAI, sealed</p>
        <p className="rounded-md border-[1.4px] border-(--danger) px-2.5 py-2 text-xs font-semibold">B · 403 · blocked before any provider</p>
      </li>
    </ol>
  );
}

export function Workflow() {
  return (
    <section id="workflow" aria-labelledby="workflow-heading" className="relative scroll-mt-20 overflow-hidden py-24 lg:py-32">
      <FlowCurves className="opacity-80" />
      <Container className="relative flex flex-col gap-14">
        <SectionIntro
          id="workflow-heading"
          title="How a request travels."
          body="Your apps change one base URL. Behind it, every request takes the same path through the gateway, and leaves a record whether it is answered or stopped."
        />

        {/* Each diagram builds step by step as it scrolls through the screen, and unbuilds on the way back. */}
        <div className="flex flex-col gap-4">
          <h3 className="text-lg font-semibold tracking-[-0.01em]">Where Tollseal sits</h3>
          <figure data-scroll-diagram className="hidden rounded-2xl border border-border bg-card p-6 md:block">
            <div className="overflow-x-auto" data-lenis-prevent>
              <WhereItSits />
            </div>
          </figure>
          <figure data-scroll-diagram className="rounded-2xl border border-border bg-card p-4 md:hidden">
            <WhereItSitsStacked />
          </figure>
        </div>

        <div className="flex flex-col gap-4">
          <h3 className="text-lg font-semibold tracking-[-0.01em]">Two requests, the same checks</h3>
          <p className="max-w-[60ch] text-[0.9375rem] leading-relaxed text-muted-foreground">
            The same prompt, sent twice. The second carries a pasted API key. Watch where they part.
          </p>
          <figure data-scroll-diagram className="hidden rounded-2xl border border-border bg-card p-6 md:block">
            <div className="overflow-x-auto" data-lenis-prevent>
              <PolicyTraces />
            </div>
          </figure>
          <figure data-scroll-diagram className="rounded-2xl border border-border bg-card p-4 md:hidden">
            <PolicyTracesStacked />
          </figure>
        </div>
      </Container>
    </section>
  );
}
