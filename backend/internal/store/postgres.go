package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostgresStore struct {
	db       *gorm.DB
	saTokens map[string]ServiceAccountToken // userID -> token info
}

func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	store := &PostgresStore{db: db, saTokens: map[string]ServiceAccountToken{}}
	if err := store.InitSchema(); err != nil {
		return nil, err
	}
	if err := store.SeedDemoData(ctx); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *PostgresStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *PostgresStore) InitSchema() error {
	return s.db.AutoMigrate(
		&User{},
		&Permission{},
		&LLMProvider{},
		&LLMModel{},
		&UserLLMBinding{},
		&ServiceAccountBinding{},
		&ChatSession{},
		&ChatMessage{},
		&AuditLog{},
	)
}

func (s *PostgresStore) SeedDemoData(ctx context.Context) error {
	now := time.Now().UTC()
	demoUser := User{
		ID:             "demo-user",
		KeycloakUserID: "keycloak-demo-user",
		Username:       "demo",
		DisplayName:    "Demo Operator",
		Email:          "operator@example.com",
		Role:           RoleOperator,
		Status:         "active",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	result := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&demoUser)
	if result.Error != nil {
		return result.Error
	}

	var count int64
	s.db.WithContext(ctx).Model(&Permission{}).Where("user_id = ?", "demo-user").Count(&count)
	if count == 0 {
		s.ReplaceUserPermissions("demo-user", []Permission{
			{Namespace: "dev", APIGroup: "", Resource: "pods", Verbs: []string{"get", "list", "watch"}, Enabled: true},
			{Namespace: "dev", APIGroup: "", Resource: "pods/log", Verbs: []string{"get"}, Enabled: true},
		})
	}
	return nil
}

func (s *PostgresStore) CurrentDemoUser() User {
	var user User
	result := s.db.Where("id = ?", "demo-user").First(&user)
	if result.Error != nil {
		return NewMemoryStore().CurrentDemoUser()
	}
	return user
}

func (s *PostgresStore) ListUsers() []User {
	var users []User
	s.db.Order("created_at, id").Find(&users)
	return users
}

func (s *PostgresStore) CreateUser(user User) User {
	now := time.Now().UTC()
	user.ID = defaultString(user.ID, "user-"+user.Username)
	user.Status = defaultString(user.Status, "active")
	user.CreatedAt = now
	user.UpdatedAt = now
	s.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&user)
	return user
}

func (s *PostgresStore) VerifyPassword(username, password string) (User, bool) {
	if username == "admin" && password == "123456" {
		return User{ID: "admin-user", Username: "admin", Role: RoleAdmin, Status: "active"}, true
	}
	var user User
	result := s.db.Where("username = ?", username).First(&user)
	if result.Error != nil {
		return User{}, false
	}
	return user, true
}

func (s *PostgresStore) ListUserPermissions(userID string) []Permission {
	var permissions []Permission
	s.db.Where("user_id = ?", userID).Order("namespace, resource").Find(&permissions)
	return permissions
}

func (s *PostgresStore) ReplaceUserPermissions(userID string, permissions []Permission) []Permission {
	s.db.Where("user_id = ?", userID).Delete(&Permission{})
	now := time.Now().UTC()
	for i := range permissions {
		permissions[i].ID = userID + "-perm-" + permissions[i].Namespace + "-" + strings.ReplaceAll(permissions[i].Resource, "/", "-")
		permissions[i].UserID = userID
		permissions[i].Enabled = true
		permissions[i].CreatedAt = now
		permissions[i].UpdatedAt = now
	}
	if len(permissions) > 0 {
		s.db.Create(&permissions)
	}
	return permissions
}

func (s *PostgresStore) ListLLMProviders() []LLMProvider {
	var providers []LLMProvider
	s.db.Order("created_at, id").Find(&providers)
	return providers
}

func (s *PostgresStore) CreateLLMProvider(provider LLMProvider) LLMProvider {
	now := time.Now().UTC()
	provider.ID = defaultString(provider.ID, "provider-"+provider.Protocol)
	provider.APIKeyCiphertext = "encrypted:" + provider.Name
	provider.CreatedAt = now
	provider.UpdatedAt = now
	s.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&provider)
	return provider
}

func (s *PostgresStore) ListLLMModels() []LLMModel {
	var models []LLMModel
	s.db.Order("created_at, id").Find(&models)
	return models
}

