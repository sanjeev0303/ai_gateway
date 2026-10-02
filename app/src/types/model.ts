import type { ID, ISODateString, StatusTone } from "./common";

export type ModelModality = "text" | "vision" | "audio" | "embedding" | "multimodal";

export type ModelStatus = "ga" | "preview" | "deprecated" | "retired";

/** Price per one million tokens, in whole USD. */
export interface ModelPricing {
  inputPerMTok: number;
  outputPerMTok: number;
}

export interface Model {
  id: ID;
  /** Machine name, e.g. "claude-sonnet-4-5". */
  name: string;
  displayName: string;
  providerId: ID;
  modality: ModelModality;
  status: ModelStatus;
  contextWindowTokens: number;
  maxOutputTokens: number;
  pricing: ModelPricing;
  createdAt: ISODateString;
  /** Spend over the trailing 30 days, in whole USD. */
  spend30d: number;
  callCount30d: number;
  /** 0-100 composite of quality, latency and cost. */
  healthScore: number;
}

export const MODEL_STATUS_TONE: Record<ModelStatus, StatusTone> = {
  ga: "success",
  preview: "info",
  deprecated: "warning",
  retired: "danger",
};
