import { useState } from "react";
import type { CreateProviderRequest, Provider } from "../domain/llm";
import { DataTable } from "../components/DataTable";
import { EmptyState } from "../components/EmptyState";
import { StatusBadge } from "../components/StatusBadge";

const initialProvider: CreateProviderRequest = {
  name: "",
  protocol: "openai",
  baseUrl: "https://api.openai.com/v1",
  apiKey: "",
  enabled: true
};

export function AdminProvidersPage({
  providers,
  onCreate,
  onUpdate
  , embedded = false
}: {
  providers: Provider[];
  onCreate: (body: CreateProviderRequest) => Promise<void>;
  onUpdate: (id: string, body: Partial<CreateProviderRequest>) => Promise<void>;
  embedded?: boolean;
}) {
  const [form, setForm] = useState<CreateProviderRequest>(initialProvider);

  async function submit() {
    await onCreate(form);
    setForm(initialProvider);
  }

  return (
    <div className={embedded ? "embeddedPage" : "workspace"}>
      {embedded ? null : <header className="toolbar"><h2>LLM Provider</h2></header>}
      <section className="panel formGrid">
        <label className="formRow">名称<input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></label>
        <label className="formRow">协议<select value={form.protocol} onChange={(e) => setForm({ ...form, protocol: e.target.value as "openai" | "anthropic" })}><option value="openai">openai</option><option value="anthropic">anthropic</option></select></label>
        <label className="formRow">Base URL<input value={form.baseUrl} onChange={(e) => setForm({ ...form, baseUrl: e.target.value })} /></label>
        <label className="formRow">API Key<input type="password" value={form.apiKey} onChange={(e) => setForm({ ...form, apiKey: e.target.value })} /></label>
        <label className="checkRow"><input type="checkbox" checked={form.enabled} onChange={(e) => setForm({ ...form, enabled: e.target.checked })} />启用</label>
        <button onClick={() => void submit()}>创建 Provider</button>
      </section>
      <section className="panel">
        {providers.length === 0 ? <EmptyState title="暂无 Provider" /> : (
          <DataTable>
            <thead><tr><th>名称</th><th>协议</th><th>Base URL</th><th>API Key</th><th>状态</th><th>操作</th></tr></thead>
            <tbody>
              {providers.map((provider) => (
                <tr key={provider.id}>
                  <td>{provider.name}</td>
                  <td>{provider.protocol}</td>
                  <td>{provider.baseUrl}</td>
                  <td>{provider.apiKeyConfigured ? "已配置" : "未配置"}</td>
                  <td><StatusBadge active={provider.enabled} text={provider.enabled ? "启用" : "停用"} /></td>
                  <td><button onClick={() => void onUpdate(provider.id, { enabled: !provider.enabled })}>{provider.enabled ? "停用" : "启用"}</button></td>
                </tr>
              ))}
            </tbody>
          </DataTable>
        )}
      </section>
    </div>
  );
}
