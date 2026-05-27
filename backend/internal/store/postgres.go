package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	store := &PostgresStore{db: db}
	if err := store.InitSchema(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := store.SeedDemoData(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}

func (s *PostgresStore) InitSchema(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			keycloak_user_id TEXT NOT NULL DEFAULT '',
			username TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL DEFAULT '',
			email TEXT NOT NULL DEFAULT '',
			role TEXT NOT NULL,
			status TEXT NOT NULL,
			created_by TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS k8s_permissions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			namespace TEXT NOT NULL,
			api_group TEXT NOT NULL DEFAULT '',
			resource TEXT NOT NULL,
			verbs_json TEXT NOT NULL,
			role_name TEXT NOT NULL DEFAULT '',
			role_binding_name TEXT NOT NULL DEFAULT '',
			enabled BOOLEAN NOT NULL,
			created_by TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS llm_providers (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			protocol TEXT NOT NULL,
			base_url TEXT NOT NULL DEFAULT '',
			api_key_ciphertext TEXT NOT NULL DEFAULT '',
			enabled BOOLEAN NOT NULL,
			created_by TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS llm_models (
			id TEXT PRIMARY KEY,
			provider_id TEXT NOT NULL,
			model_name TEXT NOT NULL,
			display_name TEXT NOT NULL DEFAULT '',
			supports_tools BOOLEAN NOT NULL,
			supports_streaming BOOLEAN NOT NULL,
			enabled BOOLEAN NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id TEXT PRIMARY KEY,
			actor_user_id TEXT NOT NULL DEFAULT '',
			action TEXT NOT NULL,
			target_type TEXT NOT NULL DEFAULT '',
			target_id TEXT NOT NULL DEFAULT '',
			namespace TEXT NOT NULL DEFAULT '',
			resource TEXT NOT NULL DEFAULT '',
			verb TEXT NOT NULL DEFAULT '',
			allowed BOOLEAN NOT NULL,
			reason TEXT NOT NULL DEFAULT '',
			request_json TEXT NOT NULL DEFAULT '',
			response_json TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS user_llm_bindings (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			model_id TEXT NOT NULL,
			is_default BOOLEAN NOT NULL,
			created_by TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS k8s_service_accounts (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			namespace TEXT NOT NULL,
			service_account_name TEXT NOT NULL,
			token_secret_name TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS chat_sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			model_id TEXT NOT NULL DEFAULT '',
			title TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS chat_messages (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			tool_name TEXT NOT NULL DEFAULT '',
			tool_args_json TEXT NOT NULL DEFAULT '',
			tool_result_json TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL
		)`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *PostgresStore) SeedDemoData(ctx context.Context) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `INSERT INTO users (id, keycloak_user_id, username, display_name, email, role, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (id) DO NOTHING`,
		"demo-user", "keycloak-demo-user", "demo", "Demo Operator", "operator@example.com", RoleOperator, "active", now, now)
	if err != nil {
		return err
	}
	if len(s.ListUserPermissions("demo-user")) == 0 {
		s.ReplaceUserPermissions("demo-user", []Permission{
			{Namespace: "dev", APIGroup: "", Resource: "pods", Verbs: []string{"get", "list", "watch"}, Enabled: true},
			{Namespace: "dev", APIGroup: "", Resource: "pods/log", Verbs: []string{"get"}, Enabled: true},
		})
	}
	return nil
}

func (s *PostgresStore) CurrentDemoUser() User {
	row := s.db.QueryRow(`SELECT id, keycloak_user_id, username, display_name, email, role, status, created_by, created_at, updated_at FROM users WHERE id=$1`, "demo-user")
	user, err := scanUser(row)
	if err != nil {
		return NewMemoryStore().CurrentDemoUser()
	}
	return user
}

func (s *PostgresStore) ListUsers() []User {
	rows, err := s.db.Query(`SELECT id, keycloak_user_id, username, display_name, email, role, status, created_by, created_at, updated_at FROM users ORDER BY created_at, id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		user, err := scanUser(rows)
		if err == nil {
			users = append(users, user)
		}
	}
	return users
}

func (s *PostgresStore) CreateUser(user User) User {
	now := time.Now().UTC()
	user.ID = defaultString(user.ID, "user-"+user.Username)
	user.Status = defaultString(user.Status, "active")
	user.CreatedAt = now
	user.UpdatedAt = now
	_, _ = s.db.Exec(`INSERT INTO users (id, keycloak_user_id, username, display_name, email, role, status, created_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (id) DO UPDATE SET username=$3, display_name=$4, email=$5, role=$6, status=$7, updated_at=$10`,
		user.ID, user.KeycloakUserID, user.Username, user.DisplayName, user.Email, user.Role, user.Status, user.CreatedBy, user.CreatedAt, user.UpdatedAt)
	return user
}

func (s *PostgresStore) ListUserPermissions(userID string) []Permission {
	rows, err := s.db.Query(`SELECT id, user_id, namespace, api_group, resource, verbs_json, role_name, role_binding_name, enabled, created_by, created_at, updated_at
		FROM k8s_permissions WHERE user_id=$1 ORDER BY namespace, resource`, userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var permissions []Permission
	for rows.Next() {
		var p Permission
		var verbsJSON string
		if err := rows.Scan(&p.ID, &p.UserID, &p.Namespace, &p.APIGroup, &p.Resource, &verbsJSON, &p.RoleName, &p.RoleBindingName, &p.Enabled, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err == nil {
			_ = json.Unmarshal([]byte(verbsJSON), &p.Verbs)
			permissions = append(permissions, p)
		}
	}
	return permissions
}

func (s *PostgresStore) ReplaceUserPermissions(userID string, permissions []Permission) []Permission {
	tx, err := s.db.Begin()
	if err != nil {
		return nil
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM k8s_permissions WHERE user_id=$1`, userID); err != nil {
		return nil
	}
	now := time.Now().UTC()
	for index := range permissions {
		permissions[index].ID = userID + "-perm-" + permissions[index].Namespace + "-" + strings.ReplaceAll(permissions[index].Resource, "/", "-")
		permissions[index].UserID = userID
		permissions[index].Enabled = true
		permissions[index].CreatedAt = now
		permissions[index].UpdatedAt = now
		verbsJSON, _ := json.Marshal(permissions[index].Verbs)
		if _, err := tx.Exec(`INSERT INTO k8s_permissions (id, user_id, namespace, api_group, resource, verbs_json, role_name, role_binding_name, enabled, created_by, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			permissions[index].ID, permissions[index].UserID, permissions[index].Namespace, permissions[index].APIGroup, permissions[index].Resource, string(verbsJSON), permissions[index].RoleName, permissions[index].RoleBindingName, permissions[index].Enabled, permissions[index].CreatedBy, permissions[index].CreatedAt, permissions[index].UpdatedAt); err != nil {
			return nil
		}
	}
	if err := tx.Commit(); err != nil {
		return nil
	}
	return permissions
}

func (s *PostgresStore) ListLLMProviders() []LLMProvider {
	rows, err := s.db.Query(`SELECT id, name, protocol, base_url, api_key_ciphertext, enabled, created_by, created_at, updated_at FROM llm_providers ORDER BY created_at, id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var providers []LLMProvider
	for rows.Next() {
		var p LLMProvider
		if err := rows.Scan(&p.ID, &p.Name, &p.Protocol, &p.BaseURL, &p.APIKeyCiphertext, &p.Enabled, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err == nil {
			providers = append(providers, p)
		}
	}
	return providers
}

func (s *PostgresStore) CreateLLMProvider(provider LLMProvider) LLMProvider {
	now := time.Now().UTC()
	provider.ID = defaultString(provider.ID, "provider-"+provider.Protocol)
	provider.APIKeyCiphertext = "encrypted:" + provider.Name
	provider.CreatedAt = now
	provider.UpdatedAt = now
	_, _ = s.db.Exec(`INSERT INTO llm_providers (id, name, protocol, base_url, api_key_ciphertext, enabled, created_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (id) DO UPDATE SET name=$2, protocol=$3, base_url=$4, api_key_ciphertext=$5, enabled=$6, updated_at=$9`,
		provider.ID, provider.Name, provider.Protocol, provider.BaseURL, provider.APIKeyCiphertext, provider.Enabled, provider.CreatedBy, provider.CreatedAt, provider.UpdatedAt)
	return provider
}

func (s *PostgresStore) ListLLMModels() []LLMModel {
	rows, err := s.db.Query(`SELECT id, provider_id, model_name, display_name, supports_tools, supports_streaming, enabled, created_at, updated_at FROM llm_models ORDER BY created_at, id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var models []LLMModel
	for rows.Next() {
		var m LLMModel
		if err := rows.Scan(&m.ID, &m.ProviderID, &m.ModelName, &m.DisplayName, &m.SupportsTools, &m.SupportsStreaming, &m.Enabled, &m.CreatedAt, &m.UpdatedAt); err == nil {
			models = append(models, m)
		}
	}
	return models
}

func (s *PostgresStore) CreateLLMModel(model LLMModel) LLMModel {
	now := time.Now().UTC()
	model.ID = defaultString(model.ID, "model-"+model.ModelName)
	model.CreatedAt = now
	model.UpdatedAt = now
	_, _ = s.db.Exec(`INSERT INTO llm_models (id, provider_id, model_name, display_name, supports_tools, supports_streaming, enabled, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (id) DO UPDATE SET provider_id=$2, model_name=$3, display_name=$4, supports_tools=$5, supports_streaming=$6, enabled=$7, updated_at=$9`,
		model.ID, model.ProviderID, model.ModelName, model.DisplayName, model.SupportsTools, model.SupportsStreaming, model.Enabled, model.CreatedAt, model.UpdatedAt)
	return model
}

func (s *PostgresStore) AppendAuditLog(log AuditLog) AuditLog {
	log.ID = defaultString(log.ID, "audit-"+time.Now().UTC().Format("20060102150405.000000000"))
	log.CreatedAt = time.Now().UTC()
	_, _ = s.db.Exec(`INSERT INTO audit_logs (id, actor_user_id, action, target_type, target_id, namespace, resource, verb, allowed, reason, request_json, response_json, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		log.ID, log.ActorUserID, log.Action, log.TargetType, log.TargetID, log.Namespace, log.Resource, log.Verb, log.Allowed, log.Reason, log.RequestJSON, log.ResponseJSON, log.CreatedAt)
	return log
}

func (s *PostgresStore) ListAuditLogs() []AuditLog {
	rows, err := s.db.Query(`SELECT id, actor_user_id, action, target_type, target_id, namespace, resource, verb, allowed, reason, request_json, response_json, created_at FROM audit_logs ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var logs []AuditLog
	for rows.Next() {
		var log AuditLog
		if err := rows.Scan(&log.ID, &log.ActorUserID, &log.Action, &log.TargetType, &log.TargetID, &log.Namespace, &log.Resource, &log.Verb, &log.Allowed, &log.Reason, &log.RequestJSON, &log.ResponseJSON, &log.CreatedAt); err == nil {
			logs = append(logs, log)
		}
	}
	return logs
}

type userScanner interface {
	Scan(dest ...any) error
}

func scanUser(scanner userScanner) (User, error) {
	var user User
	err := scanner.Scan(&user.ID, &user.KeycloakUserID, &user.Username, &user.DisplayName, &user.Email, &user.Role, &user.Status, &user.CreatedBy, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return User{}, fmt.Errorf("scan user: %w", err)
	}
	return user, nil
}
