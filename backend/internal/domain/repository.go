package domain

import "context"

// ─── 仓储接口（领域层定义，基础设施层实现） ───

// UserRepository 管理 User 聚合的持久化。
type UserRepository interface {
	FindAll(ctx context.Context) ([]User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	Save(ctx context.Context, user *User) error // upsert 整个聚合
}

// PermissionRepository 管理 Permission 的持久化。
type PermissionRepository interface {
	FindByUser(ctx context.Context, userID string) ([]Permission, error)
	Replace(ctx context.Context, userID string, permissions []Permission) error
}

// LLMProviderRepository 管理 LLMProvider。
type LLMProviderRepository interface {
	FindAll(ctx context.Context) ([]LLMProvider, error)
	FindByID(ctx context.Context, id string) (*LLMProvider, error)
	Save(ctx context.Context, provider *LLMProvider) error
}

// LLMModelRepository 管理 LLMModel。
type LLMModelRepository interface {
	FindAll(ctx context.Context) ([]LLMModel, error)
	FindByID(ctx context.Context, id string) (*LLMModel, error)
	Save(ctx context.Context, model *LLMModel) error
}

// LLMBindingRepository 管理用户-LLM 模型绑定。
type LLMBindingRepository interface {
	FindByUser(ctx context.Context, userID string) ([]LLMBinding, error)
	Replace(ctx context.Context, userID string, bindings []LLMBinding) error
}

// ChatSessionRepository 管理 ChatSession 聚合。
type ChatSessionRepository interface {
	FindByID(ctx context.Context, id string) (*ChatSession, error)
	FindByUser(ctx context.Context, userID string) ([]ChatSession, error)
	Save(ctx context.Context, session *ChatSession) error
}

// ChatMessageRepository 管理 ChatMessage。
type ChatMessageRepository interface {
	Append(ctx context.Context, msg *ChatMessage) error
	FindBySession(ctx context.Context, sessionID string) ([]ChatMessage, error)
}

// ServiceAccountRepository 管理 K8s ServiceAccount 绑定和凭据。
type ServiceAccountRepository interface {
	FindToken(ctx context.Context, userID string) (*ServiceAccountToken, error)
	SaveToken(ctx context.Context, token *ServiceAccountToken) error
	SaveBinding(ctx context.Context, binding *ServiceAccountBinding) error
	FindBindings(ctx context.Context, userID string) ([]ServiceAccountBinding, error)
}

// AuditRepository 管理审计日志。
type AuditRepository interface {
	Append(ctx context.Context, log *AuditLog) error
	FindAll(ctx context.Context) ([]AuditLog, error)
}

// ─── 聚合仓储 ───

// Repositories 聚合所有领域仓储接口。
// 由基础设施层的 postgres.NewRepositories() 创建。
type Repositories struct {
	Users        UserRepository
	Permissions  PermissionRepository
	Providers    LLMProviderRepository
	Models       LLMModelRepository
	Bindings     LLMBindingRepository
	ChatSessions ChatSessionRepository
	ChatMessages ChatMessageRepository
	ServiceAccts ServiceAccountRepository
	Audit        AuditRepository
}
