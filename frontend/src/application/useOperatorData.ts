import { useCallback, useEffect, useState } from "react";
import type { Model } from "../domain/llm";
import type { Permission } from "../domain/permission";
import type { ApiAuth } from "../infrastructure/api/client";
import { listOperatorModels } from "../infrastructure/api/llmApi";
import { listOperatorPermissions } from "../infrastructure/api/permissionApi";
import { appConfig } from "../config";
import { mockModels, mockPermissions } from "./mockData";

export function useOperatorData(auth: ApiAuth, enabled = true) {
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [models, setModels] = useState<Model[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const reload = useCallback(async () => {
    if (!enabled) return;
    setLoading(true);
    setError("");
    try {
      if (appConfig.dataMode === "mock") {
        setPermissions(mockPermissions);
        setModels(mockModels.filter((model) => model.enabled));
        return;
      }
      const [nextPermissions, nextModels] = await Promise.all([
        listOperatorPermissions(auth),
        listOperatorModels(auth)
      ]);
      setPermissions(nextPermissions);
      setModels(nextModels);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载操作员数据失败");
    } finally {
      setLoading(false);
    }
  }, [auth, enabled]);

  useEffect(() => {
    if (enabled) void reload();
  }, [enabled, reload]);

  return { permissions, models, loading, error, reload };
}
