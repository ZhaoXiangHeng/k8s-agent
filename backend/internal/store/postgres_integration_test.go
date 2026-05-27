package store

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestPostgresStoreIntegration(t *testing.T) {
	databaseURL := os.Getenv("K8S_AI_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("K8S_AI_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store, err := NewPostgresStore(ctx, databaseURL)
	if err != nil {
		t.Fatalf("new postgres store: %v", err)
	}
	defer store.Close()

	user := store.CreateUser(User{
		Username:    "integration-operator",
		DisplayName: "Integration Operator",
		Email:       "integration@example.com",
		Role:        RoleOperator,
		Status:      "active",
	})
	if user.ID == "" {
		t.Fatal("expected user id")
	}

	store.ReplaceUserPermissions(user.ID, []Permission{
		{Namespace: "it", APIGroup: "", Resource: "pods", Verbs: []string{"get", "list"}, Enabled: true},
	})
	permissions := store.ListUserPermissions(user.ID)
	if len(permissions) != 1 || permissions[0].Namespace != "it" {
		t.Fatalf("expected persisted permission, got %#v", permissions)
	}

	provider := store.CreateLLMProvider(LLMProvider{Name: "OpenAI", Protocol: "openai", BaseURL: "https://api.openai.com/v1", APIKeyCiphertext: "plain", Enabled: true})
	if provider.APIKeyCiphertext == "plain" {
		t.Fatal("expected api key to be transformed")
	}

	model := store.CreateLLMModel(LLMModel{ProviderID: provider.ID, ModelName: "gpt-4.1", DisplayName: "GPT 4.1", SupportsTools: true, SupportsStreaming: true, Enabled: true})
	if model.ID == "" {
		t.Fatal("expected model id")
	}

	store.AppendAuditLog(AuditLog{ActorUserID: user.ID, Action: "integration.audit", TargetType: "test", Allowed: true})
	if len(store.ListAuditLogs()) == 0 {
		t.Fatal("expected audit logs")
	}
}
