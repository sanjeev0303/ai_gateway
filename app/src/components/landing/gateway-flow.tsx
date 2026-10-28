"use client";

import { useState } from "react";
import { Container, SectionLabel } from "@/components/landing/section";
import { ScrollReveal } from "@/components/landing/scroll-reveal";
import {
  ShieldCheck,
  Zap,
  Lock,
  ArrowRight,
  Database,
  Terminal,
  Activity,
  CheckCircle2,
  AlertTriangle,
  RotateCcw,
  Sparkles,
  Bot,
  Workflow,
  CreditCard,
  Building2,
  MessageSquare,
  FileCode,
  Layers,
} from "lucide-react";

type FlowMode = "dataflow" | "dispatch" | "failover";

interface SourceNode {
  id: string;
  name: string;
  category: string;
  icon: typeof Terminal;
  reqs: string;
  status: string;
  payload: string;
  curveY: number;
}

interface SinkNode {
  id: string;
  name: string;
  role: string;
  latency: string;
  badge: string;
  badgeColor: string;
  curveY: number;
}

const SOURCES: SourceNode[] = [
  {
    id: "src-1",
    name: "Stripe Billing Copilot",
    category: "Financial AI Agent",
    icon: CreditCard,
    reqs: "140 req/s",
    status: "Active",
    payload: '{"action": "refund_analysis", "amount": 840}',
    curveY: 48,
  },
  {
    id: "src-2",
    name: "Customer Support Bot",
    category: "Zendesk Streaming Chat",
    icon: MessageSquare,
    reqs: "520 req/s",
    status: "Active",
    payload: '{"session": "usr_942", "msg": "return request"}',
    curveY: 132,
  },
  {
    id: "src-3",
    name: "Autonomous Coding Agent",
    category: "LangChain / AutoGen",
    icon: Bot,
    reqs: "290 req/s",
    status: "Active",
    payload: '{"tool": "execute_patch", "branch": "v2-auth"}',
    curveY: 216,
  },
  {
    id: "src-4",
    name: "Enterprise CRM Copilot",
    category: "Salesforce Sync Mesh",
    icon: Building2,
    reqs: "380 req/s",
    status: "Active",
    payload: '{"query": "summarize_opportunity_notes"}',
    curveY: 300,
  },
  {
    id: "src-5",
    name: "Analytics & ETL Pipeline",
    category: "Snowflake / Embeddings",
    icon: Database,
    reqs: "890 req/s",
    status: "Active",
    payload: '{"batch_size": 256, "vectors": "claims_v3"}',
    curveY: 384,
  },
  {
    id: "src-6",
    name: "Internal Developer Mesh",
    category: "Kubernetes Microservices",
    icon: FileCode,
    reqs: "410 req/s",
    status: "Active",
    payload: '{"service": "catalog_proxy", "auth": "mTLS"}',
    curveY: 468,
  },
];

const SINKS: SinkNode[] = [
  {
    id: "sink-1",
    name: "OpenAI GPT-4o / O3",
    role: "Primary Inference Engine",
    latency: "210ms",
    badge: "ROUTED",
    badgeColor: "bg-[#6F4E37] text-[#F5F3F0]",
    curveY: 48,
  },
  {
    id: "sink-2",
    name: "Anthropic Claude 3.5",
    role: "Complex Reasoning Upstream",
    latency: "235ms",
    badge: "READY",
    badgeColor: "bg-lavender text-[#4A3A30] border border-[rgba(111,78,55,0.25)]",
    curveY: 132,
  },
  {
    id: "sink-3",
    name: "Google Gemini 2.0",
    role: "High-Throughput / Low-Cost",
    latency: "98ms",
    badge: "FASTEST",
    badgeColor: "bg-[#6F4E37]/10 text-[#6F4E37] border border-[rgba(111,78,55,0.25)]",
    curveY: 216,
  },
  {
    id: "sink-4",
    name: "Postgres Audit Ledger",
    role: "Cryptographic Tamper-Proof Log",
    latency: "0.4ms",
    badge: "SEALED",
    badgeColor: "bg-[#1C1412] text-[#E5E4E2]",
    curveY: 300,
  },
  {
    id: "sink-5",
    name: "Pinecone / Qdrant",
    role: "Semantic Cache & Vector Store",
    latency: "11ms",
    badge: "88% HIT",
    badgeColor: "bg-[#6F4E37]/15 text-[#4A3A30]",
    curveY: 384,
  },
  {
    id: "sink-6",
    name: "Enterprise SIEM / S3",
    role: "SOC2 Compliance & Kafka Stream",
    latency: "1.2ms",
    badge: "SYNCED",
    badgeColor: "bg-lavender text-[#6F4E37]",
    curveY: 468,
  },
];

