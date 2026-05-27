package app

import (
	"context"

	"k8s-ai-ops/backend/internal/domain"
	agentv1 "k8s-ai-ops/proto/agent/v1"
)

// AgentClient 是应用层调用 Agent Server 的端口。
type AgentClient interface {
	RunStream(context.Context, *agentv1.AgentRunRequest) (AgentStream, error)
}

// AgentStream 是 Agent Server 流式响应读取端口。
type AgentStream interface {
	Recv() (*agentv1.StreamEvent, error)
}

// SecretCipher 是应用层加解密敏感配置的端口。
type SecretCipher interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

// RBACApplier 是应用层同步 Kubernetes RBAC 的端口。
type RBACApplier interface {
	Apply(ctx context.Context, userID string, permissions []domain.Permission) error
}

// RBACApplierFunc 允许用函数适配 RBACApplier。
type RBACApplierFunc func(ctx context.Context, userID string, permissions []domain.Permission) error

func (f RBACApplierFunc) Apply(ctx context.Context, userID string, permissions []domain.Permission) error {
	return f(ctx, userID, permissions)
}
