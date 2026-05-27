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
};