export function GatewayFlow() {
  const [activeMode, setActiveMode] = useState<FlowMode>("dataflow");
  const [activeSourceId, setActiveSourceId] = useState<string>("src-1");
  const [activeSinkId, setActiveSinkId] = useState<string>("sink-1");
  const [failoverTriggered, setFailoverTriggered] = useState(false);

  const selectedSource = SOURCES.find((s) => s.id === activeSourceId) || SOURCES[0];
  const selectedSink = SINKS.find((s) => s.id === activeSinkId) || SINKS[0];

  return (
    <section
      id="workflow"
      className="relative overflow-hidden border-b border-[rgba(111,78,55,0.12)] weave grain scroll-mt-14"
      style={{ background: "var(--canvas-warm)" }}
    >
      <Container className="py-24 sm:py-32 lg:py-40">
        {/* Section Header */}
        <ScrollReveal>
          <div className="max-w-3xl">
            <SectionLabel>04 / ARCHITECTURE & DATAFLOW</SectionLabel>
            <h2
              className="mt-4 font-display text-[clamp(2.5rem,5vw,4.25rem)] font-semibold text-[#1C1412] leading-[0.95]"
              style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.04em" }}
            >
              How Sentinel Unifies Dataflow & Enforces Instant Governance
            </h2>
            <p
              className="mt-6 text-base sm:text-lg leading-relaxed text-[#4A3A30] max-w-[62ch]"
              style={{ letterSpacing: "-0.01em" }}
            >
              Every request passes through Sentinel’s sub-millisecond control plane:
              masking PII, evaluating real-time spending quotas, and dynamically routing across
              multi-model providers and compliance sinks without breaking your client applications.
            </p>
          </div>
        </ScrollReveal>

        {/* Mode Selector Tabs */}
        <ScrollReveal delay={0.1} className="mt-12">
          <div className="flex flex-wrap items-center gap-2 border-b border-[rgba(111,78,55,0.14)] pb-4">
            <button
              type="button"
              onClick={() => {
                setActiveMode("dataflow");
                setFailoverTriggered(false);
              }}
              className={`flex items-center gap-2.5 px-4 py-2.5 text-sm font-semibold uppercase tracking-[0.08em] transition-all duration-150 ${
                activeMode === "dataflow"
                  ? "bg-[#6F4E37] text-[#F5F3F0] shadow-sm"
                  : "bg-[rgba(229,228,226,0.6)] text-[#7A6458] hover:text-[#1C1412] hover:bg-[rgba(229,228,226,0.9)]"
              }`}
              style={{ fontFamily: "var(--font-fira-code)" }}
            >
              <Workflow className="size-4" />
              01 / Ingress-to-Egress Dataflow Hub
            </button>

            <button
              type="button"
              onClick={() => {
                setActiveMode("dispatch");
                setFailoverTriggered(false);
              }}
              className={`flex items-center gap-2.5 px-4 py-2.5 text-sm font-semibold uppercase tracking-[0.08em] transition-all duration-150 ${
                activeMode === "dispatch"
                  ? "bg-[#6F4E37] text-[#F5F3F0] shadow-sm"
                  : "bg-[rgba(229,228,226,0.6)] text-[#7A6458] hover:text-[#1C1412] hover:bg-[rgba(229,228,226,0.9)]"
              }`}
              style={{ fontFamily: "var(--font-fira-code)" }}
            >
              <Bot className="size-4" />
              02 / Agentic Tool Dispatch & Policy Gate
            </button>

            <button
              type="button"
              onClick={() => {
                setActiveMode("failover");
                setFailoverTriggered(true);
              }}
              className={`flex items-center gap-2.5 px-4 py-2.5 text-sm font-semibold uppercase tracking-[0.08em] transition-all duration-150 ${
                activeMode === "failover"
                  ? "bg-[#6F4E37] text-[#F5F3F0] shadow-sm"
                  : "bg-[rgba(229,228,226,0.6)] text-[#7A6458] hover:text-[#1C1412] hover:bg-[rgba(229,228,226,0.9)]"
              }`}
              style={{ fontFamily: "var(--font-fira-code)" }}
            >
              <ShieldCheck className="size-4" />
              03 / Zero-Downtime Failover Simulation
            </button>
          </div>
        </ScrollReveal>

        {/* ============================================================== */}
        {/* MODE 1: UNIFIED INGRESS-TO-EGRESS DATAFLOW (Ref Images 1 & 2) */}
        {/* ============================================================== */}
        {activeMode === "dataflow" && (
          <ScrollReveal delay={0.15} className="mt-10">
            <div className="relative rounded-none border border-[rgba(111,78,55,0.18)] bg-[rgba(229,228,226,0.7)] p-6 sm:p-8 lg:p-10 backdrop-blur-md">
              {/* Top Status Bar */}
              <div className="flex flex-wrap items-center justify-between gap-4 border-b border-[rgba(111,78,55,0.12)] pb-5">
                <div className="flex items-center gap-3">
                  <span className="relative flex size-2.5">
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-[#6F4E37] opacity-75" />
                    <span className="relative inline-flex rounded-full size-2.5 bg-[#6F4E37]" />
                  </span>
                  <span
                    className="font-mono text-sm uppercase tracking-[0.12em] text-[#1C1412] font-semibold"
                    style={{ fontFamily: "var(--font-fira-code)" }}
                  >
                    Sentinel Dataflow Engine — Live Stream
                  </span>
                </div>

                <div className="flex items-center gap-3 text-xs font-mono text-[#7A6458]">
                  <span className="border border-[rgba(111,78,55,0.2)] bg-[var(--canvas)] px-3 py-1">
                    THROUGHPUT: <strong className="text-[#1C1412]">2,630 req/s</strong>
                  </span>
                  <span className="border border-[rgba(111,78,55,0.2)] bg-[var(--canvas)] px-3 py-1">
                    LATENCY: <strong className="text-[#1C1412]">0.84ms p99</strong>
                  </span>
                  <span className="border border-[rgba(111,78,55,0.2)] bg-[var(--canvas)] px-3 py-1">
                    PII LEAKS: <strong className="text-[#6F4E37]">0.00%</strong>
                  </span>
                </div>
              </div>

              {/* Main Visual Canvas: 3 Columns with SVG Bezier Curves */}
              <div className="relative mt-8 min-h-[540px]">
                {/* SVG Curves Layer (Desktop / Tablet) */}
                <svg
                  className="pointer-events-none absolute inset-0 size-full hidden lg:block"
                  viewBox="0 0 1000 520"
                  fill="none"
                  xmlns="http://www.w3.org/2000/svg"
                >
                  <defs>
                    <filter id="particleGlow" x="-20%" y="-20%" width="140%" height="140%">
                      <feGaussianBlur stdDeviation="2.5" result="blur" />
                      <feMerge>
                        <feMergeNode in="blur" />
                        <feMergeNode in="SourceGraphic" />
                      </feMerge>
                    </filter>
                  </defs>

                  {/* Concentric Organic Ripple Orbits around Central Hub (Reference Image 1) */}
                  <g className="animate-ripple-pulse">
                    <circle
                      cx="500"
                      cy="260"
                      r="84"
                      stroke="#6F4E37"
                      strokeWidth="1"
                      strokeDasharray="4 6"
                      opacity="0.25"
                    />
                    <circle
                      cx="500"
                      cy="260"
                      r="120"
                      stroke="#6F4E37"
                      strokeWidth="1"
                      strokeDasharray="3 7"
                      opacity="0.18"
                    />
                    <circle
                      cx="500"
                      cy="260"
                      r="156"
                      stroke="#6F4E37"
                      strokeWidth="0.75"
                      strokeDasharray="4 8"
                      opacity="0.12"
                    />
                  </g>

                  {/* Left-to-Center Inbound Splines */}
                  {SOURCES.map((source, index) => {
                    const startX = 260;
                    const startY = source.curveY;
                    const endX = 430;
                    const endY = 260;
                    const pathD = `M ${startX} ${startY} C 350 ${startY}, 370 ${endY}, ${endX} ${endY}`;
                    const isSelected = activeSourceId === source.id;

                    return (
                      <g key={source.id}>
                        {/* Background guide path */}
                        <path
                          d={pathD}
                          stroke={isSelected ? "#6F4E37" : "rgba(111,78,55,0.20)"}
                          strokeWidth={isSelected ? "2" : "1.25"}
                          strokeDasharray="4 4"
                          className="animate-flow-dash"
                          opacity={isSelected ? "0.9" : "0.5"}
                        />

                        {/* Traveling energy bead heading into hub */}
                        <circle r={isSelected ? "4.5" : "3.5"} fill="#6F4E37" filter="url(#particleGlow)">
                          <animateMotion
                            path={pathD}
                            dur={`${2.0 + (index % 3) * 0.4}s`}
                            repeatCount="indefinite"
                            begin={`${index * 0.35}s`}
                          />
                        </circle>
                        <circle r="1.5" fill="#FFFFFF">
                          <animateMotion
                            path={pathD}
                            dur={`${2.0 + (index % 3) * 0.4}s`}
                            repeatCount="indefinite"
                            begin={`${index * 0.35}s`}
                          />
                        </circle>
                      </g>
                    );
                  })}

                  {/* Center-to-Right Outbound Splines */}
                  {SINKS.map((sink, index) => {
                    const startX = 570;
                    const startY = 260;
                    const endX = 740;
                    const endY = sink.curveY;
                    const pathD = `M ${startX} ${startY} C 630 ${startY}, 650 ${endY}, ${endX} ${endY}`;
                    const isSelected = activeSinkId === sink.id;

                    return (
                      <g key={sink.id}>
                        {/* Background guide path */}
                        <path
                          d={pathD}
                          stroke={isSelected ? "#6F4E37" : "rgba(111,78,55,0.20)"}
                          strokeWidth={isSelected ? "2" : "1.25"}
                          strokeDasharray="4 4"
                          className="animate-flow-dash"
                          opacity={isSelected ? "0.9" : "0.5"}
                        />

                        {/* Traveling energy bead exiting hub to sink */}
                        <circle r={isSelected ? "4.5" : "3.5"} fill="#6F4E37" filter="url(#particleGlow)">
                          <animateMotion
                            path={pathD}
                            dur={`${2.2 + (index % 3) * 0.3}s`}
                            repeatCount="indefinite"
                            begin={`${index * 0.4}s`}
                          />
                        </circle>
                        <circle r="1.5" fill="#FFFFFF">
                          <animateMotion
                            path={pathD}
                            dur={`${2.2 + (index % 3) * 0.3}s`}
                            repeatCount="indefinite"
                            begin={`${index * 0.4}s`}
                          />
                        </circle>
                      </g>
                    );
                  })}
                </svg>

                {/* 3-Column Node Layout */}
                <div className="relative z-10 grid grid-cols-1 gap-8 lg:grid-cols-12 lg:items-center">
                  {/* Left Column: 6 Ingress Sources */}
                  <div className="space-y-3 lg:col-span-3">
                    <p
                      className="font-mono text-xs tracking-[0.14em] uppercase text-[#7A6458] mb-2 font-semibold"
                      style={{ fontFamily: "var(--font-fira-code)" }}
                    >
                      Inbound Sources & Clients
                    </p>
                    {SOURCES.map((source) => {
                      const Icon = source.icon;
                      const isSelected = activeSourceId === source.id;

                      return (
                        <div
                          key={source.id}
                          onClick={() => setActiveSourceId(source.id)}
                          onMouseEnter={() => setActiveSourceId(source.id)}
                          className={`cursor-pointer border p-3 transition-all duration-150 ${
                            isSelected
                              ? "border-[#6F4E37] bg-[var(--canvas)] shadow-md translate-x-1"
                              : "border-[rgba(111,78,55,0.14)] bg-[rgba(245,243,240,0.85)] hover:border-[rgba(111,78,55,0.3)] hover:bg-[var(--canvas)]"
                          }`}
                        >
                          <div className="flex items-center justify-between">
                            <div className="flex items-center gap-2.5">
                              <div
                                className={`size-8 flex items-center justify-center shrink-0 ${
                                  isSelected ? "bg-[#6F4E37] text-[#F5F3F0]" : "bg-lavender text-[#6F4E37]"
                                }`}
                              >
                                <Icon className="size-4" />
                              </div>
                              <div>
                                <p
                                  className="text-sm font-semibold text-[#1C1412] leading-tight"
                                  style={{ letterSpacing: "-0.01em" }}
                                >
                                  {source.name}
                                </p>
                                <p className="text-xs text-[#7A6458] mt-0.5">{source.category}</p>
                              </div>
                            </div>
                            <span
                              className="font-mono text-xs text-[#6F4E37] font-semibold"
                              style={{ fontFamily: "var(--font-fira-code)" }}
                            >
                              {source.reqs}
                            </span>
                          </div>
                        </div>
                      );
                    })}
                  </div>

                  {/* Center Column: The Sentinel Gateway Hub */}
                  <div className="flex flex-col items-center justify-center lg:col-span-6 px-2 py-6">
                    <div className="relative w-full max-w-sm border-2 border-[#6F4E37] bg-[var(--canvas)] p-6 shadow-xl animate-hub-beacon">
                      {/* Hub Badge */}
                      <div className="flex items-center justify-between border-b border-[rgba(111,78,55,0.16)] pb-4">
                        <div className="flex items-center gap-2.5">
                          <div className="size-9 bg-[#6F4E37] text-[#F5F3F0] flex items-center justify-center">
                            <ShieldCheck className="size-5" />
                          </div>
                          <div>
                            <span
                              className="font-mono text-xs font-bold uppercase tracking-[0.14em] text-[#1C1412]"
                              style={{ fontFamily: "var(--font-fira-code)" }}
                            >
                              Sentinel Gateway Core
                            </span>
                            <p className="text-xs text-[#7A6458]">In-Memory Governance Mesh</p>
                          </div>
                        </div>
                        <span
                          className="bg-[#6F4E37] text-[#F5F3F0] px-2 py-0.5 text-xs font-mono tracking-wider font-semibold"
                          style={{ fontFamily: "var(--font-fira-code)" }}
                        >
                          SUB-MS
                        </span>
                      </div>

                      {/* 4 Internal Micro-Gates */}
                      <div className="mt-4 space-y-2 font-mono text-xs" style={{ fontFamily: "var(--font-fira-code)" }}>
                        <div className="flex items-center justify-between border border-[rgba(111,78,55,0.12)] bg-[rgba(245,243,240,0.7)] px-3 py-2">
                          <span className="flex items-center gap-2 text-[#4A3A30] font-medium">
                            <Lock className="size-3.5 text-[#6F4E37]" /> 01. Quota & Budget Gate
                          </span>
                          <span className="text-[#6F4E37] text-xs font-semibold">VERIFIED</span>
                        </div>

                        <div className="flex items-center justify-between border border-[rgba(111,78,55,0.12)] bg-[rgba(245,243,240,0.7)] px-3 py-2">
                          <span className="flex items-center gap-2 text-[#4A3A30] font-medium">
                            <ShieldCheck className="size-3.5 text-[#6F4E37]" /> 02. PII / Secret Redaction
                          </span>
                          <span className="text-[#6F4E37] text-xs font-semibold">CLEAN</span>
                        </div>

                        <div className="flex items-center justify-between border border-[rgba(111,78,55,0.12)] bg-[rgba(245,243,240,0.7)] px-3 py-2">
                          <span className="flex items-center gap-2 text-[#4A3A30] font-medium">
                            <Zap className="size-3.5 text-[#6F4E37]" /> 03. Prompt Injection Defense
                          </span>
                          <span className="text-[#6F4E37] text-xs font-semibold">BLOCKED (0)</span>
                        </div>

                        <div className="flex items-center justify-between border border-[rgba(111,78,55,0.12)] bg-[rgba(245,243,240,0.7)] px-3 py-2">
                          <span className="flex items-center gap-2 text-[#4A3A30] font-medium">
                            <Workflow className="size-3.5 text-[#6F4E37]" /> 04. Dynamic Virtual Router
                          </span>
                          <span className="text-[#6F4E37] text-xs font-semibold">OPTIMIZED</span>
                        </div>
                      </div>

                      {/* Live Inspection Footnote */}
                      <div className="mt-4 border-t border-[rgba(111,78,55,0.12)] pt-3 text-xs text-[#7A6458] flex items-center justify-between">
                        <span>Active Stream: <strong className="text-[#1C1412]">{selectedSource.name}</strong></span>
                        <ArrowRight className="size-3.5 text-[#6F4E37]" />
                        <span>Dest: <strong className="text-[#1C1412]">{selectedSink.name.split(" ")[0]}</strong></span>
                      </div>
                    </div>
                  </div>

                  {/* Right Column: 6 Upstream Sinks */}
                  <div className="space-y-3 lg:col-span-3">
                    <p
                      className="font-mono text-xs tracking-[0.14em] uppercase text-[#7A6458] mb-2 font-semibold"
                      style={{ fontFamily: "var(--font-fira-code)" }}
                    >
                      Upstream Models & Sinks
                    </p>
                    {SINKS.map((sink) => {
                      const isSelected = activeSinkId === sink.id;

                      return (
                        <div
                          key={sink.id}
                          onClick={() => setActiveSinkId(sink.id)}
                          onMouseEnter={() => setActiveSinkId(sink.id)}
                          className={`cursor-pointer border p-3 transition-all duration-150 ${
                            isSelected
                              ? "border-[#6F4E37] bg-[var(--canvas)] shadow-md -translate-x-1"
                              : "border-[rgba(111,78,55,0.14)] bg-[rgba(245,243,240,0.85)] hover:border-[rgba(111,78,55,0.3)] hover:bg-[var(--canvas)]"
                          }`}
                        >
                          <div className="flex items-center justify-between">
                            <div>
                              <p
                                className="text-sm font-semibold text-[#1C1412] leading-tight"
                                style={{ letterSpacing: "-0.01em" }}
                              >
                                {sink.name}
                              </p>
                              <p className="text-xs text-[#7A6458] mt-0.5">{sink.role}</p>
                            </div>
                            <div className="text-right">
                              <span
                                className={`inline-block px-2 py-0.5 text-xs font-mono tracking-wider font-semibold ${sink.badgeColor}`}
                                style={{ fontFamily: "var(--font-fira-code)" }}
                              >
                                {sink.badge}
                              </span>
                              <p
                                className="text-xs text-[#7A6458] font-mono mt-0.5"
                                style={{ fontFamily: "var(--font-fira-code)" }}
                              >
                                {sink.latency}
                              </p>
                            </div>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>
              </div>

              {/* Bottom Interactive Inspection Detail Card */}
              <div className="mt-8 border border-[rgba(111,78,55,0.16)] bg-[var(--canvas)] p-4 sm:p-5">
                <div className="grid gap-4 sm:grid-cols-3">
                  <div>
                    <span
                      className="font-mono text-xs uppercase text-[#7A6458] font-semibold"
                      style={{ fontFamily: "var(--font-fira-code)" }}
                    >
                      Selected Inbound Payload
                    </span>
                    <p
                      className="mt-1 font-mono text-sm text-[#1C1412] bg-[rgba(229,228,226,0.6)] p-2.5 border border-[rgba(111,78,55,0.1)] truncate"
                      style={{ fontFamily: "var(--font-fira-code)" }}
                    >
                      {selectedSource.payload}
                    </p>
                  </div>

                  <div>
                    <span
                      className="font-mono text-xs uppercase text-[#7A6458] font-semibold"
                      style={{ fontFamily: "var(--font-fira-code)" }}
                    >
                      Gateway Inspection Action
                    </span>
                    <div className="mt-1 text-sm text-[#4A3A30] space-y-1">
                      <p className="flex items-center gap-1.5">
                        <CheckCircle2 className="size-3.5 text-[#6F4E37]" /> Token budget reserved: $0.002
                      </p>
                      <p className="flex items-center gap-1.5">
                        <CheckCircle2 className="size-3.5 text-[#6F4E37]" /> PII scan: 0 sensitive entities found
                      </p>
                    </div>
                  </div>

                  <div>
                    <span
                      className="font-mono text-xs uppercase text-[#7A6458] font-semibold"
                      style={{ fontFamily: "var(--font-fira-code)" }}
                    >
                      Egress Target & Telemetry
                    </span>
                    <div className="mt-1 text-sm text-[#4A3A30] space-y-1">
                      <p className="flex items-center gap-1.5">
                        <Activity className="size-3.5 text-[#6F4E37]" /> Routing to {selectedSink.name}
                      </p>
                      <p className="flex items-center gap-1.5">
                        <Lock className="size-3.5 text-[#6F4E37]" /> Cryptographic receipt sealed in Postgres
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </ScrollReveal>
        )}

        {/* ============================================================== */}
        {/* MODE 2: AGENT INTENT & TOOL DISPATCH (Ref Image 3)             */}
        {/* ============================================================== */}
        {activeMode === "dispatch" && (
          <ScrollReveal delay={0.15} className="mt-10">
            <div className="relative rounded-none border border-[rgba(111,78,55,0.18)] bg-[rgba(229,228,226,0.7)] p-6 sm:p-8 lg:p-10 backdrop-blur-md">
              {/* Reference Image 3: Top Agent Prompt Card */}
              <div className="mx-auto max-w-2xl border border-[rgba(111,78,55,0.22)] bg-[var(--canvas)] p-6 shadow-md">
                <div className="flex items-center justify-between border-b border-[rgba(111,78,55,0.12)] pb-3">
                  <div className="flex items-center gap-2">
                    <Sparkles className="size-4 text-[#6F4E37]" />
                    <span
                      className="font-mono text-sm font-semibold uppercase text-[#1C1412]"
                      style={{ fontFamily: "var(--font-fira-code)" }}
                    >
                      Claude 3.5 Sonnet — Enterprise Agent #409
                    </span>
                  </div>
                  <span
                    className="border border-[#6F4E37]/30 bg-lavender px-2.5 py-0.5 text-xs font-mono font-semibold text-[#6F4E37]"
                    style={{ fontFamily: "var(--font-fira-code)" }}
                  >
                    GOVERNED PROMPT
                  </span>
                </div>

                <p
                  className="mt-4 font-mono text-sm leading-relaxed text-[#1C1412]"
                  style={{ fontFamily: "var(--font-fira-code)" }}
                >
                  &quot;Inspect Zendesk ticket #8941, redact customer credit card & SSN data, verify
                  account refund limits in Stripe, update Salesforce record, and notify security on Slack.&quot;
                </p>

                <div className="mt-4 flex items-center justify-between pt-3 border-t border-[rgba(111,78,55,0.08)] text-xs font-mono text-[#7A6458]">
                  <span>Payload: 412 tokens · User: admin@enterprise.corp</span>
                  <span className="text-[#6F4E37] font-semibold">POLICY: FULLY ENFORCED</span>
                </div>
              </div>

              {/* Central Gate Node */}
              <div className="my-8 flex flex-col items-center justify-center">
                <div className="h-8 w-px bg-[#6F4E37]/40" />
                <div className="flex items-center gap-3 border-2 border-[#6F4E37] bg-[var(--canvas)] px-6 py-3.5 shadow-lg">
                  <ShieldCheck className="size-5 text-[#6F4E37]" />
                  <div>
                    <p
                      className="font-mono text-sm font-bold uppercase tracking-wider text-[#1C1412]"
                      style={{ fontFamily: "var(--font-fira-code)" }}
                    >
                      Sentinel Policy & Dispatch Gate
                    </p>
                    <p className="text-xs text-[#7A6458]">0.6ms context screening & privilege verification</p>
                  </div>
                </div>
                <div className="h-8 w-px bg-[#6F4E37]/40" />
              </div>

              {/* Bottom 4 Dispatched Tool Actions */}
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
                <div className="border border-[rgba(111,78,55,0.18)] bg-[var(--canvas)] p-4">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-semibold text-[#1C1412]">Zendesk Ticket</span>
                    <span
                      className="border border-[rgba(111,78,55,0.25)] bg-lavender px-2 py-0.5 text-xs font-mono font-bold text-[#6F4E37]"
                      style={{ fontFamily: "var(--font-fira-code)" }}
                    >
                      READ
                    </span>
                  </div>
                  <p className="mt-2 text-xs leading-relaxed text-[#7A6458]">
                    Fetched conversation body for ticket #8941 without credential disclosure.
                  </p>
                  <p className="mt-3 font-mono text-xs text-[#6F4E37] font-medium">Auth: OAuth2 Scoped</p>
                </div>

                <div className="border border-[rgba(111,78,55,0.18)] bg-[var(--canvas)] p-4">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-semibold text-[#1C1412]">PII Sanitizer</span>
                    <span
                      className="border border-[#6F4E37] bg-[#6F4E37] px-2 py-0.5 text-xs font-mono font-bold text-[#F5F3F0]"
                      style={{ fontFamily: "var(--font-fira-code)" }}
                    >
                      SANITIZE
                    </span>
                  </div>
                  <p className="mt-2 text-xs leading-relaxed text-[#7A6458]">
                    Masked 1 credit card number and 1 SSN prior to upstream propagation.
                  </p>
                  <p className="mt-3 font-mono text-xs text-[#6F4E37] font-medium">Redactions: 2 entities</p>
                </div>

                <div className="border border-[rgba(111,78,55,0.18)] bg-[var(--canvas)] p-4">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-semibold text-[#1C1412]">Stripe Billing</span>
                    <span
                      className="border border-[rgba(111,78,55,0.25)] bg-[rgba(111,78,55,0.12)] px-2 py-0.5 text-xs font-mono font-bold text-[#4A3A30]"
                      style={{ fontFamily: "var(--font-fira-code)" }}
                    >
                      WRITE
                    </span>
                  </div>
                  <p className="mt-2 text-xs leading-relaxed text-[#7A6458]">
                    Validated $50.00 refund limit against organization budget headroom.
                  </p>
                  <p className="mt-3 font-mono text-xs text-[#6F4E37] font-medium">Budget: $840 Remaining</p>
                </div>

                <div className="border border-[rgba(111,78,55,0.18)] bg-[var(--canvas)] p-4">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-semibold text-[#1C1412]">Slack Compliance</span>
                    <span
                      className="border border-[#1C1412] bg-[#1C1412] px-2 py-0.5 text-xs font-mono font-bold text-[#E5E4E2]"
                      style={{ fontFamily: "var(--font-fira-code)" }}
                    >
                      DISPATCH
                    </span>
                  </div>
                  <p className="mt-2 text-xs leading-relaxed text-[#7A6458]">
                    Sent audit digest with cryptographic hash to #security-alerts channel.
                  </p>
                  <p className="mt-3 font-mono text-xs text-[#6F4E37] font-medium">Receipt: #0x82f4e</p>
                </div>
              </div>
            </div>
          </ScrollReveal>
        )}

        {/* ============================================================== */}
        {/* MODE 3: ZERO-DOWNTIME FAILOVER SIMULATION                      */}
        {/* ============================================================== */}
        {activeMode === "failover" && (
          <ScrollReveal delay={0.15} className="mt-10">
            <div className="relative rounded-none border border-[rgba(111,78,55,0.18)] bg-[rgba(229,228,226,0.7)] p-6 sm:p-8 lg:p-10 backdrop-blur-md">
              <div className="flex flex-wrap items-center justify-between gap-4 border-b border-[rgba(111,78,55,0.12)] pb-5">
                <div>
                  <h3 className="text-lg font-semibold text-[#1C1412]">
                    High-Availability Circuit Breaker & Upstream Outage Recovery
                  </h3>
                  <p className="text-sm text-[#7A6458] mt-1">
                    What happens when an upstream AI provider experiences an outage or 429 rate limit?
                  </p>
                </div>

                <button
                  type="button"
                  onClick={() => setFailoverTriggered(!failoverTriggered)}
                  className="flex items-center gap-2 border border-[#6F4E37] bg-[#6F4E37] px-4 py-2.5 text-sm font-mono font-semibold text-[#F5F3F0] transition-colors hover:bg-[#4A3224]"
                  style={{ fontFamily: "var(--font-fira-code)" }}
                >
                  <RotateCcw className="size-4" />
                  {failoverTriggered ? "Reset Simulation" : "Trigger Upstream Outage"}
                </button>
              </div>

              {/* Failover Diagram */}
              <div className="mt-8 grid grid-cols-1 gap-6 lg:grid-cols-3 lg:items-center">
                {/* Step 1: Client Request */}
                <div className="border border-[rgba(111,78,55,0.16)] bg-[var(--canvas)] p-5">
                  <div className="flex items-center gap-2 text-sm font-semibold text-[#1C1412]">
                    <Terminal className="size-4 text-[#6F4E37]" />
                    <span>Client Application</span>
                  </div>
                  <p className="mt-2 text-sm text-[#7A6458]">
                    Sends standard OpenAI SDK request to Sentinel endpoint.
                  </p>
                  <div
                    className="mt-4 border border-[rgba(111,78,55,0.12)] bg-[rgba(245,243,240,0.8)] p-2.5 font-mono text-xs text-[#4A3A30] font-medium"
                    style={{ fontFamily: "var(--font-fira-code)" }}
                  >
                    POST /v1/chat/completions
                  </div>
                </div>

                {/* Step 2: Sentinel Circuit Breaker */}
                <div className="border-2 border-[#6F4E37] bg-[var(--canvas)] p-5 shadow-md">
                  <div className="flex items-center justify-between">
                    <span
                      className="font-mono text-sm font-bold uppercase text-[#1C1412]"
                      style={{ fontFamily: "var(--font-fira-code)" }}
                    >
                      Sentinel Circuit Breaker
                    </span>
                    <span className="text-xs font-mono text-[#6F4E37] font-semibold">0.8ms</span>
                  </div>

                  <div className="mt-4 space-y-2.5 font-mono text-xs" style={{ fontFamily: "var(--font-fira-code)" }}>
                    <div
                      className={`flex items-center justify-between border p-2.5 ${
                        failoverTriggered
                          ? "border-red-400/50 bg-red-50 text-red-900"
                          : "border-[rgba(111,78,55,0.14)] bg-[var(--canvas)] text-[#4A3A30]"
                      }`}
                    >
                      <span>Route A: OpenAI GPT-4o</span>
                      <span className="font-bold">{failoverTriggered ? "503 OUTAGE" : "ACTIVE"}</span>
                    </div>

                    <div
                      className={`flex items-center justify-between border p-2.5 ${
                        failoverTriggered
                          ? "border-[#6F4E37] bg-[#6F4E37] text-[#F5F3F0]"
                          : "border-[rgba(111,78,55,0.14)] bg-[var(--canvas)] text-[#7A6458]"
                      }`}
                    >
                      <span>Route B: Anthropic Claude</span>
                      <span className="font-bold">{failoverTriggered ? "REROUTED (200 OK)" : "STANDBY"}</span>
                    </div>
                  </div>
                </div>

                {/* Step 3: Result */}
                <div className="border border-[rgba(111,78,55,0.16)] bg-[var(--canvas)] p-5">
                  <div className="flex items-center gap-2 text-sm font-semibold text-[#1C1412]">
                    <CheckCircle2 className="size-4 text-[#6F4E37]" />
                    <span>Client Impact</span>
                  </div>
                  <div className="mt-3">
                    <span
                      className="inline-block bg-[#6F4E37] px-2.5 py-0.5 text-xs font-mono font-semibold text-[#F5F3F0]"
                      style={{ fontFamily: "var(--font-fira-code)" }}
                    >
                      ZERO ERRORS RETURNED
                    </span>
                    <p className="mt-2 text-sm text-[#4A3A30] leading-relaxed">
                      {failoverTriggered
                        ? "Circuit breaker intercepted the OpenAI failure in 1.4ms and transparently mapped the prompt to Claude 3.5. Client never saw a 503."
                        : "Sentinel monitors provider health in real-time, executing health probes every 2 seconds."}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </ScrollReveal>
        )}

        {/* 4 Architectural Telemetry Metrics */}
        <ScrollReveal delay={0.2} className="mt-16">
          <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4 border-t border-[rgba(111,78,55,0.14)] pt-12">
            <div>
              <p
                className="font-mono text-xs uppercase tracking-[0.14em] text-[#7A6458] font-semibold"
                style={{ fontFamily: "var(--font-fira-code)" }}
              >
                Service Continuity
              </p>
              <p
                className="mt-2 font-display text-4xl sm:text-5xl font-semibold text-[#1C1412]"
                style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.04em" }}
              >
                99.999%
              </p>
              <p className="mt-1.5 text-sm text-[#4A3A30]">Automatic zero-downtime multi-provider failover</p>
            </div>

            <div>
              <p
                className="font-mono text-xs uppercase tracking-[0.14em] text-[#7A6458] font-semibold"
                style={{ fontFamily: "var(--font-fira-code)" }}
              >
                Gateway Overhead
              </p>
              <p
                className="mt-2 font-display text-4xl sm:text-5xl font-semibold text-[#1C1412]"
                style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.04em" }}
              >
                &lt; 0.9ms
              </p>
              <p className="mt-1.5 text-sm text-[#4A3A30]">In-memory Rust / Go native evaluation pipeline</p>
            </div>

            <div>
              <p
                className="font-mono text-xs uppercase tracking-[0.14em] text-[#7A6458] font-semibold"
                style={{ fontFamily: "var(--font-fira-code)" }}
              >
                PII Leak Incidents
              </p>
              <p
                className="mt-2 font-display text-4xl sm:text-5xl font-semibold text-[#6F4E37]"
                style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.04em" }}
              >
                0.00%
              </p>
              <p className="mt-1.5 text-sm text-[#4A3A30]">Zero unredacted credentials or personal data to LLMs</p>
            </div>

            <div>
              <p
                className="font-mono text-xs uppercase tracking-[0.14em] text-[#7A6458] font-semibold"
                style={{ fontFamily: "var(--font-fira-code)" }}
              >
                Token Waste Reduction
              </p>
              <p
                className="mt-2 font-display text-4xl sm:text-5xl font-semibold text-[#1C1412]"
                style={{ fontFamily: "var(--font-cormorant)", letterSpacing: "-0.04em" }}
              >
                38.4%
              </p>
              <p className="mt-1.5 text-sm text-[#4A3A30]">Semantic caching and virtual model routing</p>
            </div>
          </div>
        </ScrollReveal>
      </Container>
    </section>
  );
}
