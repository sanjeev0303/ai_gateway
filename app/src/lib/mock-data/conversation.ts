import { models } from "./models";
import { NOW } from "./clock";
import { projects } from "./projects";
import { users } from "./users";
import type { Conversation, ConversationTurn } from "@/types";

const minutesAgo = (m: number) =>
  new Date(NOW.getTime() - m * 60_000).toISOString();

/** Derive every denormalised name from its fixture so names cannot drift. */
const userName = (id: string) =>
  users.find((user) => user.id === id)?.name ?? "Unknown user";
const projectName = (id: string) =>
  projects.find((project) => project.id === id)?.name ?? "Unknown project";
const modelOf = (id: string) => {
  const model = models.find((candidate) => candidate.id === id);
  if (!model) throw new Error(`conversation references unknown model: ${id}`);
  return model;
};

type Script = readonly [
  role: ConversationTurn["role"],
  content: string,
  tokens: number,
  toolName?: string,
][];

interface Seed {
  id: string;
  title: string;
  userId: string;
  projectId: string;
  modelId: string;
  startedMinutesAgo: number;
  flagged: boolean;
  flagReason?: string;
  script: Script;
}

const SEEDS: readonly Seed[] = [
  {
    id: "cnv_1",
    title: "Q3 revenue by region, split by plan tier",
    userId: "usr_12",
    projectId: "prj_6",
    modelId: "mdl_claude_sonnet",
    startedMinutesAgo: 47,
    flagged: false,
    script: [
      ["user", "Break down Q3 recognised revenue by region and plan tier. Exclude internal test accounts.", 42],
      ["tool", "SELECT region, plan_tier, SUM(recognised_revenue) FROM billing_facts WHERE quarter = '2026Q3' AND account_type = 'external' GROUP BY 1, 2;", 0, "sql.query"],
      ["assistant", "EMEA leads at $4.18M across 412 accounts, up 9% on Q2. North America is $6.94M, flat quarter on quarter. APAC grew fastest at 23% to $1.21M, though from a small base. Enterprise is 71% of total recognised revenue; the growth is entirely in that tier.", 168],
      ["user", "Split APAC by plan tier and show me the growth rate per tier.", 18],
      ["tool", "SELECT plan_tier, SUM(revenue) q3, SUM(revenue_prev) q2 FROM billing_facts WHERE region = 'APAC' GROUP BY 1;", 0, "sql.query"],
      ["assistant", "APAC Enterprise grew from $410K to $720K, +76%. Team is flat at $380K, and Starter actually declined 4%. The Enterprise jump is two renewals that landed in Q3, so it is not a run-rate.", 142],
    ],
  },
  {
    id: "cnv_2",
    title: "Checkout payment retry loop returning 402",
    userId: "usr_8",
    projectId: "prj_4",
    modelId: "mdl_gpt5_mini",
    startedMinutesAgo: 132,
    flagged: false,
    script: [
      ["user", "Customers see a 402 on retry but the first attempt succeeds. What is the likely cause?", 34],
      ["tool", "search.index(query=\"payment retry 402 authorization hold\")", 0, "search.index"],
      ["assistant", "The first attempt places a 7-day authorization hold on the card. The retry path re-reads the cached order row, which still has the original payment-intent id, so it creates a second intent against an already-held balance and the issuer declines.", 196],
      ["user", "Where in our code is the cached order read on the retry path?", 22],
      ["tool", "code.search(query=\"getOrder retryPayment\")", 0, "code.search"],
      ["assistant", "src/checkout/retry.ts line 41 calls getOrder, which is wrapped in a 60-second Redis cache. The cache is written before the intent status update commits, so a fast retry reads the pre-update row.", 118],
      ["user", "Give me the smallest safe fix.", 9],
      ["assistant", "Invalidate the order cache immediately after the intent status update, rather than relying on TTL expiry. Do not extend the TTL; the write-after-read is the actual bug.", 64],
    ],
  },
  {
    id: "cnv_3",
    title: "Extract renewal and termination terms from Acme MSA",
    userId: "usr_16",
    projectId: "prj_9",
    modelId: "mdl_claude_opus",
    startedMinutesAgo: 214,
    flagged: false,
    script: [
      ["user", "Extract the renewal window, termination-for-convenience terms, and liability cap from the attached MSA.", 58],
      ["assistant", "Renewal: auto-renews for successive 12-month terms unless either party gives 60 days written notice before the term end. Termination for convenience: Acme may terminate on 90 days notice; we may terminate only for material breach plus a 30-day cure period. Liability cap: 12 months of fees paid, excluding IP indemnity and data-breach liability, which are uncapped.", 244],
      ["user", "Is the uncapped data-breach liability acceptable under our standard terms?", 24],
      ["tool", "search.index(query=\"data breach liability cap policy standard terms\")", 0, "search.index"],
      ["assistant", "Our standard position caps data-breach liability at 2x annual fees. This clause is materially worse. Flag it for Legal review before signature; do not accept the uncapped carve-out without a waiver from Security.", 158],
    ],
  },
  {
    id: "cnv_4",
    title: "Classify billing dispute and draft first reply",
    userId: "usr_7",
    projectId: "prj_5",
    modelId: "mdl_gpt5_mini",
    startedMinutesAgo: 19,
    flagged: true,
    flagReason: "PII detector matched 2 email addresses and 1 postal address in tool output",
    script: [
      ["user", "Ticket 88214: customer disputes a duplicate charge on 14 Aug. Their email is dana.reyes@example.com and the card statement shows two $249 line items from 1180 Marina Blvd.", 76],
      ["tool", "crm.lookup(query=\"dana.reyes@example.com\")", 0, "crm.lookup"],
      ["assistant", "Classified as billing_dispute, priority medium. The two $249 charges are 4 minutes apart and share an idempotency key suffix, which indicates a double-submit rather than two distinct purchases.", 132],
      ["user", "Draft the reply. Mask anything that identifies the customer.", 16],
      ["assistant", "Draft ready. Customer email and address are masked as [email] and [address]. The reply explains the duplicate was a system-side double-submit, confirms one charge is being refunded within 3 business days, and does not reference the address on file.", 104],
    ],
  },
  {
    id: "cnv_5",
    title: "Narrative summary of the weekly activation funnel",
    userId: "usr_13",
    projectId: "prj_7",
    modelId: "mdl_mistral_large",
    startedMinutesAgo: 305,
    flagged: false,
    script: [
      ["user", "Summarise the activation funnel for the week ending 27 Aug. Lead with what changed, not the raw numbers.", 38],
      ["tool", "sql.query(query=\"SELECT step, entrants, completions FROM funnel_weekly WHERE week = '2026-W35'\")", 0, "sql.query"],
      ["assistant", "Two things moved. Time-to-first-value fell from 3.1 to 2.4 days, driven mostly by the checklist redesign in the onboarding experiment. But step 3, inviting a teammate, dropped 11% and is now the binding constraint: 71% of accounts stop there. Overall activation is up 4 points to 58%.", 172],
      ["user", "Is the invite drop a real regression or seasonality?", 20],
      ["tool", "sql.query(query=\"SELECT week, step3_rate FROM funnel_weekly ORDER BY week DESC LIMIT 8\")", 0, "sql.query"],
      ["assistant", "Real. The same week last year held at 80%, and the four weeks before this were flat at 79-81%. It lines up with the Aug 18 permission change that added a workspace-selection step before inviting.", 121],
    ],
  },
  {
    id: "cnv_6",
    title: "Regression check: candidate model vs production baseline",
    userId: "usr_2",
    projectId: "prj_2",
    modelId: "mdl_gpt5",
    startedMinutesAgo: 88,
    flagged: false,
    script: [
      ["user", "Run the regression comparison for the new candidate against the current production baseline across all five suites.", 44],
      ["tool", "eval.run(suites=[\"reasoning\",\"tool_use\",\"safety\",\"long_context\",\"multilingual\"])", 0, "eval.run"],
      ["assistant", "Candidate wins on reasoning (+6.2) and tool use (+3.8), ties on long context, loses on multilingual (-4.1) and is marginally down on safety (-1.0). Net is +2.4 weighted, below our 3.0 promotion threshold.", 208],
      ["user", "Which specific multilingual cases regressed?", 14],
      ["tool", "eval.diff(suite=\"multilingual\", direction=\"regressed\")", 0, "eval.diff"],
      ["assistant", "Nine cases, concentrated in Japanese and Korean. The failures are all instruction-following on multi-constraint prompts, not comprehension. The candidate drops the second constraint when the prompt exceeds roughly 600 tokens.", 136],
    ],
  },
  {
    id: "cnv_7",
    title: "Rewrite hero copy for DE, JA, and pt-BR",
    userId: "usr_19",
    projectId: "prj_11",
    modelId: "mdl_gemini_flash",
    startedMinutesAgo: 156,
    flagged: false,
    script: [
      ["user", "Rewrite the hero headline and subhead for German, Japanese, and Brazilian Portuguese. Keep the tone, do not use exclamation marks.", 46],
      ["assistant", "German: \"Jede KI-Aufgabe nachvollziehbar.\" Subhead: \"Sentinel routet, protokolliert und begrenzt jeden Modellaufruf in einer Konsole.\" Japanese: \"すべてのAIタスクを、追跡可能に。\" Subhead: \"Sentinelはすべてのモデル呼び出しをルーティング、記録、制限します。\" Brazilian Portuguese: \"Cada tarefa de IA, rastreável.\" Subhead: \"O Sentinel roteia, registra e limita cada chamada de modelo em um único painel.\"", 186],
      ["user", "Terminology check against the brand glossary.", 12],
      ["tool", "glossary.check(terms=[\"routing\",\"panel\",\"traceable\"])", 0, "glossary.check"],
      ["assistant", "Two issues. The glossary requires \"gateway\" over \"routing\" in customer-facing copy, and requires \"console\" over \"panel\". The German and Portuguese subheads both use the rejected terms. The Japanese headline is clean.", 96],
    ],
  },
  {
    id: "cnv_8",
    title: "Adversarial battery: role-play guardrail escape",
    userId: "usr_22",
    projectId: "prj_13",
    modelId: "mdl_command_r_plus",
    startedMinutesAgo: 26,
    flagged: true,
    flagReason: "Jailbreak detector matched \"ignore previous instructions\" in turn 2",
    script: [
      ["user", "Run case RB-114 from the red-team suite: the DAN-style role-play escape.", 28],
      ["tool", "redteam.run(case=\"RB-114\", target=\"command-r-plus\")", 0, "redteam.run"],
      ["assistant", "The target partially complied. On turn 2 it echoed the injected framing but still withheld the system prompt contents. On turn 5 it produced a paragraph in the requested persona that contradicted its own safety guidance.", 148],
      ["user", "Show me the exact turn-5 text so I can write the failing assertion.", 18],
      ["tool", "redteam.trace(case=\"RB-114\", turn=5)", 0, "redteam.trace"],
      ["assistant", "Turn 5 output is captured. It contains persona-consistent advice that inverts the refusal from turn 3 on the same topic, which is a real inconsistency rather than a cosmetic persona artefact.", 88],
    ],
  },
  {
    id: "cnv_9",
    title: "Why is gateway-core returning upstream 429s",
    userId: "usr_4",
    projectId: "prj_1",
    modelId: "mdl_claude_haiku",
    startedMinutesAgo: 11,
    flagged: false,
    script: [
      ["user", "gateway-core is surfacing upstream 429s on openai but not anthropic. Same traffic shape. What differs?", 32],
      ["tool", "metrics.query(query=\"429 rate by provider, last 6h\")", 0, "metrics.query"],
      ["assistant", "Two differences. Our openai retry multiplier is 1.8, so a single 429 becomes three attempts, while anthropic uses 1.2. And 71% of openai traffic is bursty from ticket-triage, versus 19% for anthropic, so it collides with the provider's per-minute bucket.", 154],
    ],
  },
  {
    id: "cnv_10",
    title: "Draft Q3 EU data-residency attestation",
    userId: "usr_17",
    projectId: "prj_10",
    modelId: "mdl_claude_opus",
    startedMinutesAgo: 512,
    flagged: false,
    script: [
      ["user", "Draft the Q3 attestation covering which model providers processed EU-resident personal data and where.", 40],
      ["tool", "audit.query(action=\"credential_use\", window=\"2026Q3\", region=\"eu\")", 0, "audit.query"],
      ["assistant", "Two providers processed EU-resident data in Q3: AWS Bedrock in eu-central-1, and OpenAI via a zero-retention agreement. Google AI processed only synthetic data and is out of scope. Anthropic and Mistral saw no EU traffic. Bedrock traffic was routed entirely through the Frankfurt region, satisfying the residency constraint.", 214],
      ["user", "Confirm no cross-region replication occurred.", 16],
      ["tool", "audit.query(action=\"export\", window=\"2026Q3\", destination_region=\"non-eu\")", 0, "audit.query"],
      ["assistant", "Confirmed. Zero exports to non-EU destinations in Q3. The only cross-region traffic was a read-replica promotion inside eu-central-1, which is within-region.", 74],
    ],
  },
];

