// Package app 是 Backend 的应用服务层，编排领域对象实现用例。
// HTTP Handler 只需绑定请求、调用应用服务、格式化响应。
package app

// ─── 请求 DTO ───

// CreateUserRequest 创建用户请求。
type CreateUserRequest struct {
	Username    string `json:"username"    binding:"required"`
	Password    string `json:"password"`
	Email       string `json:"email"`
	Role        string `json:"role"        binding:"required,oneof=admin operator"`
	DisplayName string `json:"displayName"`
}

// ResetPasswordRequest 重置密码请求。
type ResetPasswordRequest struct {
	Password string `json:"password" binding:"required"`
}

// UpdateModelBindingsRequest 更新用户模型绑定请求。
type UpdateModelBindingsRequest struct {
	ModelIDs []string `json:"modelIds" binding:"required"`
}

// UpdatePermissionsRequest 更新权限请求。
type UpdatePermissionsRequest struct {
	Permissions []PermissionItem `json:"permissions" binding:"required"`
}

// PermissionItem 单条权限项。
type PermissionItem struct {
	Namespace string   `json:"namespace" binding:"required"`
	APIGroup  string   `json:"apiGroup"`
	Resource  string   `json:"resource"  binding:"required"`
	Verbs     []string `json:"verbs"     binding:"required,min=1"`
}

// CreateProviderRequest 创建 LLM Provider 请求。
type CreateProviderRequest struct {
	Name     string `json:"name"     binding:"required"`
	Protocol string `json:"protocol" binding:"required,oneof=openai anthropic"`
	BaseURL  string `json:"baseUrl"  binding:"required"`
	APIKey   string `json:"apiKey"`
	Enabled  bool   `json:"enabled"`
}

// UpdateProviderRequest 更新 LLM Provider 请求。
type UpdateProviderRequest struct {
	Name    *string `json:"name"`
	BaseURL *string `json:"baseUrl"`
	APIKey  *string `json:"apiKey"`
	Enabled *bool   `json:"enabled"`
}

// CreateModelRequest 创建 LLM Model 请求。
type CreateModelRequest struct {
	ProviderID        string `json:"providerId"        binding:"required"`
	ModelName         string `json:"modelName"         binding:"required"`
	DisplayName       string `json:"displayName"`
	SupportsTools     bool   `json:"supportsTools"`
	SupportsStreaming bool   `json:"supportsStreaming"`
	Enabled           bool   `json:"enabled"`
}

// UpdateModelRequest 更新 LLM Model 请求。
type UpdateModelRequest struct {
	ProviderID        *string `json:"providerId"`
	DisplayName       *string `json:"displayName"`
	SupportsTools     *bool   `json:"supportsTools"`
	SupportsStreaming *bool   `json:"supportsStreaming"`
	Enabled           *bool   `json:"enabled"`
}

// ChatMessageRequest 发送 Chat 消息请求。
type ChatMessageRequest struct {
	ModelID string `json:"modelId" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// ─── 响应 DTO ───

// UserResponse 用户响应。
type UserResponse struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email,omitempty"`
	Role        string `json:"role"`
	Status      string `json:"status"`
}

// PermissionResponse 权限响应。
type PermissionResponse struct {
	ID        string   `json:"id"`
	Namespace string   `json:"namespace"`
	APIGroup  string   `json:"apiGroup"`
	Resource  string   `json:"resource"`
	Verbs     []string `json:"verbs"`
	Enabled   bool     `json:"enabled"`
}

// ProviderResponse LLM Provider 响应。
type ProviderResponse struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Protocol         string `json:"protocol"`
	BaseURL          string `json:"baseUrl"`
	Enabled          bool   `json:"enabled"`
	APIKeyConfigured bool   `json:"apiKeyConfigured"`
}

// ModelResponse LLM Model 响应。
type ModelResponse struct {
	ID                string `json:"id"`
	ProviderID        string `json:"providerId"`
	ModelName         string `json:"modelName"`
	DisplayName       string `json:"displayName"`
	SupportsTools     bool   `json:"supportsTools"`
	SupportsStreaming bool   `json:"supportsStreaming"`
	Enabled           bool   `json:"enabled"`
}

// ChatSessionResponse 会话响应。
type ChatSessionResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	Title     string `json:"title,omitempty"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

// ModelBindingMapResponse 所有用户的模型绑定映射。
type ModelBindingMapResponse map[string][]string

// AuditLogResponse 审计日志响应。
type AuditLogResponse struct {
	ID          string `json:"id"`
	ActorUserID string `json:"actorUserId"`
	Action      string `json:"action"`
	TargetType  string `json:"targetType"`
	TargetID    string `json:"targetId"`
	Namespace   string `json:"namespace,omitempty"`
	Resource    string `json:"resource,omitempty"`
	Verb        string `json:"verb,omitempty"`
	Allowed     bool   `json:"allowed"`
	Reason      string `json:"reason"`
	CreatedAt   string `json:"createdAt"`
}
