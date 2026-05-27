import { useState } from "react";

type View = "admin" | "operator";

const abnormalPods = [
  {
    namespace: "dev",
    name: "api-7b8f9",
    phase: "Pending",
    reason: "ImagePullBackOff",
    message: "Back-off pulling image",
    restartCount: 0,
    node: "kind-worker"
  },
  {
    namespace: "dev",
    name: "worker-5d9c7",
    phase: "Running",
    reason: "CrashLoopBackOff",
    message: "Container exits after startup",
    restartCount: 6,
    node: "kind-worker2"
  }
];

export default function App() {
  const [view, setView] = useState<View>("operator");

  return (
    <main className="shell">
      <aside className="sidebar">
        <div>
          <p className="eyebrow">K8S AI Ops</p>
          <h1>AI 运维控制台</h1>
        </div>
        <nav>
          <button className={view === "operator" ? "active" : ""} onClick={() => setView("operator")}>
            Operator Chat
          </button>
          <button className={view === "admin" ? "active" : ""} onClick={() => setView("admin")}>
            Admin Console
          </button>
        </nav>
      </aside>

      <section className="content">
        {view === "operator" ? <OperatorConsole /> : <AdminConsole />}
      </section>
    </main>
  );
}

function OperatorConsole() {
  return (
    <div className="workspace">
      <header className="toolbar">
        <div>
          <p className="eyebrow">Operator</p>
          <h2>Chat 运维</h2>
        </div>
        <select defaultValue="mock-local">
          <option value="mock-local">Mock Local</option>
          <option value="gpt">GPT Production</option>
          <option value="claude">Claude Production</option>
        </select>
      </header>

      <section className="chat">
        <div className="message user">帮我看看现在集群里有什么异常吗？</div>
        <div className="message assistant">
          dev namespace 中有 2 个异常 Pod，主要集中在镜像拉取和容器启动失败。
        </div>
        <div className="composer">
          <input placeholder="输入自然语言运维指令" />
          <button>发送</button>
        </div>
      </section>

      <section>
        <h3>异常 Pod</h3>
        <div className="tableWrap">
          <table>
            <thead>
              <tr>
                <th>Namespace</th>
                <th>Pod</th>
                <th>Phase</th>
                <th>Reason</th>
                <th>Restarts</th>
                <th>Node</th>
              </tr>
            </thead>
            <tbody>
              {abnormalPods.map((pod) => (
                <tr key={pod.name}>
                  <td>{pod.namespace}</td>
                  <td>{pod.name}</td>
                  <td>{pod.phase}</td>
                  <td>{pod.reason}</td>
                  <td>{pod.restartCount}</td>
                  <td>{pod.node}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}

function AdminConsole() {
  return (
    <div className="workspace">
      <header className="toolbar">
        <div>
          <p className="eyebrow">Admin</p>
          <h2>平台管理</h2>
        </div>
        <button>创建操作员</button>
      </header>

      <div className="grid">
        <section>
          <h3>用户与角色</h3>
          <p>管理员：admin@example.com</p>
          <p>操作员：operator@example.com</p>
        </section>
        <section>
          <h3>K8S 权限</h3>
          <p>operator@example.com: dev / pods / get,list,watch</p>
          <p>operator@example.com: dev / deployments / get,list,patch</p>
        </section>
        <section>
          <h3>LLM 模型</h3>
          <p>OpenAI: gpt-4.1</p>
          <p>Anthropic: claude-3-5-sonnet</p>
        </section>
        <section>
          <h3>审计</h3>
          <p>最近一次工具调用：list_pods dev allowed</p>
        </section>
      </div>
    </div>
  );
}
