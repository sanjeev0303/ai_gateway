import { NOW } from "./clock";
import type { Guardrail } from "@/types";

const daysAgo = (n: number) =>
  new Date(NOW.getTime() - n * 86_400_000).toISOString();

export const guardrails: Guardrail[] = [
  {
    id: "grd_1",
    name: "Email and phone redaction",
    description:
      "Masks email addresses and phone numbers in both prompts and responses.",
    layer: "input",
    action: "mask",
    enabled: true,
    threshold: 0.72,
    triggers24h: 412_806,
    interventions24h: 8_140,
    updatedAt: daysAgo(9),
  },
  {
    id: "grd_2",
    name: "Payment card detection",
    description:
      "Blocks requests containing full payment card numbers, regardless of stated intent.",
    layer: "input",
    action: "block",
    enabled: true,
    threshold: 0.6,
    triggers24h: 412_806,
    interventions24h: 34,
    updatedAt: daysAgo(9),
  },
  {
    id: "grd_3",
    name: "Instruction-override detection",
    description:
      "Detects prompt payloads that attempt to supersede the system prompt.",
    layer: "input",
    action: "block",
    enabled: true,
    threshold: 0.78,
    triggers24h: 402_940,
    interventions24h: 3_918,
    updatedAt: daysAgo(21),
  },
  {
    id: "grd_4",
    name: "Role-play escape blocklist",
    description:
      "Blocks known persona-escape framings, including the DAN family of prompts.",
    layer: "input",
    action: "block",
    enabled: true,
    threshold: 0.7,
    triggers24h: 402_940,
    interventions24h: 612,
    updatedAt: daysAgo(4),
  },
  {
    id: "grd_5",
    name: "Secret and credential echo",
    description:
      "Stops models from echoing API keys or connection strings back to callers.",
    layer: "output",
    action: "block",
    enabled: true,
    threshold: 0.55,
    triggers24h: 398_004,
    interventions24h: 27,
    updatedAt: daysAgo(30),
  },
  {
    id: "grd_6",
    name: "PII in generated output",
    description:
      "Masks personal data introduced by the model that was not present in the input.",
    layer: "output",
    action: "mask",
    enabled: true,
    threshold: 0.68,
    triggers24h: 398_004,
    interventions24h: 1_284,
    updatedAt: daysAgo(9),
  },
  {
    id: "grd_7",
    name: "Toxicity scoring",
    description:
      "Scores responses for toxic content. Logged only, so thresholds can be tuned.",
    layer: "output",
    action: "log",
    enabled: true,
    threshold: 0.5,
    triggers24h: 398_004,
    interventions24h: 0,
    updatedAt: daysAgo(38),
  },
  {
    id: "grd_8",
    name: "Hallucinated citation check",
    description:
      "Verifies that cited documents exist in the retrieval index. Disabled pending a recall fix.",
    layer: "output",
    action: "warn",
    enabled: false,
    threshold: 0.8,
    triggers24h: 0,
    interventions24h: 0,
    updatedAt: daysAgo(56),
  },
];
