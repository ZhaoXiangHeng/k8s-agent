import { useState, useMemo, useEffect } from "react";
import {
  listUsers,
  getAssignedModels,
  getAvailableModels,
  getDefaultModel,
  updateBindings,
} from "../store/auth";
import type { Model } from "../store/auth";

export default function ModelAssignment({ preselectedUserId }: { preselectedUserId?: string }) {
  const users = listUsers();
  const [selectedUserId, setSelectedUserId] = useState<string>(preselectedUserId ?? (users[0]?.id || ""));

  // Sync preselected user from parent navigation
  useEffect(() => {
    if (preselectedUserId) {
      selectUser(preselectedUserId);
    }
  }, [preselectedUserId]);

  const assignedModels = useMemo(() => getAssignedModels(selectedUserId), [selectedUserId]);
  const availableModels = useMemo(() => getAvailableModels(selectedUserId), [selectedUserId]);

  const [leftSelected, setLeftSelected] = useState<Set<string>>(new Set());
  const [rightSelected, setRightSelected] = useState<Set<string>>(new Set());

  // Track dirty changes
  const [dirtyAdded, setDirtyAdded] = useState<string[]>([]);
  const [dirtyRemoved, setDirtyRemoved] = useState<string[]>([]);
  const [displayAssigned, setDisplayAssigned] = useState<Model[]>(assignedModels);
  const [displayAvailable, setDisplayAvailable] = useState<Model[]>(availableModels);
  const [pendingDefault, setPendingDefault] = useState<string | undefined>(getDefaultModel(selectedUserId));

  // Reset when user changes
  const selectUser = (userId: string) => {
    setSelectedUserId(userId);
    setDisplayAssigned(getAssignedModels(userId));
    setDisplayAvailable(getAvailableModels(userId));
    setPendingDefault(getDefaultModel(userId));
    setDirtyAdded([]);
    setDirtyRemoved([]);
    setLeftSelected(new Set());
    setRightSelected(new Set());
  };

  const isDirty = dirtyAdded.length > 0 || dirtyRemoved.length > 0 || pendingDefault !== getDefaultModel(selectedUserId);

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
    const toMove = displayAvailable.filter((m) => ids.includes(m.id));
    setDisplayAvailable(displayAvailable.filter((m) => !ids.includes(m.id)));
    setDisplayAssigned([...displayAssigned, ...toMove]);
    setDirtyAdded([...dirtyAdded, ...ids]);
    setDirtyRemoved(dirtyRemoved.filter((id) => !ids.includes(id)));
    setLeftSelected(new Set());
  };

  const revokeModels = () => {
    const ids = Array.from(rightSelected);
    const toMove = displayAssigned.filter((m) => ids.includes(m.id));
    setDisplayAssigned(displayAssigned.filter((m) => !ids.includes(m.id)));
    setDisplayAvailable([...displayAvailable, ...toMove]);
    setDirtyRemoved([...dirtyRemoved, ...ids]);
    setDirtyAdded(dirtyAdded.filter((id) => !ids.includes(id)));
    if (pendingDefault && ids.includes(pendingDefault)) {
      setPendingDefault(undefined);
    }
    setRightSelected(new Set());
  };

  const handleSave = () => {
    updateBindings(selectedUserId, dirtyAdded, dirtyRemoved, pendingDefault);
    setDirtyAdded([]);
    setDirtyRemoved([]);
    // Re-read to sync
    setDisplayAssigned(getAssignedModels(selectedUserId));
    setDisplayAvailable(getAvailableModels(selectedUserId));
  };

  const handleCancel = () => {
    setDisplayAssigned(getAssignedModels(selectedUserId));
    setDisplayAvailable(getAvailableModels(selectedUserId));
    setPendingDefault(getDefaultModel(selectedUserId));
    setDirtyAdded([]);
    setDirtyRemoved([]);
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

      {/* User selector */}
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

      {/* Shuttle box */}
      <div className="shuttle-box">
        {/* Left: Available */}
        <div className="shuttle-panel">
          <div className="shuttle-header">
            可分配模型
            <span className="count">{displayAvailable.length} 个</span>
          </div>
          <div className="shuttle-list">
            {displayAvailable.map((m) => (
              <label key={m.id} className="shuttle-item">
                <input
                  type="checkbox"
                  checked={leftSelected.has(m.id)}
                  onChange={() => toggleLeft(m.id)}
                />
                <div>
                  <div className="model-name">{m.name}</div>
                  <div className="model-provider">{m.provider}</div>
                </div>
              </label>
            ))}
            {displayAvailable.length === 0 && (
              <div className="shuttle-empty">所有模型已分配</div>
            )}
          </div>
        </div>

        {/* Middle buttons */}
        <div className="shuttle-actions">
          <button onClick={assignModels} disabled={leftSelected.size === 0}>
            分配 &gt;
          </button>
          <button onClick={revokeModels} disabled={rightSelected.size === 0}>
            &lt; 回收
          </button>
        </div>

        {/* Right: Assigned */}
        <div className="shuttle-panel assigned">
          <div className="shuttle-header">
            已分配模型
            <span className="count">{displayAssigned.length} 个</span>
          </div>
          <div className="shuttle-list">
            {displayAssigned.map((m) => (
              <label key={m.id} className="shuttle-item">
                <input
                  type="checkbox"
                  checked={rightSelected.has(m.id)}
                  onChange={() => toggleRight(m.id)}
                />
                <div className="shuttle-item-info">
                  <div className="model-name">{m.name}</div>
                  <div className="model-provider">{m.provider}</div>
                </div>
                {pendingDefault === m.id ? (
                  <span className="default-badge">默认</span>
                ) : (
                  <button
                    className="link-btn"
                    onClick={(e) => {
                      e.preventDefault();
                      setPendingDefault(m.id);
                    }}
                  >
                    设为默认
                  </button>
                )}
              </label>
            ))}
            {displayAssigned.length === 0 && (
              <div className="shuttle-empty">尚未分配模型</div>
            )}
          </div>
        </div>
      </div>

      {/* Bottom save/cancel */}
      <div className="shuttle-footer">
        <button onClick={handleCancel} disabled={!isDirty}>
          取消
        </button>
        <button onClick={handleSave} disabled={!isDirty} className="primary-btn">
          保存变更
        </button>
      </div>
    </div>
  );
}
