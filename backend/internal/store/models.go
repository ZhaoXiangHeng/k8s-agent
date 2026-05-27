package store

import "time"

type UserRole string

const (
	RoleAdmin    UserRole = "admin"
	RoleOperator UserRole = "operator"
)

type User struct {
	ID             string    `gorm:"primaryKey;column:id"`
	KeycloakUserID string    `gorm:"column:keycloak_user_id"`
	Username       string    `gorm:"column:username;uniqueIndex"`
	DisplayName    string    `gorm:"column:display_name"`
	Email          string    `gorm:"column:email"`
	Role           UserRole  `gorm:"column:role"`
	Status         string    `gorm:"column:status"`
	CreatedBy      string    `gorm:"column:created_by"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (User) TableName() string { return "users" }

type ServiceAccountBinding struct {
	ID                 string    `gorm:"primaryKey;column:id"`
	UserID             string    `gorm:"column:user_id"`
	Namespace          string    `gorm:"column:namespace"`
	ServiceAccountName string    `gorm:"column:service_account_name"`
	TokenSecretName    string    `gorm:"column:token_secret_name"`
	Status             string    `gorm:"column:status"`
	CreatedAt          time.Time `gorm:"column:created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at"`
}

func (ServiceAccountBinding) TableName() string { return "k8s_service_accounts" }

type Permission struct {
	ID              string    `gorm:"primaryKey;column:id"`
	UserID          string    `gorm:"column:user_id"`
	Namespace       string    `gorm:"column:namespace"`
	APIGroup        string    `gorm:"column:api_group"`
	Resource        string    `gorm:"column:resource"`
	Verbs           []string  `gorm:"type:jsonb;column:verbs_json;serializer:json"`
	RoleName        string    `gorm:"column:role_name"`
	RoleBindingName string    `gorm:"column:role_binding_name"`
	Enabled         bool      `gorm:"column:enabled"`
	CreatedBy       string    `gorm:"column:created_by"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func (Permission) TableName() string { return "k8s_permissions" }

func (p Permission) Allows(namespace, apiGroup, resource, verb string) bool {
	if !p.Enabled {
		return false
	}
	if p.Namespace != namespace || p.APIGroup != apiGroup || p.Resource != resource {
		return false
	}
	for _, allowedVerb := range p.Verbs {
		if allowedVerb == verb {
			return true
		}
	}
	return false
}

type LLMProvider struct {
	ID               string    `gorm:"primaryKey;column:id"`
	Name             string    `gorm:"column:name"`
	Protocol         string    `gorm:"column:protocol"`
	BaseURL          string    `gorm:"column:base_url"`
	APIKeyCiphertext string    `gorm:"column:api_key_ciphertext"`
	Enabled          bool      `gorm:"column:enabled"`
	CreatedBy        string    `gorm:"column:created_by"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
}

func (LLMProvider) TableName() string { return "llm_providers" }

type LLMModel struct {
	ID                string    `gorm:"primaryKey;column:id"`
	ProviderID        string    `gorm:"column:provider_id"`
	ModelName         string    `gorm:"column:model_name"`
	DisplayName       string    `gorm:"column:display_name"`
	SupportsTools     bool      `gorm:"column:supports_tools"`
	SupportsStreaming bool      `gorm:"column:supports_streaming"`
	Enabled           bool      `gorm:"column:enabled"`
	CreatedAt         time.Time `gorm:"column:created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at"`
}

func (LLMModel) TableName() string { return "llm_models" }

type UserLLMBinding struct {
	ID        string    `gorm:"primaryKey;column:id"`
	UserID    string    `gorm:"column:user_id"`
	ModelID   string    `gorm:"column:model_id"`
	IsDefault bool      `gorm:"column:is_default"`
	CreatedBy string    `gorm:"column:created_by"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (UserLLMBinding) TableName() string { return "user_llm_bindings" }

type ChatSession struct {
	ID        string    `gorm:"primaryKey;column:id"`
	UserID    string    `gorm:"column:user_id"`
	ModelID   string    `gorm:"column:model_id"`
	Title     string    `gorm:"column:title"`
	Status    string    `gorm:"column:status"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (ChatSession) TableName() string { return "chat_sessions" }

type ChatMessage struct {
	ID             string    `gorm:"primaryKey;column:id"`
	SessionID      string    `gorm:"column:session_id"`
	Role           string    `gorm:"column:role"`
	Content        string    `gorm:"column:content"`
	ToolName       string    `gorm:"column:tool_name"`
	ToolArgsJSON   string    `gorm:"column:tool_args_json"`
	ToolResultJSON string    `gorm:"column:tool_result_json"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (ChatMessage) TableName() string { return "chat_messages" }

type AuditLog struct {
	ID           string    `gorm:"primaryKey;column:id"`
	ActorUserID  string    `gorm:"column:actor_user_id"`
	Action       string    `gorm:"column:action"`
	TargetType   string    `gorm:"column:target_type"`
	TargetID     string    `gorm:"column:target_id"`
	Namespace    string    `gorm:"column:namespace"`
	Resource     string    `gorm:"column:resource"`
	Verb         string    `gorm:"column:verb"`
	Allowed      bool      `gorm:"column:allowed"`
	Reason       string    `gorm:"column:reason"`
	RequestJSON  string    `gorm:"column:request_json"`
	ResponseJSON string    `gorm:"column:response_json"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }
