import type { Model } from "../domain/llm";
import { DataTable } from "../components/DataTable";
import { EmptyState } from "../components/EmptyState";
import { Notice } from "../components/Notice";
import { StatusBadge } from "../components/StatusBadge";

export function OperatorModelsPage({ models, loading, error }: { models: Model[]; loading: boolean; error: string }) {
  return (
    <div className="workspace">
      <header className="toolbar"><h2>可用模型</h2></header>
      {error ? <Notice type="error">{error}</Notice> : null}
      <section className="panel">
        {loading ? <p>正在加载...</p> : null}
        {!loading && models.length === 0 ? <EmptyState title="暂无可用模型" /> : null}
        {models.length > 0 ? (
          <DataTable>
            <thead><tr><th>模型</th><th>显示名</th><th>Provider</th><th>工具</th><th>流式</th><th>状态</th></tr></thead>
            <tbody>
              {models.map((model) => (
                <tr key={model.id}>
                  <td>{model.modelName}</td>
                  <td>{model.displayName || model.modelName}</td>
                  <td>{model.providerId}</td>
                  <td><StatusBadge active={model.supportsTools} text={model.supportsTools ? "支持" : "不支持"} /></td>
                  <td><StatusBadge active={model.supportsStreaming} text={model.supportsStreaming ? "支持" : "不支持"} /></td>
                  <td><StatusBadge active={model.enabled} text={model.enabled ? "启用" : "停用"} /></td>
                </tr>
              ))}
            </tbody>
          </DataTable>
        ) : null}
      </section>
    </div>
  );
}
