package store

import (
	"fmt"
	"strings"
	"time"
)

type ServiceAccountToken struct {
	UserID    string
	SAName    string
	Token     string
	Namespace string
	CACert    string
	APIServer string
}

type MemoryStore struct {
	users       []User
	permissions []Permission
	providers   []LLMProvider
	models      []LLMModel
	auditLogs   []AuditLog

	chatSessions []ChatSession
	chatMessages []ChatMessage
	llmBindings  []UserLLMBinding
	saBindings   []ServiceAccountBinding
	saTokens     map[string]ServiceAccountToken // userID -> token info
}

func NewMemoryStore() *MemoryStore {
	demoUser := User{
		ID:             "demo-user",
		KeycloakUserID: "keycloak-demo-user",
		Username:       "demo",
		DisplayName:    "Demo Operator",
		Email:          "operator@example.com",
		Role:           RoleOperator,
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	return &MemoryStore{
		users: []User{demoUser},
		permissions: []Permission{
			{
				ID:        "perm-dev-pods-read",
				UserID:    demoUser.ID,
				Namespace: "dev",
				APIGroup:  "",
				Resource:  "pods",
				Verbs:     []string{"get", "list", "watch"},
				Enabled:   true,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
			{
				ID:        "perm-dev-pod-logs-read",
				UserID:    demoUser.ID,
				Namespace: "dev",
				APIGroup:  "",
				Resource:  "pods/log",
				Verbs:     []string{"get"},
				Enabled:   true,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
		},
		chatSessions: []ChatSession{},
		chatMessages: []ChatMessage{},
		llmBindings:  []UserLLMBinding{},
		saBindings:   []ServiceAccountBinding{},
		saTokens:     map[string]ServiceAccountToken{},
	}
}

func (s *MemoryStore) CurrentDemoUser() User {
	return s.users[0]
}

func (s *MemoryStore) ListUsers() []User {
	return append([]User(nil), s.users...)
}

func (s *MemoryStore) CreateUser(user User) User {
	now := time.Now().UTC()
	user.ID = "user-" + user.Username
	user.Status = defaultString(user.Status, "active")
	user.CreatedAt = now
	user.UpdatedAt = now
	s.users = append(s.users, user)
	return user
}

func (s *MemoryStore) ListUserPermissions(userID string) []Permission {
	var result []Permission
	for _, permission := range s.permissions {
		if permission.UserID == userID {
			result = append(result, permission)
		}
	}
	return result
}

func (s *MemoryStore) ReplaceUserPermissions(userID string, permissions []Permission) []Permission {
	var kept []Permission
	for _, permission := range s.permissions {
		if permission.UserID != userID {
			kept = append(kept, permission)
		}
	}
	now := time.Now().UTC()
	for index := range permissions {
		permissions[index].ID = userID + "-perm-" + permissions[index].Namespace + "-" + strings.ReplaceAll(permissions[index].Resource, "/", "-")
		permissions[index].UserID = userID
		permissions[index].Enabled = true
		permissions[index].CreatedAt = now
		permissions[index].UpdatedAt = now
	}
	s.permissions = append(kept, permissions...)
	return append([]Permission(nil), permissions...)
}

func (s *MemoryStore) ListLLMProviders() []LLMProvider {
	return append([]LLMProvider(nil), s.providers...)
}

func (s *MemoryStore) CreateLLMProvider(provider LLMProvider) LLMProvider {
	now := time.Now().UTC()
	provider.ID = "provider-" + provider.Protocol
	provider.APIKeyCiphertext = "encrypted:" + provider.Name
	provider.CreatedAt = now
	provider.UpdatedAt = now
	s.providers = append(s.providers, provider)
	return provider
}

func (s *MemoryStore) ListLLMModels() []LLMModel {
	return append([]LLMModel(nil), s.models...)
}

func (s *MemoryStore) CreateLLMModel(model LLMModel) LLMModel {
	now := time.Now().UTC()
	model.ID = "model-" + model.ModelName
	model.CreatedAt = now
	model.UpdatedAt = now
	s.models = append(s.models, model)
	return model
}

func (s *MemoryStore) AppendAuditLog(log AuditLog) AuditLog {
	log.ID = "audit-" + time.Now().UTC().Format("20060102150405.000000000")
	log.CreatedAt = time.Now().UTC()
	s.auditLogs = append(s.auditLogs, log)
	return log
}

func (s *MemoryStore) ListAuditLogs() []AuditLog {
	return append([]AuditLog(nil), s.auditLogs...)
}

func (s *MemoryStore) CreateChatSession(session ChatSession) ChatSession {
	now := time.Now().UTC()
	session.ID = session.ID
	if session.ID == "" {
		session.ID = "session-" + now.Format("20060102150405.000000000")
	}
	session.CreatedAt = now
	session.UpdatedAt = now
	if session.Status == "" {
		session.Status = "active"
	}
	s.chatSessions = append(s.chatSessions, session)
	return session
}

func (s *MemoryStore) GetChatSession(id string) (ChatSession, bool) {
	for _, session := range s.chatSessions {
		if session.ID == id {
			return session, true
		}
	}
	return ChatSession{}, false
}

func (s *MemoryStore) ListUserChatSessions(userID string) []ChatSession {
	var result []ChatSession
	for _, session := range s.chatSessions {
		if session.UserID == userID {
			result = append(result, session)
		}
	}
	return result
}

func (s *MemoryStore) AppendChatMessage(msg ChatMessage) ChatMessage {
	msg.ID = msg.ID
	if msg.ID == "" {
		msg.ID = "msg-" + time.Now().UTC().Format("20060102150405.000000000")
	}
	msg.CreatedAt = time.Now().UTC()
	s.chatMessages = append(s.chatMessages, msg)
	return msg
}

func (s *MemoryStore) ListSessionMessages(sessionID string) []ChatMessage {
	var result []ChatMessage
	for _, msg := range s.chatMessages {
		if msg.SessionID == sessionID {
			result = append(result, msg)
		}
	}
	return result
}

func (s *MemoryStore) SetUserLLMBindings(userID string, modelIDs []string, defaultModelID string) []UserLLMBinding {
	var kept []UserLLMBinding
	for _, b := range s.llmBindings {
		if b.UserID != userID {
			kept = append(kept, b)
		}
	}
	now := time.Now().UTC()
	for _, modelID := range modelIDs {
		kept = append(kept, UserLLMBinding{
			ID:        userID + "-" + modelID,
			UserID:    userID,
			ModelID:   modelID,
			IsDefault: modelID == defaultModelID,
			CreatedBy: userID,
			CreatedAt: now,
		})
	}
	s.llmBindings = kept
	return s.GetUserLLMBindings(userID)
}

func (s *MemoryStore) GetUserLLMBindings(userID string) []UserLLMBinding {
	var result []UserLLMBinding
	for _, b := range s.llmBindings {
		if b.UserID == userID {
			result = append(result, b)
		}
	}
	return result
}

func (s *MemoryStore) SaveServiceAccountBinding(binding ServiceAccountBinding) ServiceAccountBinding {
	now := time.Now().UTC()
	binding.ID = binding.ID
	if binding.ID == "" {
		binding.ID = "sab-" + binding.UserID + "-" + binding.Namespace
	}
	binding.CreatedAt = now
	binding.UpdatedAt = now
	if binding.Status == "" {
		binding.Status = "active"
	}
	for i, b := range s.saBindings {
		if b.UserID == binding.UserID && b.Namespace == binding.Namespace {
			s.saBindings[i] = binding
			return binding
		}
	}
	s.saBindings = append(s.saBindings, binding)
	return binding
}

func (s *MemoryStore) GetUserServiceAccountBindings(userID string) []ServiceAccountBinding {
	var result []ServiceAccountBinding
	for _, b := range s.saBindings {
		if b.UserID == userID {
			result = append(result, b)
		}
	}
	return result
}

func (s *MemoryStore) GetServiceAccountToken(userID string) (string, string, error) {
	token, ok := s.saTokens[userID]
	if !ok {
		return "", "", fmt.Errorf("no service account token for user %s", userID)
	}
	return token.Token, token.Namespace, nil
}

func (s *MemoryStore) GetServiceAccount(userID string) (*ServiceAccountToken, error) {
	token, ok := s.saTokens[userID]
	if !ok {
		return nil, fmt.Errorf("no service account token for user %s", userID)
	}
	return &token, nil
}

func (s *MemoryStore) VerifyPassword(username, password string) (User, bool) {
	if username == "admin" && password == "123456" {
		return User{ID: "admin-user", Username: "admin", Role: RoleAdmin, Status: "active"}, true
	}
	for _, u := range s.users {
		if u.Username == username {
			return u, true
		}
	}
	return User{}, false
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
