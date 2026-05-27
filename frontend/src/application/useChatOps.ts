import { useState } from "react";
import type { ChatMessage, ChatResult, ChatSession } from "../domain/chat";
import type { Model } from "../domain/llm";
import type { ApiAuth } from "../infrastructure/api/client";
import { createChatSession, deleteChatSession, sendChatMessage } from "../infrastructure/api/chatApi";
import { appConfig } from "../config";
import { mockChatResources, mockChatSession, mockChatSessions } from "./mockData";

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
        content: "正在分析...",
        pending: true
      };
      setMessages((current) => [...current, userMessage, assistantMessage]);

      if (appConfig.dataMode === "mock") {
        setMessages((current) => current.map((message) => message.id === assistantMessage.id ? {
          ...message,
          content: "dev namespace 中发现 2 个异常 Pod：一个镜像拉取失败，一个 CrashLoopBackOff。建议先检查镜像地址、镜像凭据和最近的启动日志。",
          resources: mockChatResources,
          pending: false
        } : message));
        return;
      }

      await sendChatMessage(auth, activeSession.id, { modelId: model.id, content }, (event) => {
        const result = event as ChatResult;
        setMessages((current) => current.map((message) => {
          if (message.id !== assistantMessage.id) return message;
          if (result.error) return { ...message, content: result.error, pending: false };
          if (result.summary) {
            return {
              ...message,
              content: result.summary,
              resources: result.resources ?? [],
              pending: false
            };
          }
          if ("raw" in event) return { ...message, content: event.raw, pending: true };
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
