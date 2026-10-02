import type { ID, ISODateString } from "./common";
import type { ModelModality } from "./model";

export interface ConversationTurn {
  id: ID;
  role: "user" | "assistant" | "tool" | "system";
  content: string;
  createdAt: ISODateString;
  /** Token count attributed to this turn. */
  tokens: number;
  /** Set on tool turns. */
  toolName?: string;
}

export interface Conversation {
  id: ID;
  title: string;
  userId: ID;
  userName: string;
  projectId: ID;
  projectName: string;
  modelId: ID;
  modelName: string;
  startedAt: ISODateString;
  endedAt?: ISODateString;
  turnCount: number;
  inputTokens: number;
  outputTokens: number;
  cost: number;
  /** Redacted flag: true when content matched a PII or jailbreak detector. */
  flagged: boolean;
  flagReason?: string;
  turns: ConversationTurn[];
}

export interface ConversationSummary {
  modality: ModelModality;
  count: number;
  cost: number;
}
