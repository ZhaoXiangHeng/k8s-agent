package app

import (
	"context"
	"errors"
	"io"
	"testing"

	"k8s-ai-ops/backend/internal/domain"
	agentv1 "k8s-ai-ops/proto/agent/v1"
)

func TestAuditRecordReturnsRepositoryError(t *testing.T) {
	wantErr := errors.New("audit unavailable")
	repos := newFakeRepositories()
	repos.audit.appendErr = wantErr

	svc := &AuditService{repos: repos.domainRepositories()}

	err := svc.Record(context.Background(), "admin-1", "admin.user.create", "user", "user-1", true, "created")

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected audit append error %v, got %v", wantErr, err)
	}
}

func TestChatProcessMessageStopsWhenUserMessageCannotBeSaved(t *testing.T) {
	wantErr := errors.New("message store unavailable")
	repos := newFakeRepositories()
	repos.messages.appendErr = wantErr
	agentClient := &fakeAgentClient{stream: &fakeAgentStream{}}
	svc := NewChatService(repos.domainRepositories(), agentClient, staticCipher{plaintext: "secret"})

	err := svc.ProcessMessage(context.Background(), ChatMessageRequest{
		ModelID: "model-1",
		Content: "kubectl get pods",
	}, "user-1", "alice", "session-1", nil)

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected message append error %v, got %v", wantErr, err)
	}
	if agentClient.calls != 0 {
		t.Fatalf("expected agent client not to be called, got %d calls", agentClient.calls)
	}
}

func TestChatProcessMessageReturnsPermissionLookupError(t *testing.T) {
	wantErr := errors.New("permission repository unavailable")
	repos := newFakeRepositories()
	repos.permissions.findErr = wantErr
	agentClient := &fakeAgentClient{stream: &fakeAgentStream{}}
	svc := NewChatService(repos.domainRepositories(), agentClient, staticCipher{plaintext: "secret"})

	err := svc.ProcessMessage(context.Background(), ChatMessageRequest{
		ModelID: "model-1",
		Content: "kubectl get pods",
	}, "user-1", "alice", "session-1", nil)

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected permission lookup error %v, got %v", wantErr, err)
	}
	if agentClient.calls != 0 {
		t.Fatalf("expected agent client not to be called, got %d calls", agentClient.calls)
	}
}

func TestChatProcessMessageRejectsSessionOwnedByAnotherUser(t *testing.T) {
	repos := newFakeRepositories()
	repos.sessions.session = &domain.ChatSession{ID: "session-1", UserID: "user-2", Status: "active"}
	repos.bindings.bindings = []domain.LLMBinding{{UserID: "user-1", ModelID: "model-1"}}
	agentClient := &fakeAgentClient{stream: &fakeAgentStream{}}
	svc := NewChatService(repos.domainRepositories(), agentClient, staticCipher{plaintext: "secret"})

	err := svc.ProcessMessage(context.Background(), ChatMessageRequest{
		ModelID: "model-1",
		Content: "kubectl get pods",
	}, "user-1", "alice", "session-1", nil)

	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden session error, got %v", err)
	}
	if agentClient.calls != 0 {
		t.Fatalf("expected agent client not to be called, got %d calls", agentClient.calls)
	}
}

func TestChatProcessMessageRejectsUnboundModel(t *testing.T) {
	repos := newFakeRepositories()
	repos.sessions.session = &domain.ChatSession{ID: "session-1", UserID: "user-1", Status: "active"}
	repos.bindings.bindings = []domain.LLMBinding{{UserID: "user-1", ModelID: "model-2"}}
	agentClient := &fakeAgentClient{stream: &fakeAgentStream{}}
	svc := NewChatService(repos.domainRepositories(), agentClient, staticCipher{plaintext: "secret"})

	err := svc.ProcessMessage(context.Background(), ChatMessageRequest{
		ModelID: "model-1",
		Content: "kubectl get pods",
	}, "user-1", "alice", "session-1", nil)

	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden model error, got %v", err)
	}
	if agentClient.calls != 0 {
		t.Fatalf("expected agent client not to be called, got %d calls", agentClient.calls)
	}
}

func TestListModelsReturnsOnlyModelsBoundToUser(t *testing.T) {
	repos := newFakeRepositories()
	repos.models.models = []domain.LLMModel{
		{ID: "model-1", ProviderID: "provider-1", ModelName: "gpt-allowed", Enabled: true},
		{ID: "model-2", ProviderID: "provider-1", ModelName: "gpt-denied", Enabled: true},
		{ID: "model-3", ProviderID: "provider-1", ModelName: "gpt-disabled", Enabled: false},
	}
	repos.bindings.bindings = []domain.LLMBinding{{UserID: "user-1", ModelID: "model-1"}}
	svc := NewLLMService(repos.domainRepositories(), staticCipher{})

	models, err := svc.ListModels(context.Background(), "user-1", true)

	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected one bound enabled model, got %#v", models)
	}
	if models[0].ID != "model-1" {
		t.Fatalf("expected model-1, got %q", models[0].ID)
	}
}

type fakeRepositories struct {
	users        fakeUserRepo
	permissions  fakePermissionRepo
	providers    fakeProviderRepo
	models       fakeModelRepo
	bindings     fakeBindingRepo
	sessions     fakeSessionRepo
	messages     fakeMessageRepo
	serviceAccts fakeServiceAccountRepo
	audit        fakeAuditRepo
}

