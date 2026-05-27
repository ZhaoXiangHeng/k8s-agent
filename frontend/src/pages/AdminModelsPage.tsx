import { useState } from "react";
import type { CreateModelRequest, Model, Provider } from "../domain/llm";
import { DataTable } from "../components/DataTable";
import { EmptyState } from "../components/EmptyState";
import { StatusBadge } from "../components/StatusBadge";

const initialModel: CreateModelRequest = {
  providerId: "",
  modelName: "",
  displayName: "",
  supportsTools: true,
  supportsStreaming: true,
  enabled: true
};

export function AdminModelsPage({
  models,
  providers,
  onCreate,
  onUpdate,
  onDelete,
  embedded = false
}: {
  models: Model[];
  providers: Provider[];
  onCreate: (body: CreateModelRequest) => Promise<void>;
  onUpdate: (id: string, body: Partial<CreateModelRequest>) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
  embedded?: boolean;
}) {
  const [form, setForm] = useState<CreateModelRequest>(initialModel);

  async function submit() {
    await onCreate({ ...form, providerId: form.providerId || providers[0]?.id || "" });
    setForm(initialModel);
  }

  return (
    <div className={embedded ? "embeddedPage" : "workspace"}>
      {embedded ? null : <header className="toolbar"><h2>LLM Model</h2></header>}
      <section className="panel formGrid">
        <label className="formRow">Provider<select value={form.providerId || providers[0]?.id || ""} onChange={(e) => setForm({ ...form, providerId: e.target.value })}>{providers.map((provider) => <option key={provider.id} value={provider.id}>{provider.name}</option>)}</select></label>
        <label className="formRow">模型名<input value={form.modelName} onChange={(e) => setForm({ ...form, modelName: e.target.value })} /></label>
        <label className="formRow">显示名<input value={form.displayName} onChange={(e) => setForm({ ...form, displayName: e.target.value })} /></label>
        <label className="checkRow"><input type="checkbox" checked={form.supportsTools} onChange={(e) => setForm({ ...form, supportsTools: e.target.checked })} />支持工具</label>
        <label className="checkRow"><input type="checkbox" checked={form.supportsStreaming} onChange={(e) => setForm({ ...form, supportsStreaming: e.target.checked })} />支持流式</label>
        <label className="checkRow"><input type="checkbox" checked={form.enabled} onChange={(e) => setForm({ ...form, enabled: e.target.checked })} />启用</label>
        <button onClick={() => void submit()}>创建 Model</button>
      </section>
      <section className="panel">
        {models.length === 0 ? <EmptyState title="暂无 Model" /> : (
          <DataTable>
            <thead><tr><th>模型</th><th>显示名</th><th>Provider</th><th>工具</th><th>流式</th><th>状态</th><th>操作</th></tr></thead>
            <tbody>
              {models.map((model) => (
                <tr key={model.id}>
                  <td>{model.modelName}</td>
                  <td>{model.displayName || model.modelName}</td>
                  <td>{model.providerId}</td>
                  <td><StatusBadge active={model.supportsTools} text={model.supportsTools ? "支持" : "不支持"} /></td>
                  <td><StatusBadge active={model.supportsStreaming} text={model.supportsStreaming ? "支持" : "不支持"} /></td>
                  <td><StatusBadge active={model.enabled} text={model.enabled ? "启用" : "停用"} /></td>
                  <td>
                    <div className="rowActions">
                      <button onClick={() => void onUpdate(model.id, { enabled: !model.enabled })}>{model.enabled ? "停用" : "启用"}</button>
                      <button className="dangerButton" onClick={() => void onDelete(model.id)}>删除模型</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </DataTable>
        )}
      </section>
    </div>
  );
}
