import type { ID, ISODateString, StatusTone } from "./common";

export type GuardrailAction = "block" | "mask" | "warn" | "log";

export type GuardrailLayer = "input" | "output";

export interface Guardrail {
  id: ID;
  name: string;
  description: string;
  layer: GuardrailLayer;
  action: GuardrailAction;
  enabled: boolean;
  /** Detector tuning knob, 0-1. Higher is stricter. */
  threshold: number;
  /** Rolling 24h inspections. */
  triggers24h: number;
  /** Rolling 24h interventions (block or mask applied). */
  interventions24h: number;
  updatedAt: ISODateString;
}

export const GUARDRAIL_ACTION_TONE: Record<GuardrailAction, StatusTone> = {
  block: "danger",
  mask: "warning",
  warn: "info",
  log: "neutral",
};
