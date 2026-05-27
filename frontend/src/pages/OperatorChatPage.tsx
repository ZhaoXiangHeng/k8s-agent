import { useMemo, useRef, useState, useEffect } from "react";
import { useChatOps } from "../application/useChatOps";
import type { Model } from "../domain/llm";
import type { ChatMessage, ChatResource } from "../domain/chat";
import type { ApiAuth } from "../infrastructure/api/client";
import { DataTable } from "../components/DataTable";
import { EmptyState } from "../components/EmptyState";

// ---- simple markdown to HTML (no external lib) ----
function renderMarkdown(md: string): string {
  let html = md
    // escape HTML
    .replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;")
    // headers
    .replace(/^### (.+)$/gm, "<h4>$1</h4>")
    .replace(/^## (.+)$/gm, "<h3>$1</h3>")
    .replace(/^# (.+)$/gm, "<h2>$1</h2>")
    // bold & italic
    .replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")
    .replace(/\*(.+?)\*/g, "<em>$1</em>")
    // inline code
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    // links
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, "<a href=\"$2\" target=\"_blank\">$1</a>")
    // unordered lists
    .replace(/^- (.+)$/gm, "<li>$1</li>")
    .replace(/(<li>.*<\/li>)/s, "<ul>$1</ul>")
    // line breaks
    .replace(/\n\n/g, "</p><p>")
    .replace(/\n/g, "<br/>");

  html = "<p>" + html + "</p>";
  // fix nested <p> inside <li> etc — just clean up empty <p></p>
  html = html.replace(/<p><\/p>/g, "");
  // wrap adjacent <li> groups in <ul>
  html = html.replace(/((?:<li>.*?<\/li>)+)/g, "<ul>$1</ul>");
  // fix double ul
  html = html.replace(/<\/ul><ul>/g, "");

  return html;
}

// ---- resource table inside messages ----
function ResourceTable({ resources }: { resources: ChatResource[] }) {
  if (!resources || resources.length === 0) return null;
  return (
    <div className="messageResources">
      <DataTable>
        <thead><tr><th>Namespace</th><th>Kind</th><th>Name</th><th>Status</th></tr></thead>
        <tbody>
          {resources.map((r, i) => (
            <tr key={`${r.namespace}-${r.name}-${i}`}>
              <td>{r.namespace || "-"}</td>
              <td>{r.kind || "-"}</td>
              <td>{r.name || "-"}</td>
              <td>{r.phase || "-"}</td>
            </tr>
          ))}
        </tbody>
      </DataTable>
    </div>
  );
}

// ---- thinking display ----
function ThinkingBlock({ text }: { text: string }) {
  const [open, setOpen] = useState(false);
  if (!text) return null;
  return (
    <div className="thinkingBlock">
      <button className="link-btn" onClick={() => setOpen(!open)}>
        {open ? "🔽 隐藏思考过程" : "▶ 查看思考过程"}
      </button>
      {open && <pre className="thinkingContent">{text}</pre>}
    </div>
  );
}

function ToolCallBlock({ calls }: { calls: ChatMessage["toolCalls"] }) {
  if (!calls || calls.length === 0) return null;
  return (
    <div className="toolCallBlock">
      {calls.map((tc, i) => (
        <div key={i} className={`toolCallItem ${tc.success === false ? "toolCallFail" : ""}`}>
          <span className="toolCallName">🔧 {tc.name}</span>
          {tc.result && <span className="toolCallResult">{tc.success === false ? "失败" : "完成"}</span>}
        </div>
      ))}
    </div>
  );
}

// ---- message bubble ----
function MessageBubble({ message }: { message: ChatMessage }) {
  return (
    <div className={`message ${message.role}`}>
      {message.thinking && <ThinkingBlock text={message.thinking} />}
      {message.toolCalls && <ToolCallBlock calls={message.toolCalls} />}
      {message.content && (
        <div
          className="messageContent"
          dangerouslySetInnerHTML={{ __html: renderMarkdown(message.content) }}
        />
      )}
      {message.pending && message.role === "assistant" && (
        <span className="pending">{message.content || "分析中..."}</span>
      )}
      {message.resources && message.role === "assistant" && (
        <ResourceTable resources={message.resources} />
      )}
    </div>
  );
}

// ---- main page ----
export function OperatorChatPage({ auth, models }: { auth: ApiAuth; models: Model[] }) {
  const [content, setContent] = useState("");
  const [modelId, setModelId] = useState("");
  const [showHistory, setShowHistory] = useState(false);
  const [searchText, setSearchText] = useState("");
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState("");
  const historyRef = useRef<HTMLDivElement>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const chat = useChatOps(auth);
  const selectedModel = useMemo(() => models.find((model) => model.id === (modelId || models[0]?.id)), [modelId, models]);

  // Auto-scroll to bottom on new messages
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [chat.messages]);

  // Close dropdown on outside click
  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (historyRef.current && !historyRef.current.contains(e.target as Node)) {
        setShowHistory(false);
      }
    }
    if (showHistory) document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [showHistory]);

  const filteredSessions = searchText
    ? chat.sessions.filter((s) => (s.title || "未命名会话").toLowerCase().includes(searchText.toLowerCase()))
    : chat.sessions;

  function startRename(id: string, currentTitle: string) {
    setEditingId(id);
    setEditTitle(currentTitle || "");
  }

  function saveRename() {
    if (editingId && editTitle.trim()) {
      chat.renameSession(editingId, editTitle.trim());
    }
    setEditingId(null);
    setEditTitle("");
  }

  const handleSend = () => {
    if (!content.trim() || chat.sending || !selectedModel) return;
    chat.send(content, selectedModel);
    setContent("");
  };

  return (
    <div className="workspace">
      <header className="toolbar">
        <div>
          <p className="eyebrow">Operator</p>
          <h2>Chat 运维</h2>
        </div>
        <div className="chatToolbarActions" style={{ position: "relative" }}>
          <select value={modelId || models[0]?.id || ""} onChange={(event) => setModelId(event.target.value)}>
            {models.length === 0 ? <option value="">暂无可用模型</option> : null}
            {models.map((model) => (
              <option key={model.id} value={model.id}>{model.displayName || model.modelName}</option>
            ))}
          </select>
          <div ref={historyRef}>
            <button className="iconButton" aria-label="历史会话" title="历史会话" onClick={() => setShowHistory((c) => !c)}>◷</button>
            {showHistory && (
              <div className="sessionDropdown">
                <div className="sessionDropdownHeader">
                  <input
                    placeholder="搜索会话..."
                    value={searchText}
                    onChange={(e) => setSearchText(e.target.value)}
                    onClick={(e) => e.stopPropagation()}
                  />
                </div>
                <div className="sessionDropdownList">
                  {filteredSessions.length === 0 ? (
                    <div className="shuttle-empty">暂无会话</div>
                  ) : (
                    filteredSessions.map((session) => (
                      <div
                        key={session.id}
                        className={`sessionDropdownItem ${chat.activeSessionId === session.id ? "activeSession" : ""}`}
                      >
                        {editingId === session.id ? (
                          <div className="sessionEditRow">
                            <input
                              value={editTitle}
                              onChange={(e) => setEditTitle(e.target.value)}
                              onKeyDown={(e) => { if (e.key === "Enter") saveRename(); if (e.key === "Escape") setEditingId(null); }}
                              autoFocus
                              onClick={(e) => e.stopPropagation()}
                            />
                            <button className="link-btn" onClick={saveRename}>保存</button>
                          </div>
                        ) : (
                          <>
                            <button
                              className="sessionTitleBtn"
                              onClick={() => { chat.selectSession(session.id); setShowHistory(false); }}
                            >
                              <strong>{session.title || "未命名会话"}</strong>
                              <span>{new Date(session.createdAt).toLocaleString()}</span>
                            </button>
                            <div className="sessionItemActions">
                              <button
                                className="iconButton"
                                title="重命名"
                                onClick={(e) => { e.stopPropagation(); startRename(session.id, session.title || ""); }}
                                style={{ fontSize: 13, height: 28, width: 28 }}
                              >
                                ✏️
                              </button>
                              <button
                                className="iconButton dangerButton"
                                title="删除"
                                onClick={(e) => { e.stopPropagation(); void chat.deleteSession(session.id); }}
                                style={{ fontSize: 13, height: 28, width: 28 }}
                              >
                                ×
                              </button>
                            </div>
                          </>
                        )}
                      </div>
                    ))
                  )}
                </div>
              </div>
            )}
          </div>
          <button className="iconButton" aria-label="新建会话" title="新建会话" onClick={() => void chat.startNewSession()}>＋</button>
        </div>
      </header>

      <div className="chatPanel">
        <div className="chatMessages">
          {chat.messages.length === 0 ? <EmptyState title="选择一个模型，输入自然语言运维指令开始巡检" /> : null}
          {chat.messages.map((message) => (
            <MessageBubble key={message.id} message={message} />
          ))}
          <div ref={messagesEndRef} />
        </div>
        <div className="composer">
          <textarea
            value={content}
            onChange={(event) => setContent(event.target.value)}
            placeholder="输入自然语言运维指令..."
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                handleSend();
              }
            }}
          />
          <button disabled={chat.sending || !selectedModel} onClick={handleSend}>
            {chat.sending ? "发送中" : "发送"}
          </button>
        </div>
      </div>
    </div>
  );
}