func newFakeRepositories() *fakeRepositories {
	return &fakeRepositories{
		providers: fakeProviderRepo{provider: domain.LLMProvider{
			ID: "provider-1", Protocol: domain.ProtocolOpenAI, BaseURL: "https://llm.example", APIKeyCiphertext: "cipher", Enabled: true,
		}},
		models: fakeModelRepo{model: domain.LLMModel{
			ID: "model-1", ProviderID: "provider-1", ModelName: "gpt-test", Enabled: true,
		}},
		permissions: fakePermissionRepo{permissions: []domain.Permission{{
			UserID: "user-1", Namespace: "default", Resource: "pods", Verbs: []string{"get"}, Enabled: true,
		}}},
		bindings: fakeBindingRepo{bindings: []domain.LLMBinding{{UserID: "user-1", ModelID: "model-1"}}},
		sessions: fakeSessionRepo{session: &domain.ChatSession{ID: "session-1", UserID: "user-1", Status: "active"}},
	}
}

func (r *fakeRepositories) domainRepositories() *domain.Repositories {
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

type fakeUserRepo struct{}

func (r *fakeUserRepo) FindAll(context.Context) ([]domain.User, error) { return nil, nil }
func (r *fakeUserRepo) FindByID(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (r *fakeUserRepo) FindByUsername(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (r *fakeUserRepo) Save(context.Context, *domain.User) error { return nil }

type fakePermissionRepo struct {
	permissions []domain.Permission
	findErr     error
}

func (r *fakePermissionRepo) FindByUser(context.Context, string) ([]domain.Permission, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	return r.permissions, nil
}
func (r *fakePermissionRepo) Replace(context.Context, string, []domain.Permission) error { return nil }

type fakeProviderRepo struct {
	provider domain.LLMProvider
}

func (r *fakeProviderRepo) FindAll(context.Context) ([]domain.LLMProvider, error) {
	return []domain.LLMProvider{r.provider}, nil
}
func (r *fakeProviderRepo) FindByID(context.Context, string) (*domain.LLMProvider, error) {
	return &r.provider, nil
}
func (r *fakeProviderRepo) Save(_ context.Context, provider *domain.LLMProvider) error {
	r.provider = *provider
	return nil
}

type fakeModelRepo struct {
	model  domain.LLMModel
	models []domain.LLMModel
}

func (r *fakeModelRepo) FindAll(context.Context) ([]domain.LLMModel, error) {
	if r.models != nil {
		return r.models, nil
	}
	return []domain.LLMModel{r.model}, nil
}
func (r *fakeModelRepo) FindByID(context.Context, string) (*domain.LLMModel, error) {
	return &r.model, nil
}
func (r *fakeModelRepo) Save(_ context.Context, model *domain.LLMModel) error {
	r.model = *model
	return nil
}

func (r *fakeBindingRepo) FindByUser(context.Context, string) ([]domain.LLMBinding, error) {
	return r.bindings, nil
}
func (r *fakeBindingRepo) Replace(context.Context, string, []domain.LLMBinding) error { return nil }

type fakeBindingRepo struct {
	bindings []domain.LLMBinding
}

type fakeSessionRepo struct {
	session *domain.ChatSession
}

func (r *fakeSessionRepo) FindByID(context.Context, string) (*domain.ChatSession, error) {
	if r.session != nil {
		return r.session, nil
	}
	return nil, domain.ErrNotFound
}
func (r *fakeSessionRepo) FindByUser(context.Context, string) ([]domain.ChatSession, error) {
	return nil, nil
}
func (r *fakeSessionRepo) Save(context.Context, *domain.ChatSession) error { return nil }

type fakeMessageRepo struct {
	appendErr error
	messages  []domain.ChatMessage
}

func (r *fakeMessageRepo) Append(_ context.Context, msg *domain.ChatMessage) error {
	if r.appendErr != nil {
		return r.appendErr
	}
	r.messages = append(r.messages, *msg)
	return nil
}
func (r *fakeMessageRepo) FindBySession(context.Context, string) ([]domain.ChatMessage, error) {
	return r.messages, nil
}

type fakeServiceAccountRepo struct{}

func (r *fakeServiceAccountRepo) FindToken(context.Context, string) (*domain.ServiceAccountToken, error) {
	return nil, domain.ErrNotFound
}
func (r *fakeServiceAccountRepo) SaveToken(context.Context, *domain.ServiceAccountToken) error {
	return nil
}
func (r *fakeServiceAccountRepo) SaveBinding(context.Context, *domain.ServiceAccountBinding) error {
	return nil
}
func (r *fakeServiceAccountRepo) FindBindings(context.Context, string) ([]domain.ServiceAccountBinding, error) {
	return nil, nil
}

type fakeAuditRepo struct {
	appendErr error
	logs      []domain.AuditLog
}

func (r *fakeAuditRepo) Append(_ context.Context, log *domain.AuditLog) error {
	if r.appendErr != nil {
		return r.appendErr
	}
	r.logs = append(r.logs, *log)
	return nil
}
func (r *fakeAuditRepo) FindAll(context.Context) ([]domain.AuditLog, error) { return r.logs, nil }

type fakeAgentClient struct {
	calls  int
	stream AgentStream
}

func (c *fakeAgentClient) RunStream(context.Context, *agentv1.AgentRunRequest) (AgentStream, error) {
	c.calls++
	return c.stream, nil
}

type fakeAgentStream struct{}

func (s *fakeAgentStream) Recv() (*agentv1.StreamEvent, error) { return nil, io.EOF }

type staticCipher struct {
	ciphertext string
	plaintext  string
}

func (c staticCipher) Encrypt(string) (string, error) { return c.ciphertext, nil }
func (c staticCipher) Decrypt(string) (string, error) { return c.plaintext, nil }
