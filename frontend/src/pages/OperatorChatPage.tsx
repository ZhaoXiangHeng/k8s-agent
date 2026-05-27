import { useMemo, useRef, useState, useEffect } from "react";
import { useChatOps } from "../application/useChatOps";
import type { Model } from "../domain/llm";
import type { ApiAuth } from "../infrastructure/api/client";
import { DataTable } from "../components/DataTable";
import { EmptyState } from "../components/EmptyState";

function ResourceTable({ resources }: { resources: NonNullable<ReturnType<typeof useChatOps>["messages"][number]["resources"]> }) {
  return (
    <div className="messageResources">
      <DataTable>
        <thead><tr><th>Namespace</th><th>Kind</th><th>Name</th><th>Phase</th><th>Reason</th><th>Restarts</th><th>Node</th></tr></thead>
        <tbody>
          {resources.map((resource, index) => (
            <tr key={`${resource.namespace}-${resource.name}-${index}`}>
              <td>{resource.namespace || "-"}</td>
              <td>{resource.kind || "-"}</td>
              <td>{resource.name || "-"}</td>
              <td>{resource.phase || "-"}</td>
              <td>{resource.reason || "-"}</td>
              <td>{resource.restartCount ?? "-"}</td>
              <td>{resource.node || "-"}</td>
            </tr>
          ))}
        </tbody>
      </DataTable>
    </div>
  );
}

export function OperatorChatPage({ auth, models }: { auth: ApiAuth; models: Model[] }) {
  const [content, setContent] = useState("帮我看看现在集群里有什么异常吗？");
  const [modelId, setModelId] = useState("");
  const [showHistory, setShowHistory] = useState(false);
  const [searchText, setSearchText] = useState("");
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState("");
  const historyRef = useRef<HTMLDivElement>(null);
  const chat = useChatOps(auth);
  const selectedModel = useMemo(() => models.find((model) => model.id === (modelId || models[0]?.id)), [modelId, models]);

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
          {chat.messages.length === 0 ? <EmptyState title="输入自然语言运维指令开始巡检" /> : null}
          {chat.messages.map((message) => (
            <div key={message.id} className={`message ${message.role}`}>
              {message.content}
              {message.pending ? <span className="pending">分析中...</span> : null}
              {message.resources?.length ? <ResourceTable resources={message.resources} /> : null}
            </div>
          ))}
        </div>
        <div className="composer">
          <textarea
            value={content}
            onChange={(event) => setContent(event.target.value)}
            placeholder="输入自然语言运维指令..."
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                if (!chat.sending && selectedModel) chat.send(content, selectedModel);
              }
            }}
          />
          <button disabled={chat.sending || !selectedModel} onClick={() => void chat.send(content, selectedModel)}>
            {chat.sending ? "发送中" : "发送"}
          </button>
        </div>
      </div>
    </div>
  );
}
