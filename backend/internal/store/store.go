package store

type Store interface {
	// Demo user
	CurrentDemoUser() User

	// User management
	ListUsers() []User
	CreateUser(user User) User
	VerifyPassword(username, password string) (User, bool)

	// Permission management
	ListUserPermissions(userID string) []Permission
	ReplaceUserPermissions(userID string, permissions []Permission) []Permission

	// LLM providers
	ListLLMProviders() []LLMProvider
	CreateLLMProvider(provider LLMProvider) LLMProvider

	// LLM models
	ListLLMModels() []LLMModel
	CreateLLMModel(model LLMModel) LLMModel

	// LLM bindings
	SetUserLLMBindings(userID string, modelIDs []string, defaultModelID string) []UserLLMBinding
	GetUserLLMBindings(userID string) []UserLLMBinding

	// Chat sessions
	CreateChatSession(session ChatSession) ChatSession
	GetChatSession(id string) (ChatSession, bool)
	ListUserChatSessions(userID string) []ChatSession

	// Chat messages
	AppendChatMessage(msg ChatMessage) ChatMessage
	ListSessionMessages(sessionID string) []ChatMessage

	// ServiceAccount bindings
	SaveServiceAccountBinding(binding ServiceAccountBinding) ServiceAccountBinding
	GetUserServiceAccountBindings(userID string) []ServiceAccountBinding
	// GetServiceAccountToken returns the K8S SA token and namespace for a user.
	GetServiceAccountToken(userID string) (token, namespace string, err error)

	// Audit logs
	AppendAuditLog(log AuditLog) AuditLog
	ListAuditLogs() []AuditLog
}
