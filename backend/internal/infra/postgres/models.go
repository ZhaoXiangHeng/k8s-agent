package postgres

import "time"

// allModels 是 AutoMigrate 注册的全部 GORM 持久化模型。
var allModels = []any{
	&userModel{}, &permModel{}, &providerModel{}, &modelModel{},
	&bindingModel{}, &sessionModel{}, &messageModel{}, &auditModel{},
	&serviceAccountTokenModel{}, &serviceAccountBindingModel{},
}

// ─── GORM 持久化模型（与领域模型分离，仅承载数据库映射） ───

type userModel struct {
	ID          string `gorm:"primaryKey;column:id"`
	Username    string `gorm:"column:username;uniqueIndex"`
	DisplayName string `gorm:"column:display_name"`
	Email       string `gorm:"column:email"`
	Role        string `gorm:"column:role"`
	Status      string `gorm:"column:status"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (userModel) TableName() string { return "users" }

type permModel struct {
	ID        string   `gorm:"primaryKey"`
	UserID    string   `gorm:"column:user_id"`
	Namespace string   `gorm:"column:namespace"`
	APIGroup  string   `gorm:"column:api_group"`
	Resource  string   `gorm:"column:resource"`
	Verbs     []string `gorm:"type:jsonb;column:verbs_json;serializer:json"`
	Enabled   bool     `gorm:"column:enabled"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (permModel) TableName() string { return "k8s_permissions" }

type providerModel struct {
	ID               string `gorm:"primaryKey"`
	Name             string `gorm:"column:name"`
	Protocol         string `gorm:"column:protocol"`
	BaseURL          string `gorm:"column:base_url"`
	APIKeyCiphertext string `gorm:"column:api_key_ciphertext"`
	Enabled          bool   `gorm:"column:enabled"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (providerModel) TableName() string { return "llm_providers" }

type modelModel struct {
	ID                string `gorm:"primaryKey"`
	ProviderID        string `gorm:"column:provider_id"`
	ModelName         string `gorm:"column:model_name"`
	DisplayName       string `gorm:"column:display_name"`
	SupportsTools     bool   `gorm:"column:supports_tools"`
	SupportsStreaming bool   `gorm:"column:supports_streaming"`
	Enabled           bool   `gorm:"column:enabled"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (modelModel) TableName() string { return "llm_models" }

type bindingModel struct {
	ID        string `gorm:"primaryKey"`
	UserID    string `gorm:"column:user_id"`
	ModelID   string `gorm:"column:model_id"`
	IsDefault bool   `gorm:"column:is_default"`
	CreatedAt time.Time
}

func (bindingModel) TableName() string { return "user_llm_bindings" }

type sessionModel struct {
	ID        string `gorm:"primaryKey"`
	UserID    string `gorm:"column:user_id"`
	ModelID   string `gorm:"column:model_id"`
	Title     string `gorm:"column:title"`
	Status    string `gorm:"column:status"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (sessionModel) TableName() string { return "chat_sessions" }

type messageModel struct {
	ID             string `gorm:"primaryKey"`
	SessionID      string `gorm:"column:session_id"`
	Role           string `gorm:"column:role"`
	Content        string `gorm:"column:content"`
	ToolName       string `gorm:"column:tool_name"`
	ToolArgsJSON   string `gorm:"column:tool_args_json"`
	ToolResultJSON string `gorm:"column:tool_result_json"`
	CreatedAt      time.Time
}

func (messageModel) TableName() string { return "chat_messages" }

type auditModel struct {
	ID          string `gorm:"primaryKey"`
	ActorUserID string `gorm:"column:actor_user_id"`
	Action      string `gorm:"column:action"`
	TargetType  string `gorm:"column:target_type"`
	TargetID    string `gorm:"column:target_id"`
	Namespace   string `gorm:"column:namespace"`
	Resource    string `gorm:"column:resource"`
	Verb        string `gorm:"column:verb"`
	Allowed     bool   `gorm:"column:allowed"`
	Reason      string `gorm:"column:reason"`
	CreatedAt   time.Time
}

func (auditModel) TableName() string { return "audit_logs" }

type serviceAccountTokenModel struct {
	UserID          string `gorm:"primaryKey;column:user_id"`
	ServiceAccount  string `gorm:"column:service_account"`
	Namespace       string `gorm:"column:namespace"`
	TokenCiphertext string `gorm:"column:token_ciphertext"`
	CACert          string `gorm:"column:ca_cert"`
	APIServer       string `gorm:"column:api_server"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (serviceAccountTokenModel) TableName() string { return "service_account_tokens" }

type serviceAccountBindingModel struct {
	ID             string `gorm:"primaryKey"`
	UserID         string `gorm:"column:user_id;index"`
	Namespace      string `gorm:"column:namespace"`
	ServiceAccount string `gorm:"column:service_account"`
	Status         string `gorm:"column:status"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (serviceAccountBindingModel) TableName() string { return "service_account_bindings" }
