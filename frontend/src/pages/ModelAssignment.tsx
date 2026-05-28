import { useEffect, useMemo, useState } from "react";
import { Notice } from "../components/Notice";
import type { Model, Provider } from "../domain/llm";
import type { User } from "../domain/user";
import type { UserModelBindingMap } from "../domain/userModelBinding";

type ModelAssignmentProps = {
  users: User[];
  models: Model[];
  providers: Provider[];
  modelBindingsByUserId: UserModelBindingMap;
  onUpdateModelBindings: (userId: string, modelIds: string[]) => Promise<void>;
  preselectedUserId?: string;
};

export default function ModelAssignment({
  users,
  models,
  providers,
  modelBindingsByUserId,
  onUpdateModelBindings,
  preselectedUserId,
}: ModelAssignmentProps) {
  const [selectedUserId, setSelectedUserId] = useState<string>(preselectedUserId ?? (users[0]?.id || ""));
  const [leftSelected, setLeftSelected] = useState<Set<string>>(new Set());
  const [rightSelected, setRightSelected] = useState<Set<string>>(new Set());
  const [displayAssignedIds, setDisplayAssignedIds] = useState<string[]>(modelBindingsByUserId[selectedUserId] ?? []);
  const [submitting, setSubmitting] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  const providerNameById = useMemo(
    () => Object.fromEntries(providers.map((provider) => [provider.id, provider.name])),
    [providers]
  );
  const modelById = useMemo(
    () => Object.fromEntries(models.map((model) => [model.id, model])),
    [models]
  );
  const assignedModels = displayAssignedIds
    .map((modelId) => modelById[modelId])
    .filter((model): model is Model => Boolean(model));
  const assignedSet = new Set(displayAssignedIds);
  const availableModels = models.filter((model) => !assignedSet.has(model.id));
  const persistedIds = modelBindingsByUserId[selectedUserId] ?? [];
  const isDirty = displayAssignedIds.join("|") !== persistedIds.join("|");

  useEffect(() => {
    if (preselectedUserId) {
      selectUser(preselectedUserId);
    }
  }, [preselectedUserId]);

  useEffect(() => {
    if (!selectedUserId && users[0]?.id) {
      setSelectedUserId(users[0].id);
    }
  }, [selectedUserId, users]);

  useEffect(() => {
    setDisplayAssignedIds(modelBindingsByUserId[selectedUserId] ?? []);
  }, [modelBindingsByUserId, selectedUserId]);

  const selectUser = (userId: string) => {
    setSelectedUserId(userId);
    setDisplayAssignedIds(modelBindingsByUserId[userId] ?? []);
    setLeftSelected(new Set());
    setRightSelected(new Set());
    setMessage("");
    setError("");
  };

  const toggleLeft = (modelId: string) => {
    const next = new Set(leftSelected);
    next.has(modelId) ? next.delete(modelId) : next.add(modelId);
    setLeftSelected(next);
  };

  const toggleRight = (modelId: string) => {
    const next = new Set(rightSelected);
    next.has(modelId) ? next.delete(modelId) : next.add(modelId);
    setRightSelected(next);
  };

  const assignModels = () => {
    const ids = Array.from(leftSelected);
    setDisplayAssignedIds([...displayAssignedIds, ...ids]);
    setLeftSelected(new Set());
  };

  const revokeModels = () => {
    const ids = Array.from(rightSelected);
    setDisplayAssignedIds(displayAssignedIds.filter((modelId) => !ids.includes(modelId)));
    setRightSelected(new Set());
  };

  const handleSave = async () => {
    if (!selectedUserId) return;
    setSubmitting(true);
    setMessage("");
    setError("");
    try {
      await onUpdateModelBindings(selectedUserId, displayAssignedIds);
      setMessage("模型绑定已保存");
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存模型绑定失败");
    } finally {
      setSubmitting(false);
    }
  };

  const handleCancel = () => {
    setDisplayAssignedIds(persistedIds);
    setLeftSelected(new Set());
    setRightSelected(new Set());
  };

  return (
    <div className="workspace">
      <header className="toolbar">
        <div>
          <p className="eyebrow">Admin</p>
          <h2>模型分配</h2>
        </div>
      </header>

      {error ? <Notice type="error">{error}</Notice> : null}
      {message ? <Notice type="info">{message}</Notice> : null}

      <div className="user-selector">
        <label>当前用户：</label>
        <select value={selectedUserId} onChange={(e) => selectUser(e.target.value)}>
          {users.map((u) => (
            <option key={u.id} value={u.id}>
              {u.username} ({u.role})
            </option>
          ))}
        </select>
      </div>

      <div className="shuttle-box">
        <div className="shuttle-panel">
          <div className="shuttle-header">
            可分配模型
            <span className="count">{availableModels.length} 个</span>
          </div>
          <div className="shuttle-list">
            {availableModels.map((m) => (
              <label key={m.id} className="shuttle-item">
                <input
                  type="checkbox"
                  checked={leftSelected.has(m.id)}
                  onChange={() => toggleLeft(m.id)}
                />
                <div>
                  <div className="model-name">{m.displayName || m.modelName}</div>
                  <div className="model-provider">{providerNameById[m.providerId] || m.providerId}</div>
                </div>
              </label>
            ))}
            {availableModels.length === 0 && <div className="shuttle-empty">所有模型已分配</div>}
          </div>
        </div>

        <div className="shuttle-actions">
          <button onClick={assignModels} disabled={leftSelected.size === 0}>
            分配 &gt;
          </button>
          <button onClick={revokeModels} disabled={rightSelected.size === 0}>
            &lt; 回收
          </button>
        </div>

        <div className="shuttle-panel assigned">
          <div className="shuttle-header">
            已分配模型
            <span className="count">{assignedModels.length} 个</span>
          </div>
          <div className="shuttle-list">
            {assignedModels.map((m) => (
              <label key={m.id} className="shuttle-item">
                <input
                  type="checkbox"
                  checked={rightSelected.has(m.id)}
                  onChange={() => toggleRight(m.id)}
                />
                <div className="shuttle-item-info">
                  <div className="model-name">{m.displayName || m.modelName}</div>
                  <div className="model-provider">{providerNameById[m.providerId] || m.providerId}</div>
                </div>
              </label>
            ))}
            {assignedModels.length === 0 && <div className="shuttle-empty">尚未分配模型</div>}
          </div>
        </div>
      </div>

      <div className="shuttle-footer">
        <button onClick={handleCancel} disabled={!isDirty}>
          取消
        </button>
        <button onClick={handleSave} disabled={!isDirty || submitting} className="primary-btn">
          保存变更
        </button>
      </div>
    </div>
  );
}
