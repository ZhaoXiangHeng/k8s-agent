import { useState } from "react";
import type { CreateModelRequest, CreateProviderRequest, Model, Provider } from "../domain/llm";
import { AdminModelsPage } from "./AdminModelsPage";
import { AdminProvidersPage } from "./AdminProvidersPage";

export function AdminLlmConfigPage({
  models,
  providers,
  onCreateProvider,
  onUpdateProvider,
  onCreateModel,
  onUpdateModel,
  onDeleteModel
}: {
  models: Model[];
  providers: Provider[];
  onCreateProvider: (body: CreateProviderRequest) => Promise<void>;
  onUpdateProvider: (id: string, body: Partial<CreateProviderRequest>) => Promise<void>;
  onCreateModel: (body: CreateModelRequest) => Promise<void>;
  onUpdateModel: (id: string, body: Partial<CreateModelRequest>) => Promise<void>;
  onDeleteModel: (id: string) => Promise<void>;
}) {
  const [tab, setTab] = useState<"providers" | "models">("providers");

  return (
    <div className="workspace">
      <header className="toolbar">
        <div>
          <p className="eyebrow">Platform</p>
          <h2>LLM 配置</h2>
        </div>
        <div className="tabs">
          <button className={tab === "providers" ? "activeTab" : ""} onClick={() => setTab("providers")}>Provider</button>
          <button className={tab === "models" ? "activeTab" : ""} onClick={() => setTab("models")}>模型</button>
        </div>
      </header>
      {tab === "providers" ? (
        <AdminProvidersPage providers={providers} onCreate={onCreateProvider} onUpdate={onUpdateProvider} embedded />
      ) : (
        <AdminModelsPage models={models} providers={providers} onCreate={onCreateModel} onUpdate={onUpdateModel} onDelete={onDeleteModel} embedded />
      )}
    </div>
  );
}
