package llm

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/openai"
	agentv1 "k8s-ai-ops/proto/agent/v1"
)

// NewFromConfig creates an OpenAI-compatible ChatModel from the proto ModelRuntimeConfig.
func NewFromConfig(ctx context.Context, cfg *agentv1.ModelRuntimeConfig) (*openai.ChatModel, error) {
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: cfg.GetBaseUrl(),
		APIKey:  cfg.GetApiKey(),
		Model:   cfg.GetModelName(),
	})
}
