export type Provider = {
  id: string;
  name: string;
  protocol: "openai" | "anthropic" | string;
  baseUrl: string;
  enabled: boolean;
  apiKeyConfigured: boolean;
};

export type CreateProviderRequest = {
  name: string;
  protocol: "openai" | "anthropic";
  baseUrl: string;
  apiKey: string;
  enabled: boolean;
};

export type Model = {
  id: string;
  providerId: string;
  modelName: string;
  displayName: string;
  supportsTools: boolean;
  supportsStreaming: boolean;
  enabled: boolean;
};

export type CreateModelRequest = {
  providerId: string;
  modelName: string;
  displayName: string;
  supportsTools: boolean;
  supportsStreaming: boolean;
  enabled: boolean;
};