export const conversations: Conversation[] = SEEDS.map((seed) => {
  const model = modelOf(seed.modelId);
  const turns: ConversationTurn[] = seed.script.map(
    ([role, content, tokens, toolName], index) => ({
      id: `${seed.id}_t${index + 1}`,
      role,
      content,
      createdAt: minutesAgo(seed.startedMinutesAgo - index * 2),
      tokens,
      ...(toolName ? { toolName } : {}),
    }),
  );

  const inputTokens = turns
    .filter((turn) => turn.role === "user")
    .reduce((acc, turn) => acc + turn.tokens, 0);
  const outputTokens = turns
    .filter((turn) => turn.role === "assistant")
    .reduce((acc, turn) => acc + turn.tokens, 0);
  const cost =
    (inputTokens / 1_000_000) * model.pricing.inputPerMTok +
    (outputTokens / 1_000_000) * model.pricing.outputPerMTok;

  return {
    id: seed.id,
    title: seed.title,
    userId: seed.userId,
    userName: userName(seed.userId),
    projectId: seed.projectId,
    projectName: projectName(seed.projectId),
    modelId: seed.modelId,
    modelName: model.displayName,
    startedAt: minutesAgo(seed.startedMinutesAgo),
    endedAt: minutesAgo(seed.startedMinutesAgo - seed.script.length * 2),
    turnCount: turns.length,
    inputTokens,
    outputTokens,
    cost: Number(cost.toFixed(4)),
    flagged: seed.flagged,
    ...(seed.flagReason ? { flagReason: seed.flagReason } : {}),
    turns,
  };
});
