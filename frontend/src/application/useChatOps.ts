import { useState } from "react";
import type { ChatMessage, ChatResource, ChatSession, ProtoResource } from "../domain/chat";
import type { Model } from "../domain/llm";
import type { ApiAuth } from "../infrastructure/api/client";
import { createChatSession, deleteChatSession, sendChatMessage } from "../infrastructure/api/chatApi";
import type { SseEvent } from "../infrastructure/api/chatApi";
import { appConfig } from "../config";
import { mockChatResources, mockChatSession, mockChatSessions } from "./mockData";

function mapProtoResource(r: ProtoResource): ChatResource {
  return {
    kind: r.kind,
    namespace: r.namespace,
    name: r.name,
    phase: r.status,
  };
}

export function useChatOps(auth: ApiAuth) {
  const [sessions, setSessions] = useState<ChatSession[]>(appConfig.dataMode === "mock" ? mockChatSessions : []);
  const [session, setSession] = useState<ChatSession | null>(appConfig.dataMode === "mock" ? mockChatSession : null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [sending, setSending] = useState(false);

  async function startNewSession() {
    const nextSession = appConfig.dataMode === "mock"
      ? {
        id: `chat-session-${Date.now()}`,
        userId: "user-operator",
        status: "active",
        createdAt: new Date().toISOString(),
        title: "新会话"
      }
      : await createChatSession(auth);
    setSessions((current) => [nextSession, ...current]);
    setSession(nextSession);
    setMessages([]);
  }

  function selectSession(sessionId: string) {
    const nextSession = sessions.find((item) => item.id === sessionId);
    if (!nextSession) return;
    setSession(nextSession);
    setMessages([]);
  }

  async function deleteSession(sessionId: string) {
    if (appConfig.dataMode !== "mock") {
      await deleteChatSession(auth, sessionId);
    }
    setSessions((current) => current.filter((item) => item.id !== sessionId));
    if (session?.id === sessionId) {
      const nextSession = sessions.find((item) => item.id !== sessionId) ?? null;
      setSession(nextSession);
      setMessages([]);
    }
  }

  async function send(content: string, model: Model | undefined) {
    if (!model || !content.trim()) return;

    setSending(true);
    try {
      const activeSession = session ?? (appConfig.dataMode === "mock" ? mockChatSession : await createChatSession(auth));
      setSession(activeSession);
      setSessions((current) => current.some((item) => item.id === activeSession.id) ? current : [activeSession, ...current]);

      const userMessage: ChatMessage = { id: `user-${Date.now()}`, role: "user", content };
      const assistantMessage: ChatMessage = {
        id: `assistant-${Date.now()}`,
        role: "assistant",
        content: "",
        pending: true
      };
      setMessages((current) => [...current, userMessage, assistantMessage]);

      if (appConfig.dataMode === "mock") {
        setMessages((current) => current.map((message) => message.id === assistantMessage.id ? {
          ...message,
          content: "dev namespace 中发现 2 个异常 Pod：一个镜像拉取失败，一个 CrashLoopBackOff。\n\n建议先检查镜像地址、镜像凭据和最近的启动日志。",
          resources: mockChatResources,
          pending: false
        } : message));
        return;
      }

      await sendChatMessage(auth, activeSession.id, { modelId: model.id, content }, (event: SseEvent) => {
        setMessages((current) => current.map((message) => {
          if (message.id !== assistantMessage.id) return message;

          const e = event as Record<string, unknown>;

          // thinking event: show in a thinking field
          if (e.thinking && typeof e.thinking === "object") {
            const t = e.thinking as { content: string };
            return { ...message, thinking: (message.thinking || "") + t.content, content: "思考中..." };
          }

          // toolCall event
          if (e.toolCall && typeof e.toolCall === "object") {
            const tc = e.toolCall as { toolName: string; argumentsJson: string };
            const toolCalls = [...(message.toolCalls || []), { name: tc.toolName, args: tc.argumentsJson }];
            return { ...message, toolCalls, content: `调用工具：${tc.toolName}...` };
          }

          // toolResult event
          if (e.toolResult && typeof e.toolResult === "object") {
            const tr = e.toolResult as { toolName: string; success: boolean; resultJson: string };
            const toolCalls = (message.toolCalls || []).map((tc) =>
              tc.name === tr.toolName && !tc.result ? { ...tc, result: tr.resultJson, success: tr.success } : tc
            );
            return { ...message, toolCalls, content: `工具 ${tr.toolName} ${tr.success ? "完成" : "失败"}` };
          }

          // resource event
          if (e.resource && typeof e.resource === "object") {
            const r = e.resource as { resource?: ProtoResource };
            if (r.resource) {
              const newRes = mapProtoResource(r.resource);
              const resources = [...(message.resources || []), newRes];
              return { ...message, resources };
            }
            return message;
          }

          // complete event: final markdown summary
          if (e.complete && typeof e.complete === "object") {
            const c = e.complete as { summary: string; resources?: ProtoResource[] };
            const resources = (c.resources || []).map(mapProtoResource);
            return { ...message, content: c.summary, resources, pending: false };
          }

          // error event
          if (e.error && typeof e.error === "object") {
            const err = e.error as { code: string; message: string };
            return { ...message, content: err.message, pending: false };
          }

          return message;
        }));
      });
    } finally {
      setSending(false);
    }
  }

  function renameSession(sessionId: string, title: string) {
    setSessions((current) => current.map((s) => (s.id === sessionId ? { ...s, title } : s)));
    if (session?.id === sessionId) {
      setSession({ ...session, title });
    }
  }

  return { session, sessions, activeSessionId: session?.id ?? "", messages, sending, send, startNewSession, selectSession, deleteSession, renameSession };
}
