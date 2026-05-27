package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"k8s-ai-ops/backend/internal/app"
	"k8s-ai-ops/backend/internal/domain"
	"k8s-ai-ops/backend/internal/infra/auth"
)

func TestCreateUserAuditsAuthenticatedAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repos := newHTTPFakeRepositories()
	server := NewServer(app.NewServices(repos.domainRepositories(), staticCipher{}, nil))
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", &auth.UserContext{UserID: "admin-42", Username: "root", Role: "admin"})
		c.Next()
	})
	router.POST("/api/admin/users", server.createUser)

	body := bytes.NewBufferString(`{"username":"alice","role":"operator","displayName":"Alice"}`)
	req := httptest.NewRequest(nethttp.MethodPost, "/api/admin/users", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	if len(repos.audit.logs) != 1 {
		t.Fatalf("expected one audit log, got %d", len(repos.audit.logs))
	}
	if got := repos.audit.logs[0].ActorUserID; got != "admin-42" {
		t.Fatalf("expected audit actor admin-42, got %q", got)
	}
}

func TestCreateUserAuditFailureUsesRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repos := newHTTPFakeRepositories()
	repos.audit.appendErr = errors.New("audit database unavailable")
	server := NewServer(app.NewServices(repos.domainRepositories(), staticCipher{}, nil))
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Header("X-Request-ID", "req-123")
		c.Set("requestId", "req-123")
		c.Set("user", &auth.UserContext{UserID: "admin-42", Username: "root", Role: "admin"})
		c.Next()
	})
	router.POST("/api/admin/users", server.createUser)

	body := bytes.NewBufferString(`{"username":"alice","role":"operator","displayName":"Alice"}`)
	req := httptest.NewRequest(nethttp.MethodPost, "/api/admin/users", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Error struct {
			Code      string `json:"code"`
			RequestID string `json:"requestId"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error.Code != "AUDIT_RECORD_FAILED" {
		t.Fatalf("expected AUDIT_RECORD_FAILED, got %q", payload.Error.Code)
	}
	if payload.Error.RequestID != "req-123" {
		t.Fatalf("expected requestId req-123, got %q", payload.Error.RequestID)
	}
}

type httpFakeRepositories struct {
	users        httpFakeUserRepo
	permissions  httpFakePermissionRepo
	providers    httpFakeProviderRepo
	models       httpFakeModelRepo
	bindings     httpFakeBindingRepo
	sessions     httpFakeSessionRepo
	messages     httpFakeMessageRepo
	serviceAccts httpFakeServiceAccountRepo
	audit        httpFakeAuditRepo
}

func newHTTPFakeRepositories() *httpFakeRepositories {
	return &httpFakeRepositories{}
}

func (r *httpFakeRepositories) domainRepositories() *domain.Repositories {
	return &domain.Repositories{
		Users:        &r.users,
		Permissions:  &r.permissions,
		Providers:    &r.providers,
		Models:       &r.models,
		Bindings:     &r.bindings,
		ChatSessions: &r.sessions,
		ChatMessages: &r.messages,
		ServiceAccts: &r.serviceAccts,
		Audit:        &r.audit,
	}
}

type httpFakeUserRepo struct {
	users []domain.User
}

func (r *httpFakeUserRepo) FindAll(context.Context) ([]domain.User, error) { return r.users, nil }
func (r *httpFakeUserRepo) FindByID(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (r *httpFakeUserRepo) FindByUsername(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (r *httpFakeUserRepo) Save(_ context.Context, user *domain.User) error {
	if user.ID == "" {
		user.ID = "user-" + user.Username
	}
	r.users = append(r.users, *user)
	return nil
}

type httpFakePermissionRepo struct{}

func (r *httpFakePermissionRepo) FindByUser(context.Context, string) ([]domain.Permission, error) {
	return nil, nil
}
func (r *httpFakePermissionRepo) Replace(context.Context, string, []domain.Permission) error {
	return nil
}

type httpFakeProviderRepo struct{}

func (r *httpFakeProviderRepo) FindAll(context.Context) ([]domain.LLMProvider, error) {
	return nil, nil
}
func (r *httpFakeProviderRepo) FindByID(context.Context, string) (*domain.LLMProvider, error) {
	return nil, domain.ErrNotFound
}
func (r *httpFakeProviderRepo) Save(context.Context, *domain.LLMProvider) error { return nil }

type httpFakeModelRepo struct{}

func (r *httpFakeModelRepo) FindAll(context.Context) ([]domain.LLMModel, error) { return nil, nil }
func (r *httpFakeModelRepo) FindByID(context.Context, string) (*domain.LLMModel, error) {
	return nil, domain.ErrNotFound
}
func (r *httpFakeModelRepo) Save(context.Context, *domain.LLMModel) error { return nil }

type httpFakeBindingRepo struct{}

func (r *httpFakeBindingRepo) FindByUser(context.Context, string) ([]domain.LLMBinding, error) {
	return nil, nil
}
func (r *httpFakeBindingRepo) Replace(context.Context, string, []domain.LLMBinding) error { return nil }

type httpFakeSessionRepo struct{}

func (r *httpFakeSessionRepo) FindByID(context.Context, string) (*domain.ChatSession, error) {
	return nil, domain.ErrNotFound
}
func (r *httpFakeSessionRepo) FindByUser(context.Context, string) ([]domain.ChatSession, error) {
	return nil, nil
}
func (r *httpFakeSessionRepo) Save(context.Context, *domain.ChatSession) error { return nil }

type httpFakeMessageRepo struct{}

func (r *httpFakeMessageRepo) Append(context.Context, *domain.ChatMessage) error { return nil }
func (r *httpFakeMessageRepo) FindBySession(context.Context, string) ([]domain.ChatMessage, error) {
	return nil, nil
}

type httpFakeServiceAccountRepo struct{}

func (r *httpFakeServiceAccountRepo) FindToken(context.Context, string) (*domain.ServiceAccountToken, error) {
	return nil, domain.ErrNotFound
}
func (r *httpFakeServiceAccountRepo) SaveToken(context.Context, *domain.ServiceAccountToken) error {
	return nil
}
func (r *httpFakeServiceAccountRepo) SaveBinding(context.Context, *domain.ServiceAccountBinding) error {
	return nil
}
func (r *httpFakeServiceAccountRepo) FindBindings(context.Context, string) ([]domain.ServiceAccountBinding, error) {
	return nil, nil
}

type httpFakeAuditRepo struct {
	appendErr error
	logs      []domain.AuditLog
}

func (r *httpFakeAuditRepo) Append(_ context.Context, log *domain.AuditLog) error {
	if r.appendErr != nil {
		return r.appendErr
	}
	r.logs = append(r.logs, *log)
	return nil
}
func (r *httpFakeAuditRepo) FindAll(context.Context) ([]domain.AuditLog, error) { return r.logs, nil }

type staticCipher struct{}

func (staticCipher) Encrypt(plaintext string) (string, error)  { return plaintext, nil }
func (staticCipher) Decrypt(ciphertext string) (string, error) { return ciphertext, nil }
