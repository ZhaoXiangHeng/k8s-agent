const abnormalPods = [
  {
    namespace: "dev",
    name: "api-7b8f9",
    phase: "Pending",
    reason: "ImagePullBackOff",
    message: "Back-off pulling image",
    restartCount: 0,
    node: "kind-worker",
  },
  {
    namespace: "dev",
    name: "worker-5d9c7",
    phase: "Running",
    reason: "CrashLoopBackOff",
    message: "Container exits after startup",
    restartCount: 6,
    node: "kind-worker2",
  },
];

export default function OperatorConsole() {
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
