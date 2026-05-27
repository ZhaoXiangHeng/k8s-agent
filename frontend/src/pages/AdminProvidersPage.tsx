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
  enabled: true,
};

export function AdminProvidersPage({
  providers,
  onCreate,
  onUpdate,
  onEditModels,
  embedded = false,
}: {
  providers: Provider[];
  onCreate: (body: CreateProviderRequest) => Promise<void>;
  onUpdate: (id: string, body: Partial<CreateProviderRequest>) => Promise<void>;
  onEditModels?: (providerId: string) => void;
  embedded?: boolean;
}) {
  const [drawer, setDrawer] = useState<"create" | "edit" | null>(null);
  const [editTarget, setEditTarget] = useState<Provider | null>(null);
  const [form, setForm] = useState<CreateProviderRequest>(initialProvider);

  function openCreate() {
    setForm(initialProvider);
    setEditTarget(null);
    setDrawer("create");
  }

  function openEdit(provider: Provider) {
    setEditTarget(provider);
    setForm({
      name: provider.name,
      protocol: provider.protocol as "openai" | "anthropic",
      baseUrl: provider.baseUrl,
      apiKey: "",
      enabled: provider.enabled,
    });
    setDrawer("edit");
  }

  async function save() {
    if (drawer === "create") {
      await onCreate(form);
    } else if (drawer === "edit" && editTarget) {
      const updateFields: Partial<CreateProviderRequest> = {
        name: form.name,
        protocol: form.protocol,
        baseUrl: form.baseUrl,
        enabled: form.enabled,
      };
      if (form.apiKey) updateFields.apiKey = form.apiKey;
      await onUpdate(editTarget.id, updateFields);
    }
    setDrawer(null);
    setForm(initialProvider);
  }

  function closeDrawer() {
    setDrawer(null);
    setForm(initialProvider);
  }

  return (
    <div className={embedded ? "embeddedPage" : "workspace"}>
      <header className="toolbar">
        {embedded ? <h3>Provider 列表</h3> : <h2>LLM Provider</h2>}
        <button className="iconButton" onClick={openCreate} title="新建 Provider">
          ＋
        </button>
      </header>

      <section className="panel">
        {providers.length === 0 ? (
          <EmptyState title="暂无 Provider" />
        ) : (
          <DataTable>
            <thead>
              <tr>
                <th>名称</th>
                <th>协议</th>
                <th>Base URL</th>
                <th>API Key</th>
                <th>状态</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {providers.map((provider) => (
                <tr key={provider.id}>
                  <td>{provider.name}</td>
                  <td>{provider.protocol}</td>
                  <td>{provider.baseUrl}</td>
                  <td>{provider.apiKeyConfigured ? "已配置" : "未配置"}</td>
                  <td>
                    <StatusBadge active={provider.enabled} text={provider.enabled ? "启用" : "停用"} />
                  </td>
                  <td>
                    <div className="rowActions">
                      <button onClick={() => openEdit(provider)}>编辑</button>
                      <button onClick={() => void onUpdate(provider.id, { enabled: !provider.enabled })}>
                        {provider.enabled ? "停用" : "启用"}
                      </button>
                      {onEditModels && (
                        <button onClick={() => onEditModels(provider.id)}>编辑模型</button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </DataTable>
        )}
      </section>

      {/* Create / Edit drawer */}
      {drawer && (
        <div className="drawerOverlay" onClick={closeDrawer}>
          <div className="drawerPanel" onClick={(e) => e.stopPropagation()}>
            <header className="drawerHeader">
              <h3>{drawer === "create" ? "新建 Provider" : `编辑 Provider — ${editTarget?.name || ""}`}</h3>
              <button className="iconButton" onClick={closeDrawer}>×</button>
            </header>
            <label className="formRow">
              名称
              <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="例如：openai-prod" />
            </label>
            <label className="formRow">
              协议
              <select value={form.protocol} onChange={(e) => setForm({ ...form, protocol: e.target.value as "openai" | "anthropic" })}>
                <option value="openai">openai</option>
                <option value="anthropic">anthropic</option>
              </select>
            </label>
            <label className="formRow">
              Base URL
              <input value={form.baseUrl} onChange={(e) => setForm({ ...form, baseUrl: e.target.value })} />
            </label>
            <label className="formRow">
              API Key{drawer === "edit" ? "（留空不修改）" : ""}
              <input type="password" value={form.apiKey} onChange={(e) => setForm({ ...form, apiKey: e.target.value })} placeholder={drawer === "edit" ? "留空则不修改" : ""} />
            </label>
            <label className="checkRow">
              <input type="checkbox" checked={form.enabled} onChange={(e) => setForm({ ...form, enabled: e.target.checked })} />
              启用
            </label>
            <div className="actions">
              <button onClick={closeDrawer}>取消</button>
              <button onClick={() => void save()}>保存</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
