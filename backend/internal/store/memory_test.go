package store

import "testing"

func TestMemoryStoreCreatesAndListsUsers(t *testing.T) {
	store := NewMemoryStore()

	created := store.CreateUser(User{
		Username:    "operator-b",
		DisplayName: "Operator B",
		Email:       "operator-b@example.com",
		Role:        RoleOperator,
		Status:      "active",
	})

	users := store.ListUsers()

	if created.ID == "" {
		t.Fatal("expected created user id")
	}
	if len(users) != 2 {
		t.Fatalf("expected demo user plus created user, got %d", len(users))
	}
	if users[1].Username != "operator-b" {
		t.Fatalf("expected operator-b, got %s", users[1].Username)
	}
}

func TestMemoryStoreReplacesUserPermissions(t *testing.T) {
	store := NewMemoryStore()

	store.ReplaceUserPermissions("demo-user", []Permission{
		{Namespace: "test", APIGroup: "", Resource: "pods", Verbs: []string{"get", "list"}, Enabled: true},
	})

	permissions := store.ListUserPermissions("demo-user")

	if len(permissions) != 1 {
		t.Fatalf("expected one permission, got %d", len(permissions))
	}
	if permissions[0].Namespace != "test" {
		t.Fatalf("expected test namespace, got %s", permissions[0].Namespace)
	}
}

func TestMemoryStoreStoresProviderWithoutPlaintextAPIKey(t *testing.T) {
	store := NewMemoryStore()

	provider := store.CreateLLMProvider(LLMProvider{
		Name:             "OpenAI",
		Protocol:         "openai",
		BaseURL:          "https://api.openai.com/v1",
		APIKeyCiphertext: "secret-key",
		Enabled:          true,
	})

	if provider.APIKeyCiphertext == "secret-key" {
		t.Fatal("expected provider api key to be transformed before storage")
	}
	if provider.APIKeyCiphertext == "" {
		t.Fatal("expected encrypted api key marker")
	}
}

func TestMemoryStoreCreatesLLMModel(t *testing.T) {
	store := NewMemoryStore()

	model := store.CreateLLMModel(LLMModel{
		ProviderID:        "provider-openai",
		ModelName:         "gpt-4.1",
		DisplayName:       "GPT 4.1",
		SupportsTools:     true,
		SupportsStreaming: true,
		Enabled:           true,
	})

	models := store.ListLLMModels()

	if model.ID == "" {
		t.Fatal("expected model id")
	}
	if len(models) != 1 || models[0].ModelName != "gpt-4.1" {
		t.Fatalf("expected stored gpt-4.1 model, got %#v", models)
	}
}

func TestMemoryStoreAppendsAuditLog(t *testing.T) {
	store := NewMemoryStore()

	store.AppendAuditLog(AuditLog{
		ActorUserID: "demo-user",
		Action:      "chat.message.create",
		TargetType:  "chat_message",
		Allowed:     true,
	})

	logs := store.ListAuditLogs()

	if len(logs) != 1 {
		t.Fatalf("expected one audit log, got %d", len(logs))
	}
	if logs[0].ID == "" {
		t.Fatal("expected audit log id")
	}
	if logs[0].Action != "chat.message.create" {
		t.Fatalf("unexpected action: %s", logs[0].Action)
	}
}

func TestMemoryStoreCreatesAndGetsChatSession(t *testing.T) {
	store := NewMemoryStore()
	created := store.CreateChatSession(ChatSession{UserID: "demo-user", Title: "test"})
	if created.ID == "" {
		t.Fatal("expected session id")
	}
	got, ok := store.GetChatSession(created.ID)
	if !ok || got.ID != created.ID {
		t.Fatal("expected to get created session")
	}
	sessions := store.ListUserChatSessions("demo-user")
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
}

func TestMemoryStoreAppendsAndListsChatMessages(t *testing.T) {
	store := NewMemoryStore()
	msg := store.AppendChatMessage(ChatMessage{SessionID: "s1", Role: "user", Content: "hello"})
	if msg.ID == "" {
		t.Fatal("expected message id")
	}
	msgs := store.ListSessionMessages("s1")
	if len(msgs) != 1 || msgs[0].Content != "hello" {
		t.Fatalf("unexpected messages: %#v", msgs)
	}
}

func TestMemoryStoreSetsAndGetsUserLLMBindings(t *testing.T) {
	store := NewMemoryStore()
	bindings := store.SetUserLLMBindings("u1", []string{"m1", "m2"}, "m1")
	if len(bindings) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(bindings))
	}
	got := store.GetUserLLMBindings("u1")
	if len(got) != 2 {
		t.Fatalf("expected 2 bindings from get, got %d", len(got))
	}
}

func TestMemoryStoreSavesAndGetsServiceAccountBinding(t *testing.T) {
	store := NewMemoryStore()
	binding := store.SaveServiceAccountBinding(ServiceAccountBinding{
		UserID:             "u1",
		Namespace:          "dev",
		ServiceAccountName: "k8s-ai-operator-u1",
	})
	if binding.ID == "" {
		t.Fatal("expected binding id")
	}
	bindings := store.GetUserServiceAccountBindings("u1")
	if len(bindings) != 1 {
		t.Fatalf("expected 1 binding, got %d", len(bindings))
	}
	store.saTokens["u1"] = ServiceAccountToken{UserID: "u1", Token: "tok", Namespace: "dev"}
	token, ns, err := store.GetServiceAccountToken("u1")
	if err != nil || token != "tok" || ns != "dev" {
		t.Fatalf("unexpected token: token=%s ns=%s err=%v", token, ns, err)
	}
}