func (s *PostgresStore) CreateLLMModel(model LLMModel) LLMModel {
	now := time.Now().UTC()
	model.ID = defaultString(model.ID, "model-"+model.ModelName)
	model.CreatedAt = now
	model.UpdatedAt = now
	s.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&model)
	return model
}

func (s *PostgresStore) SetUserLLMBindings(userID string, modelIDs []string, defaultModelID string) []UserLLMBinding {
	s.db.Where("user_id = ?", userID).Delete(&UserLLMBinding{})
	now := time.Now().UTC()
	var bindings []UserLLMBinding
	for _, modelID := range modelIDs {
		bindings = append(bindings, UserLLMBinding{
			ID:        userID + "-" + modelID,
			UserID:    userID,
			ModelID:   modelID,
			IsDefault: modelID == defaultModelID,
			CreatedBy: userID,
			CreatedAt: now,
		})
	}
	if len(bindings) > 0 {
		s.db.Create(&bindings)
	}
	return s.GetUserLLMBindings(userID)
}

func (s *PostgresStore) GetUserLLMBindings(userID string) []UserLLMBinding {
	var bindings []UserLLMBinding
	s.db.Where("user_id = ?", userID).Find(&bindings)
	return bindings
}

func (s *PostgresStore) CreateChatSession(session ChatSession) ChatSession {
	now := time.Now().UTC()
	if session.ID == "" {
		session.ID = "session-" + now.Format("20060102150405.000000000")
	}
	if session.Status == "" {
		session.Status = "active"
	}
	session.CreatedAt = now
	session.UpdatedAt = now
	s.db.Create(&session)
	return session
}

func (s *PostgresStore) GetChatSession(id string) (ChatSession, bool) {
	var session ChatSession
	result := s.db.Where("id = ?", id).First(&session)
	if result.Error != nil {
		return ChatSession{}, false
	}
	return session, true
}

func (s *PostgresStore) ListUserChatSessions(userID string) []ChatSession {
	var sessions []ChatSession
	s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&sessions)
	return sessions
}

func (s *PostgresStore) AppendChatMessage(msg ChatMessage) ChatMessage {
	if msg.ID == "" {
		msg.ID = "msg-" + time.Now().UTC().Format("20060102150405.000000000")
	}
	msg.CreatedAt = time.Now().UTC()
	s.db.Create(&msg)
	return msg
}

func (s *PostgresStore) ListSessionMessages(sessionID string) []ChatMessage {
	var messages []ChatMessage
	s.db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&messages)
	return messages
}

func (s *PostgresStore) SaveServiceAccountBinding(binding ServiceAccountBinding) ServiceAccountBinding {
	now := time.Now().UTC()
	if binding.ID == "" {
		binding.ID = "sab-" + binding.UserID + "-" + binding.Namespace
	}
	binding.CreatedAt = now
	binding.UpdatedAt = now
	if binding.Status == "" {
		binding.Status = "active"
	}
	s.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&binding)
	return binding
}

func (s *PostgresStore) GetUserServiceAccountBindings(userID string) []ServiceAccountBinding {
	var bindings []ServiceAccountBinding
	s.db.Where("user_id = ?", userID).Find(&bindings)
	return bindings
}

func (s *PostgresStore) GetServiceAccountToken(userID string) (string, string, error) {
	token, ok := s.saTokens[userID]
	if !ok {
		return "", "", fmt.Errorf("no service account token for user %s", userID)
	}
	return token.Token, token.Namespace, nil
}

func (s *PostgresStore) GetServiceAccount(userID string) (*ServiceAccountToken, error) {
	token, ok := s.saTokens[userID]
	if !ok {
		return nil, fmt.Errorf("no service account token for user %s", userID)
	}
	return &token, nil
}

func (s *PostgresStore) AppendAuditLog(log AuditLog) AuditLog {
	log.ID = defaultString(log.ID, "audit-"+time.Now().UTC().Format("20060102150405.000000000"))
	log.CreatedAt = time.Now().UTC()
	s.db.Create(&log)
	return log
}

func (s *PostgresStore) ListAuditLogs() []AuditLog {
	var logs []AuditLog
	s.db.Order("created_at DESC, id DESC").Find(&logs)
	return logs
}
