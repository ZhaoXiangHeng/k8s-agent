import { useMemo, useState } from "react";
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
  const chat = useChatOps(auth);
  const selectedModel = useMemo(() => models.find((model) => model.id === (modelId || models[0]?.id)), [modelId, models]);

  return (
    <div className="workspace">
      <header className="toolbar">
        <div>
          <p className="eyebrow">Operator</p>
          <h2>Chat 运维</h2>
        </div>
        <div className="chatToolbarActions">
          <select value={modelId || models[0]?.id || ""} onChange={(event) => setModelId(event.target.value)}>
            {models.length === 0 ? <option value="">暂无可用模型</option> : null}
            {models.map((model) => (
              <option key={model.id} value={model.id}>{model.displayName || model.modelName}</option>
            ))}
          </select>
          <button className="iconButton" aria-label="历史会话" title="历史会话" onClick={() => setShowHistory((current) => !current)}>◷</button>
          <button className="iconButton" aria-label="新建会话" title="新建会话" onClick={() => void chat.startNewSession()}>＋</button>
        </div>
      </header>
      {showHistory ? (
        <section className="panel sessionHistory">
          {chat.sessions.length === 0 ? <EmptyState title="暂无历史会话" /> : chat.sessions.map((session) => (
            <div key={session.id} className={`sessionItem ${chat.activeSessionId === session.id ? "activeSession" : ""}`}>
              <button onClick={() => chat.selectSession(session.id)}>
                <strong>{session.title || "未命名会话"}</strong>
                <span>{new Date(session.createdAt).toLocaleString()}</span>
              </button>
              <button
                className="iconButton dangerButton"
                aria-label={`删除会话 ${session.title || "未命名会话"}`}
                title="删除会话"
                onClick={() => void chat.deleteSession(session.id)}
              >
                ×
              </button>
            </div>
          ))}
        </section>
      ) : null}
      <section className="panel chatMessages">
        {chat.messages.length === 0 ? <EmptyState title="输入自然语言运维指令开始巡检" /> : null}
        {chat.messages.map((message) => (
          <div key={message.id} className={`message ${message.role}`}>
            {message.content}
            {message.pending ? <span className="pending"> 分析中</span> : null}
            {message.resources?.length ? <ResourceTable resources={message.resources} /> : null}
          </div>
        ))}
        <div className="composer">
          <textarea value={content} onChange={(event) => setContent(event.target.value)} />
          <button disabled={chat.sending || !selectedModel} onClick={() => void chat.send(content, selectedModel)}>
            {chat.sending ? "发送中" : "发送"}
          </button>
        </div>
      </section>
    </div>
  );
}
