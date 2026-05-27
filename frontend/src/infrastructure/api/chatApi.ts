import type { ChatResult, ChatSession } from "../../domain/chat";
import { apiRequest, buildAuthHeaders, type ApiAuth } from "./client";

export type RawSseEvent = ChatResult | { raw: string };

export function parseSseChunk(chunk: string): RawSseEvent[] {
  return chunk
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line.startsWith("data:"))
    .map((line) => line.slice(5).trim())
    .filter(Boolean)
    .map((data) => {
      try {
        return JSON.parse(data) as ChatResult;
      } catch {
        return { raw: data };
      }
    });
}

export async function createChatSession(auth: ApiAuth): Promise<ChatSession> {
  return apiRequest<ChatSession>("/api/operator/chat/sessions", { method: "POST", auth });
}

export async function deleteChatSession(auth: ApiAuth, sessionId: string): Promise<void> {
  await apiRequest<void>(`/api/operator/chat/sessions/${sessionId}`, { method: "DELETE", auth });
}

export async function sendChatMessage(
  auth: ApiAuth,
  sessionId: string,
  body: { modelId: string; content: string },
  onEvent: (event: RawSseEvent) => void
): Promise<void> {
  const response = await fetch(`/api/operator/chat/sessions/${sessionId}/messages`, {
    method: "POST",
    headers: {
      ...buildAuthHeaders(auth),
      "Content-Type": "application/json"
    },
    body: JSON.stringify(body)
  });

  if (!response.ok || !response.body) {
    onEvent({ error: `Chat 请求失败：HTTP ${response.status}` });
    return;
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const { done, value } = await reader.read();
    if (done) {
      if (buffer) {
        parseSseChunk(buffer).forEach(onEvent);
      }
      break;
    }
    buffer += decoder.decode(value, { stream: true });
    const chunks = buffer.split("\n\n");
    buffer = chunks.pop() ?? "";
    chunks.forEach((chunk) => parseSseChunk(chunk).forEach(onEvent));
  }
}
