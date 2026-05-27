export type ChatSession = {
  id: string;
  userId: string;
  status: string;
  createdAt: string;
  title?: string;
};

export type ChatResource = {
  namespace?: string;
  kind?: string;
  name?: string;
  phase?: string;
  reason?: string;
  message?: string;
  restartCount?: number;
  node?: string;
};

// Proto StreamEvent from backend (camelCase JSON via protojson)
export type ProtoStreamEvent = {
  eventId?: string;
  timestamp?: number;
  thinking?: { content: string };
  toolCall?: { toolName: string; argumentsJson: string };
  toolResult?: { toolName: string; success: boolean; resultJson: string };
  resource?: { resource?: ProtoResource };
  complete?: { summary: string; resources?: ProtoResource[] };
  error?: { code: string; message: string };
};

export type ProtoResource = {
  kind?: string;
  apiGroup?: string;
  namespace?: string;
  name?: string;
  status?: string;
};

export type ChatResult = {
  messageId?: string;
  summary?: string;
  resources?: ChatResource[];
  error?: string;
};

export type ChatMessage = {
  id: string;
  role: "user" | "assistant" | "system";
  content: string;
  resources?: ChatResource[];
  pending?: boolean;
  thinking?: string;
  toolCalls?: { name: string; args: string; result?: string; success?: boolean }[];
};
