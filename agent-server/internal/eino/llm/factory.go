package llm

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/openai"
	agentv1 "k8s-ai-ops/proto/agent/v1"
)

// NewFromConfig 根据 proto 中的模型运行时配置创建 OpenAI 兼容 ChatModel。
func NewFromConfig(ctx context.Context, cfg *agentv1.ModelRuntimeConfig) (*openai.ChatModel, error) {
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: cfg.GetBaseUrl(),
		APIKey:  cfg.GetApiKey(),
		Model:   cfg.GetModelName(),
	})
}
