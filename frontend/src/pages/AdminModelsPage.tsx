import { useEffect, useRef, useState } from "react";
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
  enabled: true,
};

export function AdminModelsPage({
  models,
  providers,
  onCreate,
  onUpdate,
  onDelete,
  preselectedProviderId,
  embedded = false,
}: {
  models: Model[];
  providers: Provider[];
  onCreate: (body: CreateModelRequest) => Promise<void>;
  onUpdate: (id: string, body: Partial<CreateModelRequest>) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
  preselectedProviderId?: string;
  embedded?: boolean;
}) {
  const [drawer, setDrawer] = useState<"create" | null>(null);
  const [filterProviderId, setFilterProviderId] = useState(preselectedProviderId || "");
  const [form, setForm] = useState<CreateModelRequest>(() =>
    preselectedProviderId ? { ...initialModel, providerId: preselectedProviderId } : initialModel
  );
  const [menuOpen, setMenuOpen] = useState<string | null>(null);
  const [menuPos, setMenuPos] = useState({ top: 0, left: 0 });
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(null);
      }
    }
    if (menuOpen) document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [menuOpen]);

  useEffect(() => {
    if (preselectedProviderId) {
      setFilterProviderId(preselectedProviderId);
      setForm((prev) => ({ ...prev, providerId: preselectedProviderId }));
    }
  }, [preselectedProviderId]);

  const filteredModels = filterProviderId
    ? models.filter((m) => m.providerId === filterProviderId)
    : models;

  function openCreate() {
    setForm(
      filterProviderId
        ? { ...initialModel, providerId: filterProviderId }
        : initialModel
    );
    setDrawer("create");
  }

  async function save() {
    await onCreate({ ...form, providerId: form.providerId || providers[0]?.id || "" });
    setDrawer(null);
    setForm(initialModel);
  }

  return (
    <div className={embedded ? "embeddedPage" : "workspace"}>
      <header className="toolbar">
        {embedded ? <h3>模型列表</h3> : <h2>LLM Model</h2>}
        <div className="chatToolbarActions">
          <select
            value={filterProviderId}
            onChange={(e) => setFilterProviderId(e.target.value)}
          >
            <option value="">全部 Provider</option>
            {providers.map((p) => (
              <option key={p.id} value={p.id}>{p.name}</option>
            ))}
          </select>
          <button className="iconButton" onClick={openCreate} title="新建 Model">
            ＋
          </button>
        </div>
      </header>

      <section className="panel">
        {filteredModels.length === 0 ? (
          <EmptyState title={filterProviderId ? "该 Provider 暂无模型" : "暂无 Model"} />
        ) : (
          <DataTable>
            <thead>
              <tr>
                <th>模型</th>
                <th>显示名</th>
                <th>Provider</th>
                <th>工具</th>
                <th>流式</th>
                <th>状态</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {filteredModels.map((model) => (
                <tr
                  key={model.id}
                  className={
                    preselectedProviderId && model.providerId === preselectedProviderId
                      ? "selectedRow"
                      : ""
                  }
                >
                  <td>{model.modelName}</td>
                  <td>{model.displayName || model.modelName}</td>
                  <td>
                    {providers.find((p) => p.id === model.providerId)?.name || model.providerId}
                  </td>
                  <td>
                    <StatusBadge active={model.supportsTools} text={model.supportsTools ? "支持" : "不支持"} />
                  </td>
                  <td>
                    <StatusBadge active={model.supportsStreaming} text={model.supportsStreaming ? "支持" : "不支持"} />
                  </td>
                  <td>
                    <StatusBadge active={model.enabled} text={model.enabled ? "启用" : "停用"} />
                  </td>
                  <td>
                    <div className="rowActions" style={{ position: "relative" }}>
                      <button
                        className="iconButton"
                        onClick={(e) => {
                          e.stopPropagation();
                          if (menuOpen === model.id) {
                            setMenuOpen(null);
                          } else {
                            setMenuPos({ top: e.clientY, left: e.clientX - 140 });
                            setMenuOpen(model.id);
                          }
                        }}
                        title="更多操作"
                      >
                        ···
                      </button>
                      {menuOpen === model.id && (
                        <div
                          className="contextMenu"
                          ref={menuRef}
                          style={{ top: menuPos.top, left: menuPos.left }}
                        >
                          <button onClick={() => void onUpdate(model.id, { enabled: !model.enabled })}>
                            {model.enabled ? "🔴 停用" : "🟢 启用"}
                          </button>
                          <button className="dangerButton" onClick={() => { setMenuOpen(null); void onDelete(model.id); }}>
                            🗑️ 删除
                          </button>
                        </div>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </DataTable>
        )}
      </section>

      {/* Create drawer */}
      {drawer && (
        <div className="drawerOverlay" onClick={() => setDrawer(null)}>
          <div className="drawerPanel" onClick={(e) => e.stopPropagation()}>
            <header className="drawerHeader">
              <h3>新建 Model</h3>
              <button className="iconButton" onClick={() => setDrawer(null)}>×</button>
            </header>
            <label className="formRow">
              Provider
              <select
                value={form.providerId || providers[0]?.id || ""}
                onChange={(e) => setForm({ ...form, providerId: e.target.value })}
              >
                {providers.map((provider) => (
                  <option key={provider.id} value={provider.id}>
                    {provider.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="formRow">
              模型名
              <input value={form.modelName} onChange={(e) => setForm({ ...form, modelName: e.target.value })} placeholder="例如：gpt-4.1" />
            </label>
            <label className="formRow">
              显示名
              <input value={form.displayName} onChange={(e) => setForm({ ...form, displayName: e.target.value })} placeholder="可选" />
            </label>
            <label className="checkRow">
              <input type="checkbox" checked={form.supportsTools} onChange={(e) => setForm({ ...form, supportsTools: e.target.checked })} />
              支持工具
            </label>
            <label className="checkRow">
              <input type="checkbox" checked={form.supportsStreaming} onChange={(e) => setForm({ ...form, supportsStreaming: e.target.checked })} />
              支持流式
            </label>
            <label className="checkRow">
              <input type="checkbox" checked={form.enabled} onChange={(e) => setForm({ ...form, enabled: e.target.checked })} />
              启用
            </label>
            <div className="actions">
              <button onClick={() => setDrawer(null)}>取消</button>
              <button onClick={() => void save()}>保存</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
