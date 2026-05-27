# K8S AI Ops MVP 补齐实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将当前架构骨架推进到可演示的 MVP，实现 LLM Chat → 授权校验 → MCP K8S 工具执行的真实调用链

**Architecture:** Backend（HTTP + gRPC 双协议）+ MCP Server（gRPC 回查 Backend 获取 SA Token）+ Traefik Ingress + Keycloak 认证。Helm 重组为 Umbrella Chart + 7 个子 chart

**Tech Stack:** Go 1.26, gRPC, React + TypeScript, Helm, Traefik, Keycloak, PostgreSQL, Redis

---

## 阶段一：数据库和 Store 层补齐

### Task 1.1: Store 接口新增方法

**Files:**
- Modify: `backend/internal/store/store.go:1-15`
- Test: `backend/internal/store/memory_test.go`

- [ ] **Step 1: Store 接口新增 Chat/LLM/SA 方法**

```go
// 在 backend/internal/store/store.go 中添加以下方法到 Store interface
type Store interface {
    // ... 已有方法

    // Chat 会话
    CreateChatSession(session ChatSession) ChatSession
    GetChatSession(id string) (ChatSession, bool)
    ListUserChatSessions(userID string) []ChatSession

    // Chat 消息
    AppendChatMessage(msg ChatMessage) ChatMessage
    ListSessionMessages(sessionID string) []ChatMessage

    // LLM 绑定
    SetUserLLMBindings(userID string, modelIDs []string, defaultModelID string) []UserLLMBinding
    GetUserLLMBindings(userID string) []UserLLMBinding

    // ServiceAccount 绑定
    SaveServiceAccountBinding(binding ServiceAccountBinding) ServiceAccountBinding
    GetUserServiceAccountBindings(userID string) []ServiceAccountBinding
    GetServiceAccountToken(userID string) (string, string, error) // token, namespace, error
}
```

- [ ] **Step 2: Run existing tests to verify interface compiles**

Run: `cd backend && go build ./...`
Expected: Build fails because MemoryStore doesn't implement new methods

### Task 1.2: MemoryStore 实现

**Files:**
- Modify: `backend/internal/store/memory.go`
- Test: `backend/internal/store/memory_test.go`

- [ ] **Step 1: MemoryStore 新增字段和初始化**

```go
// 在 MemoryStore struct 中新增字段
type MemoryStore struct {
    users       []User
    permissions []Permission
    providers   []LLMProvider
    models      []LLMModel
    auditLogs   []AuditLog
    // 新增
    chatSessions []ChatSession
    chatMessages []ChatMessage
    llmBindings  []UserLLMBinding
    saBindings   []ServiceAccountBinding
    saTokens     map[string]ServiceAccountToken // userID -> token info
}

// 在 NewMemoryStore 中初始化
func NewMemoryStore() *MemoryStore {
    return &MemoryStore{
        // ... 已有初始化
        chatSessions: []ChatSession{},
        chatMessages: []ChatMessage{},
        llmBindings:  []UserLLMBinding{},
        saBindings:   []ServiceAccountBinding{},
        saTokens:     map[string]ServiceAccountToken{},
    }
}
```

- [ ] **Step 2: 实现 ChatSession 方法**

```go
func (s *MemoryStore) CreateChatSession(session ChatSession) ChatSession {
    now := time.Now().UTC()
    session.ID = "session-" + now.Format("20060102150405.000000000")
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
```

- [ ] **Step 3: 实现 ChatMessage 方法**

```go
func (s *MemoryStore) AppendChatMessage(msg ChatMessage) ChatMessage {
    msg.ID = "msg-" + time.Now().UTC().Format("20060102150405.000000000")
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
```

- [ ] **Step 4: 实现 LLMBinding 方法**

```go
func (s *MemoryStore) SetUserLLMBindings(userID string, modelIDs []string, defaultModelID string) []UserLLMBinding {
    // 移除旧绑定
    var kept []UserLLMBinding
    for _, b := range s.llmBindings {
        if b.UserID != userID {
            kept = append(kept, b)
        }
    }
    // 添加新绑定
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
```

- [ ] **Step 5: 实现 ServiceAccount 绑定方法**

```go
type ServiceAccountToken struct {
    UserID    string
    Token     string
    Namespace string
}

func (s *MemoryStore) SaveServiceAccountBinding(binding ServiceAccountBinding) ServiceAccountBinding {
    now := time.Now().UTC()
    binding.ID = "sab-" + binding.UserID + "-" + binding.Namespace
    binding.CreatedAt = now
    binding.UpdatedAt = now
    if binding.Status == "" {
        binding.Status = "active"
    }
    // 替换已存在的
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
```

- [ ] **Step 6: 写 MemoryStore 测试**

```go
// backend/internal/store/memory_test.go 新增

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
```

- [ ] **Step 7: Run tests**

Run: `cd backend && go test ./internal/store -run TestMemoryStore -v`
Expected: All PASS

- [ ] **Step 8: Commit**

```bash
git add backend/internal/store/store.go backend/internal/store/memory.go backend/internal/store/memory_test.go
git commit -m "feat(store): add chat session, LLM binding, and SA binding methods to Store interface and MemoryStore"

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

### Task 1.3: PostgresStore schema 新增表

**Files:**
- Modify: `backend/internal/store/postgres.go`

- [ ] **Step 1: 在 InitSchema 中添加 4 张新表**

```go
// 在 InitSchema 的 statements 切片中新增
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
```

- [ ] **Step 2: Run compile check**

Run: `cd backend && go build ./...`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add backend/internal/store/postgres.go
git commit -m "feat(store): add user_llm_bindings, k8s_service_accounts, chat_sessions, chat_messages tables to PostgresStore"

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

### Task 1.4: PostgresStore 实现新方法

**Files:**
- Modify: `backend/internal/store/postgres.go`
- Test: `backend/internal/store/postgres_integration_test.go`

- [ ] **Step 1: 实现 ChatSession CRUD**

```go
// 在 postgres.go 中添加

func (s *PostgresStore) CreateChatSession(session ChatSession) ChatSession {
    now := time.Now().UTC()
    session.ID = defaultString(session.ID, "session-"+now.Format("20060102150405.000000000"))
    session.CreatedAt = now
    session.UpdatedAt = now
    if session.Status == "" {
        session.Status = "active"
    }
    s.db.Exec(`INSERT INTO chat_sessions (id, user_id, model_id, title, status, created_at, updated_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (id) DO NOTHING`,
        session.ID, session.UserID, session.ModelID, session.Title, session.Status, session.CreatedAt, session.UpdatedAt)
    return session
}

func (s *PostgresStore) GetChatSession(id string) (ChatSession, bool) {
    row := s.db.QueryRow(`SELECT id, user_id, model_id, title, status, created_at, updated_at FROM chat_sessions WHERE id=$1`, id)
    var session ChatSession
    err := row.Scan(&session.ID, &session.UserID, &session.ModelID, &session.Title, &session.Status, &session.CreatedAt, &session.UpdatedAt)
    if err != nil {
        return ChatSession{}, false
    }
    return session, true
}

func (s *PostgresStore) ListUserChatSessions(userID string) []ChatSession {
    rows, err := s.db.Query(`SELECT id, user_id, model_id, title, status, created_at, updated_at FROM chat_sessions WHERE user_id=$1 ORDER BY created_at DESC`, userID)
    if err != nil {
        return nil
    }
    defer rows.Close()
    var sessions []ChatSession
    for rows.Next() {
        var session ChatSession
        if err := rows.Scan(&session.ID, &session.UserID, &session.ModelID, &session.Title, &session.Status, &session.CreatedAt, &session.UpdatedAt); err == nil {
            sessions = append(sessions, session)
        }
    }
    return sessions
}
```

- [ ] **Step 2: 实现 ChatMessage CRUD**

```go
func (s *PostgresStore) AppendChatMessage(msg ChatMessage) ChatMessage {
    msg.ID = defaultString(msg.ID, "msg-"+time.Now().UTC().Format("20060102150405.000000000"))
    msg.CreatedAt = time.Now().UTC()
    s.db.Exec(`INSERT INTO chat_messages (id, session_id, role, content, tool_name, tool_args_json, tool_result_json, created_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (id) DO NOTHING`,
        msg.ID, msg.SessionID, msg.Role, msg.Content, msg.ToolName, msg.ToolArgsJSON, msg.ToolResultJSON, msg.CreatedAt)
    return msg
}

func (s *PostgresStore) ListSessionMessages(sessionID string) []ChatMessage {
    rows, err := s.db.Query(`SELECT id, session_id, role, content, tool_name, tool_args_json, tool_result_json, created_at
        FROM chat_messages WHERE session_id=$1 ORDER BY created_at, id`, sessionID)
    if err != nil {
        return nil
    }
    defer rows.Close()
    var msgs []ChatMessage
    for rows.Next() {
        var msg ChatMessage
        if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &msg.ToolName, &msg.ToolArgsJSON, &msg.ToolResultJSON, &msg.CreatedAt); err == nil {
            msgs = append(msgs, msg)
        }
    }
    return msgs
}
```

- [ ] **Step 3: 实现 LLMBinding CRUD**

```go
func (s *PostgresStore) SetUserLLMBindings(userID string, modelIDs []string, defaultModelID string) []UserLLMBinding {
    tx, err := s.db.Begin()
    if err != nil {
        return nil
    }
    defer tx.Rollback()
    tx.Exec(`DELETE FROM user_llm_bindings WHERE user_id=$1`, userID)
    now := time.Now().UTC()
    for _, modelID := range modelIDs {
        tx.Exec(`INSERT INTO user_llm_bindings (id, user_id, model_id, is_default, created_by, created_at)
            VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (id) DO NOTHING`,
            userID+"-"+modelID, userID, modelID, modelID == defaultModelID, userID, now)
    }
    tx.Commit()
    return s.GetUserLLMBindings(userID)
}

func (s *PostgresStore) GetUserLLMBindings(userID string) []UserLLMBinding {
    rows, err := s.db.Query(`SELECT id, user_id, model_id, is_default, created_by, created_at FROM user_llm_bindings WHERE user_id=$1`, userID)
    if err != nil {
        return nil
    }
    defer rows.Close()
    var bindings []UserLLMBinding
    for rows.Next() {
        var b UserLLMBinding
        if err := rows.Scan(&b.ID, &b.UserID, &b.ModelID, &b.IsDefault, &b.CreatedBy, &b.CreatedAt); err == nil {
            bindings = append(bindings, b)
        }
    }
    return bindings
}
```

- [ ] **Step 4: 实现 ServiceAccount 方法**

```go
func (s *PostgresStore) SaveServiceAccountBinding(binding ServiceAccountBinding) ServiceAccountBinding {
    now := time.Now().UTC()
    binding.ID = defaultString(binding.ID, "sab-"+binding.UserID+"-"+binding.Namespace)
    binding.CreatedAt = now
    binding.UpdatedAt = now
    if binding.Status == "" {
        binding.Status = "active"
    }
    s.db.Exec(`INSERT INTO k8s_service_accounts (id, user_id, namespace, service_account_name, token_secret_name, status, created_at, updated_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (id) DO UPDATE SET status=$6, updated_at=$8`,
        binding.ID, binding.UserID, binding.Namespace, binding.ServiceAccountName, binding.TokenSecretName, binding.Status, binding.CreatedAt, binding.UpdatedAt)
    return binding
}

func (s *PostgresStore) GetUserServiceAccountBindings(userID string) []ServiceAccountBinding {
    rows, err := s.db.Query(`SELECT id, user_id, namespace, service_account_name, token_secret_name, status, created_at, updated_at
        FROM k8s_service_accounts WHERE user_id=$1`, userID)
    if err != nil {
        return nil
    }
    defer rows.Close()
    var bindings []ServiceAccountBinding
    for rows.Next() {
        var b ServiceAccountBinding
        if err := rows.Scan(&b.ID, &b.UserID, &b.Namespace, &b.ServiceAccountName, &b.TokenSecretName, &b.Status, &b.CreatedAt, &b.UpdatedAt); err == nil {
            bindings = append(bindings, b)
        }
    }
    return bindings
}

func (s *PostgresStore) GetServiceAccountToken(userID string) (string, string, error) {
    // 从 k8s_service_accounts 查询关联的 Secret token
    // 注意：实际 token 存储在 K8S Secret 中，这里返回占位
    // 真实实现在阶段三通过 K8S API 读取
    return "", "", fmt.Errorf("not implemented: use real K8S token lookup")
}
```

- [ ] **Step 5: Build and check**

Run: `cd backend && go build ./...`
Expected: Build succeeds

- [ ] **Step 6: Commit**

```bash
git add backend/internal/store/postgres.go
git commit -m "feat(store): implement new Store methods in PostgresStore for chat, LLM bindings, SA bindings"

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

---

## 阶段二：LLM Provider + MCP Client + 授权校验

### Task 2.1: LLM Provider 接口重构

**Files:**
- Modify: `backend/internal/llm/provider.go`
- Test: `backend/internal/llm/provider_test.go`

- [ ] **Step 1: 重构 provider.go 为完整接口**

```go
package llm

import "context"

type Role string

const (
    RoleUser      Role = "user"
    RoleAssistant Role = "assistant"
    RoleTool      Role = "tool"
)

type Protocol string

const (
    ProtocolOpenAI    Protocol = "openai"
    ProtocolAnthropic Protocol = "anthropic"
    ProtocolMock      Protocol = "mock"
)

type Message struct {
    Role    Role   `json:"role"`
    Content string `json:"content"`
}

type ToolDefinition struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    InputSchema any    `json:"inputSchema"`
}

type ToolCall struct {
    ID        string         `json:"id"`
    Name      string         `json:"name"`
    Arguments map[string]any `json:"arguments"`
}

type ChatRequest struct {
    Model    string
    System   string
    Messages []Message
    Tools    []ToolDefinition
}

type ChatResponse struct {
    Content  string
    ToolCall *ToolCall
}

type Provider interface {
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
}

type ProviderConfig struct {
    ID       string
    Name     string
    Protocol Protocol
    BaseURL  string
    APIKey   string
    Enabled  bool
}

type ModelBinding struct {
    ModelID           string
    DisplayName       string
    ProviderID        string
    IsDefault         bool
    SupportsTools     bool
    SupportsStreaming bool
}

func SelectDefaultModel(models []ModelBinding) (ModelBinding, bool) {
    if len(models) == 0 {
        return ModelBinding{}, false
    }
    for _, model := range models {
        if model.IsDefault {
            return model, true
        }
    }
    return models[0], true
}
```

- [ ] **Step 2: Update tests to match new interface**

```go
// backend/internal/llm/provider_test.go
package llm

import "testing"

func TestSelectDefaultModelReturnsExplicitDefault(t *testing.T) {
    models := []ModelBinding{
        {ModelID: "claude", DisplayName: "Claude", IsDefault: false},
        {ModelID: "gpt", DisplayName: "GPT", IsDefault: true},
    }
    model, ok := SelectDefaultModel(models)
    if !ok { t.Fatal("expected a default model") }
    if model.ModelID != "gpt" { t.Fatalf("expected gpt, got %s", model.ModelID) }
}

func TestSelectDefaultModelFallsBackToFirst(t *testing.T) {
    models := []ModelBinding{
        {ModelID: "claude", DisplayName: "Claude"},
        {ModelID: "gpt", DisplayName: "GPT"},
    }
    model, ok := SelectDefaultModel(models)
    if !ok { t.Fatal("expected first model fallback") }
    if model.ModelID != "claude" { t.Fatalf("expected claude, got %s", model.ModelID) }
}
```

- [ ] **Step 3: Run tests**

Run: `cd backend && go test ./internal/llm -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/llm/
git commit -m "refactor(llm): extract Provider interface, Message/ToolCall types, preserve SelectDefaultModel"

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

### Task 2.2: OpenAI Provider 实现

**Files:**
- Create: `backend/internal/llm/openai.go`
- Test: `backend/internal/llm/openai_test.go`

- [ ] **Step 1: 实现 OpenAI provider**

```go
package llm

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type openaiProvider struct {
    baseURL string
    apiKey  string
    client  *http.Client
}

func NewOpenAIProvider(baseURL, apiKey string) Provider {
    return &openaiProvider{
        baseURL: baseURL,
        apiKey:  apiKey,
        client:  &http.Client{Timeout: 120 * time.Second},
    }
}

type openaiMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type openaiTool struct {
    Type     string `json:"type"`
    Function struct {
        Name        string `json:"name"`
        Description string `json:"description"`
        Parameters  any    `json:"parameters"`
    } `json:"function"`
}

type openaiToolCall struct {
    ID       string `json:"id"`
    Type     string `json:"type"`
    Function struct {
        Name      string `json:"name"`
        Arguments string `json:"arguments"`
    } `json:"function"`
}

type openaiChoice struct {
    Message struct {
        Role      string           `json:"role"`
        Content   string           `json:"content"`
        ToolCalls []openaiToolCall `json:"tool_calls"`
    } `json:"message"`
}

type openaiResponse struct {
    Choices []openaiChoice `json:"choices"`
}

func (p *openaiProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    messages := []openaiMessage{}
    if req.System != "" {
        messages = append(messages, openaiMessage{Role: "system", Content: req.System})
    }
    for _, msg := range req.Messages {
        messages = append(messages, openaiMessage{Role: string(msg.Role), Content: msg.Content})
    }

    body := map[string]any{
        "model":       req.Model,
        "messages":    messages,
        "temperature": 0.1,
    }

    if len(req.Tools) > 0 {
        tools := make([]openaiTool, len(req.Tools))
        for i, t := range req.Tools {
            tools[i] = openaiTool{
                Type: "function",
            }
            tools[i].Function.Name = t.Name
            tools[i].Function.Description = t.Description
            if t.InputSchema != nil {
                tools[i].Function.Parameters = t.InputSchema
            } else {
                tools[i].Function.Parameters = map[string]any{
                    "type": "object",
                    "properties": map[string]any{},
                }
            }
        }
        body["tools"] = tools
    }

    payload, _ := json.Marshal(body)
    httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(payload))
    if err != nil {
        return nil, fmt.Errorf("openai: create request: %w", err)
    }
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

    resp, err := p.client.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("openai: do request: %w", err)
    }
    defer resp.Body.Close()

    respBody, _ := io.ReadAll(resp.Body)
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("openai: status=%d body=%s", resp.StatusCode, string(respBody))
    }

    var result openaiResponse
    if err := json.Unmarshal(respBody, &result); err != nil {
        return nil, fmt.Errorf("openai: decode response: %w", err)
    }
    if len(result.Choices) == 0 {
        return nil, fmt.Errorf("openai: no choices in response")
    }

    choice := result.Choices[0]
    chatResp := &ChatResponse{Content: choice.Message.Content}

    if len(choice.Message.ToolCalls) > 0 {
        tc := choice.Message.ToolCalls[0]
        var args map[string]any
        json.Unmarshal([]byte(tc.Function.Arguments), &args)
        chatResp.ToolCall = &ToolCall{
            ID:        tc.ID,
            Name:      tc.Function.Name,
            Arguments: args,
        }
    }

    return chatResp, nil
}
```

- [ ] **Step 2: Write OpenAI provider test（mock HTTP 验证）**

```go
package llm

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestOpenAIProviderChat(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{
            "choices": [{
                "message": {
                    "role": "assistant",
                    "content": "Hello from OpenAI"
                }
            }]
        }`))
    }))
    defer server.Close()

    provider := NewOpenAIProvider(server.URL, "test-key")
    resp, err := provider.Chat(context.Background(), ChatRequest{
        Model:    "gpt-4.1",
        Messages: []Message{{Role: RoleUser, Content: "hi"}},
    })
    if err != nil {
        t.Fatalf("chat: %v", err)
    }
    if resp.Content != "Hello from OpenAI" {
        t.Fatalf("unexpected content: %s", resp.Content)
    }
    if resp.ToolCall != nil {
        t.Fatal("expected no tool call")
    }
}

func TestOpenAIProviderToolCall(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{
            "choices": [{
                "message": {
                    "role": "assistant",
                    "content": "",
                    "tool_calls": [{
                        "id": "call_1",
                        "type": "function",
                        "function": {
                            "name": "list_pods",
                            "arguments": "{\"namespace\":\"dev\"}"
                        }
                    }]
                }
            }]
        }`))
    }))
    defer server.Close()

    provider := NewOpenAIProvider(server.URL, "test-key")
    resp, err := provider.Chat(context.Background(), ChatRequest{
        Model: "gpt-4.1",
        Tools: []ToolDefinition{{Name: "list_pods", Description: "List pods"}},
    })
    if err != nil {
        t.Fatalf("chat: %v", err)
    }
    if resp.ToolCall == nil || resp.ToolCall.Name != "list_pods" {
        t.Fatalf("expected tool call list_pods, got %#v", resp.ToolCall)
    }
    ns, ok := resp.ToolCall.Arguments["namespace"].(string)
    if !ok || ns != "dev" {
        t.Fatalf("expected namespace=dev, got %#v", resp.ToolCall.Arguments)
    }
}
```

- [ ] **Step 3: Run tests**

Run: `cd backend && go test ./internal/llm -run TestOpenAI -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/llm/openai.go backend/internal/llm/openai_test.go
git commit -m "feat(llm): implement OpenAI provider with tool call support"

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

### Task 2.3: Anthropic Provider 实现

**Files:**
- Create: `backend/internal/llm/anthropic.go`
- Test: `backend/internal/llm/anthropic_test.go`

- [ ] **Step 1: 实现 Anthropic provider**

```go
package llm

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type anthropicProvider struct {
    apiKey  string
    client  *http.Client
}

func NewAnthropicProvider(apiKey string) Provider {
    return &anthropicProvider{
        apiKey: apiKey,
        client: &http.Client{Timeout: 120 * time.Second},
    }
}

type anthropicContent struct {
    Type  string `json:"type"`
    Text  string `json:"text,omitempty"`
    ID    string `json:"id,omitempty"`
    Name  string `json:"name,omitempty"`
    Input any    `json:"input,omitempty"`
}

type anthropicToolUse struct {
    Type  string `json:"type"`
    Name  string `json:"name"`
    Input any    `json:"input"`
}

type anthropicMessage struct {
    Role    string             `json:"role"`
    Content []anthropicContent `json:"content"`
}

type anthropicResponse struct {
    ID      string             `json:"id"`
    Content []anthropicContent `json:"content"`
    StopReason string          `json:"stop_reason"`
}

func (p *anthropicProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    messages := []anthropicMessage{}
    for _, msg := range req.Messages {
        messages = append(messages, anthropicMessage{
            Role:    string(msg.Role),
            Content: []anthropicContent{{Type: "text", Text: msg.Content}},
        })
    }

    body := map[string]any{
        "model":      req.Model,
        "messages":   messages,
        "max_tokens": 4096,
    }

    if req.System != "" {
        body["system"] = req.System
    }

    if len(req.Tools) > 0 {
        tools := make([]map[string]any, len(req.Tools))
        for i, t := range req.Tools {
            inputSchema := t.InputSchema
            if inputSchema == nil {
                inputSchema = map[string]any{"type": "object", "properties": map[string]any{}}
            }
            tools[i] = map[string]any{
                "name":        t.Name,
                "description": t.Description,
                "input_schema": inputSchema,
            }
        }
        body["tools"] = tools
    }

    payload, _ := json.Marshal(body)
    httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(payload))
    if err != nil {
        return nil, fmt.Errorf("anthropic: create request: %w", err)
    }
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("x-api-key", p.apiKey)
    httpReq.Header.Set("anthropic-version", "2023-06-01")

    resp, err := p.client.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("anthropic: do request: %w", err)
    }
    defer resp.Body.Close()

    respBody, _ := io.ReadAll(resp.Body)
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("anthropic: status=%d body=%s", resp.StatusCode, string(respBody))
    }

    var result anthropicResponse
    if err := json.Unmarshal(respBody, &result); err != nil {
        return nil, fmt.Errorf("anthropic: decode response: %w", err)
    }

    chatResp := &ChatResponse{}
    for _, block := range result.Content {
        switch block.Type {
        case "text":
            chatResp.Content += block.Text
        case "tool_use":
            chatResp.ToolCall = &ToolCall{
                ID:        block.ID,
                Name:      block.Name,
                Arguments: block.Input.(map[string]any),
            }
        }
    }

    return chatResp, nil
}
```

- [ ] **Step 2: Write Anthropic provider test**

```go
package llm

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestAnthropicProviderChat(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{
            "id": "msg_1",
            "content": [{"type": "text", "text": "Hello from Claude"}],
            "stop_reason": "end_turn"
        }`))
    }))
    defer server.Close()

    provider := &anthropicProvider{apiKey: "test-key", client: &http.Client{}}
    // Override base URL for testing
    resp, err := provider.Chat(context.Background(), ChatRequest{
        Model: "claude-sonnet-4-6",
        Messages: []Message{{Role: RoleUser, Content: "hi"}},
    })
    if err != nil {
        t.Fatalf("chat: %v", err)
    }
    if resp.Content != "Hello from Claude" {
        t.Fatalf("unexpected content: %s", resp.Content)
    }
}
```

- [ ] **Step 3: Anthropic provider 需要在代码中将 baseURL 改为可配置**  
  修改：anthropicProvider 中增加 `baseURL` 字段，从环境变量读取（默认 `https://api.anthropic.com`）

- [ ] **Step 4: Run tests**

Run: `cd backend && go test ./internal/llm -run TestAnthropic -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/llm/anthropic.go backend/internal/llm/anthropic_test.go
git commit -m "feat(llm): implement Anthropic provider with tool call support"

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

### Task 2.4: Provider 注册中心

**Files:**
- Create: `backend/internal/llm/registry.go`
- Test: `backend/internal/llm/registry_test.go`

- [ ] **Step 1: 实现注册中心**

```go
package llm

import "fmt"

type Registry struct {
    providers map[string]Provider
}

func NewRegistry() *Registry {
    return &Registry{providers: map[string]Provider{}}
}

func (r *Registry) Register(id string, provider Provider) {
    r.providers[id] = provider
}

func (r *Registry) Get(id string) (Provider, error) {
    p, ok := r.providers[id]
    if !ok {
        return nil, fmt.Errorf("provider not found: %s", id)
    }
    return p, nil
}

func (r *Registry) CreateFromConfig(cfg ProviderConfig) Provider {
    switch cfg.Protocol {
    case ProtocolOpenAI:
        return NewOpenAIProvider(cfg.BaseURL, cfg.APIKey)
    case ProtocolAnthropic:
        return NewAnthropicProvider(cfg.APIKey) // baseURL hardcoded, can be overridden
    default:
        return nil
    }
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/llm/registry.go
git commit -m "feat(llm): add Provider registry for runtime provider creation"

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

### Task 2.5: API Key 加密

**Files:**
- Create: `backend/internal/crypto/cipher.go`
- Test: `backend/internal/crypto/cipher_test.go`

- [ ] **Step 1: 实现 AES-256-GCM 加解密**

```go
package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/hex"
    "errors"
    "io"
)

func Encrypt(plaintext string, key []byte) (string, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }
    aead, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    nonce := make([]byte, aead.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }
    ciphertext := aead.Seal(nonce, nonce, []byte(plaintext), nil)
    return hex.EncodeToString(ciphertext), nil
}

func Decrypt(cipherhex string, key []byte) (string, error) {
    ciphertext, err := hex.DecodeString(cipherhex)
    if err != nil {
        return "", err
    }
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }
    aead, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    nonceSize := aead.NonceSize()
    if len(ciphertext) < nonceSize {
        return "", errors.New("ciphertext too short")
    }
    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
    plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", err
    }
    return string(plaintext), nil
}
```

- [ ] **Step 2: 写测试**

```go
package crypto

import "testing"

func TestEncryptDecryptRoundTrip(t *testing.T) {
    key := []byte("01234567890123456789012345678901") // 32 bytes for AES-256
    secret := "sk-this-is-a-secret-api-key"

    encrypted, err := Encrypt(secret, key)
    if err != nil {
        t.Fatalf("encrypt: %v", err)
    }
    if encrypted == "" {
        t.Fatal("expected non-empty ciphertext")
    }

    decrypted, err := Decrypt(encrypted, key)
    if err != nil {
        t.Fatalf("decrypt: %v", err)
    }
    if decrypted != secret {
        t.Fatalf("round trip failed: %s != %s", decrypted, secret)
    }
}

func TestEncryptProducesDifferentOutput(t *testing.T) {
    key := []byte("01234567890123456789012345678901")
    secret := "same-secret"

    e1, _ := Encrypt(secret, key)
    e2, _ := Encrypt(secret, key)
    if e1 == e2 {
        t.Fatal("expected different ciphertext due to random nonce")
    }
}
```

- [ ] **Step 3: 修改 Provider 存储/读取时加解密**

修改 `backend/internal/store/memory.go` 的 `CreateLLMProvider`：

```go
func (s *MemoryStore) CreateLLMProvider(provider LLMProvider) LLMProvider {
    now := time.Now().UTC()
    provider.ID = defaultString(provider.ID, "provider-"+provider.Protocol)
    // 加密 API Key（简化：实际应由调用方传入加密后的值）
    // provider.APIKeyCiphertext = "encrypted:" + provider.Name
    provider.CreatedAt = now
    provider.UpdatedAt = now
    s.providers = append(s.providers, provider)
    return provider
}
```

注意：MemoryStore 中的加密标记在实际运行时由 HTTP 层调用 crypto.Encrypt 处理。PostgresStore 同理。

- [ ] **Step 4: Run tests**

Run: `cd backend && go test ./internal/crypto -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/crypto/
git commit -m "feat(crypto): add AES-256-GCM encrypt/decrypt for API key protection"

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

### Task 2.6: MCP Client

**Files:**
- Create: `backend/internal/mcpclient/client.go`
- Test: `backend/internal/mcpclient/client_test.go`

- [ ] **Step 1: 实现 MCP Client**

```go
package mcpclient

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type ToolRequest struct {
    UserID string         `json:"user_id"`
    Tool   string         `json:"tool"`
    Args   map[string]any `json:"args"`
}

type ToolResponse struct {
    Success bool            `json:"success"`
    Data    json.RawMessage `json:"data,omitempty"`
    Error   string          `json:"error,omitempty"`
}

type Client struct {
    baseURL    string
    httpClient *http.Client
}

func NewClient(baseURL string) *Client {
    return &Client{
        baseURL:    baseURL,
        httpClient: &http.Client{Timeout: 60 * time.Second},
    }
}

func (c *Client) CallTool(ctx context.Context, req ToolRequest) (*ToolResponse, error) {
    payload, _ := json.Marshal(req)
    httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/tools/"+req.Tool, bytes.NewReader(payload))
    if err != nil {
        return nil, fmt.Errorf("mcp: create request: %w", err)
    }
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("mcp: do request: %w", err)
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("mcp: status=%d body=%s", resp.StatusCode, string(body))
    }

    var result ToolResponse
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, fmt.Errorf("mcp: decode response: %w", err)
    }
    return &result, nil
}
```

- [ ] **Step 2: Write test**

```go
package mcpclient

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestClientCallTool(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"success":true,"data":{"pods":[]}}`))
    }))
    defer server.Close()

    client := NewClient(server.URL)
    resp, err := client.CallTool(context.Background(), ToolRequest{
        UserID: "u1",
        Tool:   "list_pods",
        Args:   map[string]any{"namespace": "dev"},
    })
    if err != nil {
        t.Fatalf("call tool: %v", err)
    }
    if !resp.Success {
        t.Fatal("expected success")
    }
    var data map[string]any
    if err := json.Unmarshal(resp.Data, &data); err != nil {
        t.Fatalf("parse data: %v", err)
    }
    pods, ok := data["pods"].([]any)
    if !ok {
        t.Fatal("expected pods array")
    }
    if len(pods) != 0 {
        t.Fatalf("expected empty pods, got %d", len(pods))
    }
}
```

- [ ] **Step 3: Run tests**

Run: `cd backend && go test ./internal/mcpclient -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/mcpclient/
git commit -m "feat(mcp): add MCP Client for Backend-to-MCP-Server HTTP calls"

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

### Task 2.7: Chat Service 重构（集成真实调用链）

**Files:**
- Modify: `backend/internal/http/router.go`
- Test: `backend/internal/http/router_test.go`

- [ ] **Step 1: 重构 createChatMessage 替换硬编码 mock**

```go
// 修改 Server 结构体，添加 LLM Registry 和 MCP Client 字段
type Server struct {
    store       store.Store
    models      []llm.ModelBinding
    pods        []chat.AbnormalPod
    rbacApplier RBACApplier
    llmRegistry *llm.Registry  // 新增
    mcpClient   *mcpclient.Client  // 新增
}

// 修改 createChatMessage 实现真实调用链
func (s *Server) createChatMessage(w http.ResponseWriter, r *http.Request) {
    var request chatMessageRequest
    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request body must be valid JSON.")
        return
    }
    if !s.isModelAllowed(request.ModelID) {
        s.audit("operator.chat.message.create", "chat_message", "", "", "", "", false, "model not allowed")
        writeError(w, http.StatusForbidden, "LLM_MODEL_NOT_ALLOWED", "The selected model is not assigned to the current user.")
        return
    }

    // 获取用户权限
    userID := s.store.CurrentDemoUser().ID
    permissions := s.store.ListUserPermissions(userID)

    // 构建系统提示词
    ctx := chat.UserContext{
        UserID:      userID,
        Username:    s.store.CurrentDemoUser().Username,
        Permissions: toChatPermissions(permissions),
    }
    systemPrompt := chat.BuildSystemPrompt(ctx)

    // 构造 LLM 请求
    llmReq := llm.ChatRequest{
        Model:    request.ModelID,
        System:   systemPrompt,
        Messages: []llm.Message{{Role: llm.RoleUser, Content: request.Content}},
        Tools:    getAvailableTools(),
    }

    // 调用 LLM
    provider, err := s.llmRegistry.Get(request.ModelID)
    if err != nil {
        s.audit("operator.chat.message", "chat_message", "", "", "", "", false, "llm provider unavailable")
        writeError(w, http.StatusServiceUnavailable, "LLM_PROVIDER_UNAVAILABLE", "The LLM provider is not available.")
        return
    }

    llmResp, err := provider.Chat(r.Context(), llmReq)
    if err != nil {
        s.audit("operator.chat.message", "chat_message", "", "", "", "", false, "llm call failed")
        writeError(w, http.StatusInternalServerError, "LLM_CALL_FAILED", "Failed to get response from LLM.")
        return
    }

    // 处理工具调用
    if llmResp.ToolCall != nil {
        toolReq := mapToolCallToRequest(userID, llmResp.ToolCall)
        if toolReq == nil {
            s.audit("operator.chat.tool", "chat_message", "", "", "", "", false, "unknown tool")
            writeError(w, http.StatusBadRequest, "UNKNOWN_TOOL", "The requested tool is not recognized.")
            return
        }

        // 授权校验
        toolCtx := chat.ToolRequest{
            Name:      toolReq.Tool,
            Namespace: getArg[string](toolReq.Args, "namespace"),
            Resource:  toolNameToResource(toolReq.Tool),
            Verb:      toolNameToVerb(toolReq.Tool),
        }
        if err := chat.AuthorizeTool(ctx, toolCtx); err != nil {
            s.audit("operator.chat.tool", "chat_message", "", toolCtx.Namespace, toolCtx.Resource, toolCtx.Verb, false, "permission denied")
            writeError(w, http.StatusForbidden, "K8S_PERMISSION_DENIED", "You do not have permission to perform this action.")
            return
        }

        // 调用 MCP
        mcpResp, err := s.mcpClient.CallTool(r.Context(), *toolReq)
        if err != nil {
            s.audit("operator.chat.tool", "chat_message", "", toolCtx.Namespace, toolCtx.Resource, toolCtx.Verb, false, "mcp tool failed")
            writeError(w, http.StatusInternalServerError, "MCP_TOOL_UNAVAILABLE", "The MCP tool is not available.")
            return
        }

        // 将工具结果送回 LLM 生成最终总结
        toolResultJSON, _ := json.Marshal(mcpResp.Data)
        llmReq.Messages = append(llmReq.Messages,
            llm.Message{Role: llm.RoleAssistant, Content: llmResp.Content},
            llm.Message{Role: llm.RoleTool, Content: string(toolResultJSON)},
        )
        finalResp, err := provider.Chat(r.Context(), llmReq)
        if err != nil {
            // MCP 工具结果有了，但 LLM 总结失败，直接返回工具结果
            writeJSON(w, http.StatusOK, map[string]any{
                "messageId": "msg-" + time.Now().UTC().Format("20060102150405"),
                "summary":   "Tool executed successfully, but failed to generate summary.",
                "resources": mcpResp.Data,
            })
            return
        }

        writeJSON(w, http.StatusOK, map[string]any{
            "messageId": "msg-" + time.Now().UTC().Format("20060102150405"),
            "summary":   finalResp.Content,
            "resources": mcpResp.Data,
        })
        s.audit("operator.chat.tool", "chat_message", "", toolCtx.Namespace, toolCtx.Resource, toolCtx.Verb, true, "tool executed")
        return
    }

    // 纯文本响应（无工具调用）
    writeJSON(w, http.StatusOK, map[string]any{
        "messageId": "msg-" + time.Now().UTC().Format("20060102150405"),
        "summary":   llmResp.Content,
        "resources": nil,
    })
    s.audit("operator.chat.message", "chat_message", "", "", "", "", true, "message processed")
}
```

- [ ] **Step 2: 添加辅助函数**

```go
// 将 store.Permission 转为 chat.Permission
func toChatPermissions(permissions []store.Permission) []chat.Permission {
    result := make([]chat.Permission, len(permissions))
    for i, p := range permissions {
        result[i] = chat.Permission{
            Namespace: p.Namespace,
            APIGroup:  p.APIGroup,
            Resource:  p.Resource,
            Verbs:     p.Verbs,
        }
    }
    return result
}

// 可用工具列表
func getAvailableTools() []llm.ToolDefinition {
    return []llm.ToolDefinition{
        {Name: "list_pods", Description: "List pods in a namespace"},
        {Name: "get_pod", Description: "Get pod details"},
        {Name: "get_pod_logs", Description: "Get pod logs"},
        {Name: "list_events", Description: "List events in a namespace"},
        {Name: "list_deployments", Description: "List deployments in a namespace"},
        {Name: "restart_deployment", Description: "Restart a deployment"},
        {Name: "list_namespaces", Description: "List accessible namespaces"},
    }
}

// 映射 ToolCall 到 MCP 请求
func mapToolCallToRequest(userID string, tc *llm.ToolCall) *mcpclient.ToolRequest {
    return &mcpclient.ToolRequest{
        UserID: userID,
        Tool:   tc.Name,
        Args:   tc.Arguments,
    }
}

// 泛型辅助：从 map 读取 string 类型参数
func getArg[T string](args map[string]any, key string) T {
    raw, ok := args[key]
    if !ok {
        return ""
    }
    val, _ := raw.(string)
    return T(val)
}

// 工具名到 Kubernetes resource 映射
func toolNameToResource(tool string) string {
    switch tool {
    case "list_pods", "get_pod":
        return "pods"
    case "get_pod_logs":
        return "pods/log"
    case "list_events":
        return "events"
    case "list_deployments", "restart_deployment":
        return "deployments"
    case "list_namespaces":
        return "namespaces"
    default:
        return ""
    }
}

func toolNameToVerb(tool string) string {
    switch tool {
    case "list_pods", "list_events", "list_deployments", "list_namespaces":
        return "list"
    case "get_pod", "get_pod_logs":
        return "get"
    case "restart_deployment":
        return "patch"
    default:
        return ""
    }
}
```

- [ ] **Step 3: 修改 NewServer 注入 LLM Registry 和 MCP Client**

```go
func NewServer(stores ...store.Store) *Server {
    activeStore := store.Store(store.NewMemoryStore())
    if len(stores) > 0 && stores[0] != nil {
        activeStore = stores[0]
    }

    // 初始化 LLM Registry 和 MCP Client
    reg := llm.NewRegistry()
    // 注册 mock provider 用于开发测试
    reg.Register("mock-local", llm.NewOpenAIProvider("http://localhost:9999", "mock"))

    return &Server{
        store:       activeStore,
        llmRegistry: reg,
        mcpClient:   mcpclient.NewClient("http://localhost:8081"),
        models: []llm.ModelBinding{
            {ModelID: "mock-local", DisplayName: "Mock Local", ProviderID: "mock", IsDefault: true, SupportsTools: true},
        },
        pods: []chat.AbnormalPod{
            {Namespace: "dev", Name: "api-7b8f9", Phase: "Pending", Reason: "ImagePullBackOff", Message: "Back-off pulling image", RestartCount: 0, Node: "kind-worker"},
            {Namespace: "dev", Name: "worker-5d9c7", Phase: "Running", Reason: "CrashLoopBackOff", Message: "Container exits after startup", RestartCount: 6, Node: "kind-worker2"},
        },
    }
}
```

- [ ] **Step 4: Run existing tests to verify they still pass**

Run: `cd backend && go test ./internal/http -v`
Expected: Existing tests PASS, some tests may need mock provider update

- [ ] **Step 5: Commit**

```bash
git add backend/internal/http/router.go backend/internal/http/router_test.go
git commit -m "feat(chat): integrate real LLM provider, authorization check, and MCP client into chat message flow"

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

---

## 阶段三：MCP Server 真实 K8S 集成 + gRPC RPC

### Task 3.1: 定义 gRPC proto 并生成代码

**Files:**
- Create: `backend/api/auth.proto`

- [ ] **Step 1: 编写 proto 文件**

```protobuf
syntax = "proto3";
package k8sauth;
option go_package = "k8s-ai-ops/backend/api;api";

service K8SAuthService {
  rpc GetServiceAccountToken (TokenRequest) returns (TokenResponse);
}

message TokenRequest {
  string user_id = 1;
}

message TokenResponse {
  string token = 1;
  string namespace = 2;
}
```

- [ ] **Step 2: 安装 protoc 和 Go 插件并生成代码**

Run:
```bash
cd backend
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  api/auth.proto
```

Expected: `backend/api/auth.pb.go` 和 `backend/api/auth_grpc.pb.go` 被生成

- [ ] **Step 3: 添加 go 1.26 build tag 兼容（如有需要）**

- [ ] **Step 4: Commit**

```bash
git add backend/api/
git commit -m "feat(api): add gRPC auth proto and generated Go code"

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

### Task 3.2: Backend gRPC Server

**Files:**
- Create: `backend/internal/rpc/server.go`
- Modify: `backend/cmd/api/main.go`

- [ ] **Step 1: 实现 gRPC 服务端**

```go
package rpc

import (
    "context"
    "fmt"
    "net"

    "google.golang.org/grpc"
    "k8s-ai-ops/backend/api"
    "k8s-ai-ops/backend/internal/store"
)

type K8SAuthServer struct {
    api.UnimplementedK8SAuthServiceServer
    store store.Store
}

func NewK8SAuthServer(s store.Store) *K8SAuthServer {
    return &K8SAuthServer{store: s}
}

func (s *K8SAuthServer) GetServiceAccountToken(ctx context.Context, req *api.TokenRequest) (*api.TokenResponse, error) {
    token, namespace, err := s.store.GetServiceAccountToken(req.UserId)
    if err != nil {
        return nil, fmt.Errorf("get token for user %s: %w", req.UserId, err)
    }
    return &api.TokenResponse{
        Token:     token,
        Namespace: namespace,
    }, nil
}

func StartGRPCServer(addr string, store store.Store) (*grpc.Server, error) {
    lis, err := net.Listen("tcp", addr)
    if err != nil {
        return nil, fmt.Errorf("grpc listen: %w", err)
    }
    srv := grpc.NewServer()
    api.RegisterK8SAuthServiceServer(srv, NewK8AuthServer(store))
    go srv.Serve(lis)
    return srv, nil
}
```

- [ ] **Step 2: 修改 main.go 同时启动 HTTP + gRPC**

```go
func main() {
    cfg := config.Load()

    // ... 初始化 store, redis, k8s client（已有代码）...

    httpServer := apihttp.NewServer(appStore)
    if cfg.K8SRBACSyncEnabled {
        client, err := k8sops.NewClientset(cfg.Kubeconfig)
        if err != nil {
            log.Fatalf("level=ERROR component=backend-api event=k8s_client_create_failed error=%q", err)
        }
        httpServer.SetRBACApplier(k8sops.NewRBACManager(client))
    }

    // 启动 gRPC 服务
    grpcSrv, err := rpc.StartGRPCServer(cfg.GRPCAddr, appStore)
    if err != nil {
        log.Fatalf("level=ERROR component=backend-api event=grpc_server_start_failed error=%q", err)
    }
    defer grpcSrv.Stop()
    log.Printf("level=INFO component=backend-api event=grpc_server_start addr=%s", cfg.GRPCAddr)

    log.Printf("level=INFO component=backend-api event=http_server_start addr=%s", cfg.HTTPAddr)
    if err := http.ListenAndServe(cfg.HTTPAddr, httpServer); err != nil {
        log.Fatalf("level=ERROR component=backend-api event=server_exit error=%q", err)
    }
}
```

- [ ] **Step 3: 添加 GRPCAddr 到 config**

```go
// backend/internal/config/config.go 新增 GRPCAddr
type Config struct {
    HTTPAddr           string
    GRPCAddr           string  // 新增
    // ... 已有字段
}

func Load() Config {
    return Config{
        HTTPAddr:           env("HTTP_ADDR", ":8080"),
        GRPCAddr:           env("GRPC_ADDR", ":8082"),  // 新增
        // ... 已有配置
    }
}
```

- [ ] **Step 4: Build and check**

Run: `cd backend && go build ./...`
Expected: Build succeeds

- [ ] **Step 5: Commit**

```bash
git add backend/internal/rpc/ backend/internal/config/ backend/cmd/api/ backend/api/
git commit -m "feat(rpc): add gRPC server for K8S auth token lookup alongside HTTP"

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

### Task 3.3: MCP Server K8S Client 真实实现

**Files:**
- Modify: `mcp-server/internal/k8s/client.go`
- Modify: `mcp-server/go.mod`

- [ ] **Step 1: 重写 client.go 使用 client-go 和 Bearer Token**

```go
package k8s

import (
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
)

func NewClientForToken(token string) (kubernetes.Interface, error) {
    config := &rest.Config{
        Host:        "https://kubernetes.default.svc",
        BearerToken: token,
        TLSClientConfig: rest.TLSClientConfig{
            Insecure: false,
        },
    }
    client, err := kubernetes.NewForConfig(config)
    if err != nil {
        return nil, err
    }
    return client, nil
}
```

- [ ] **Step 2: 添加 go.mod 依赖**

Run: `cd mcp-server && go mod init k8s-ai-ops/mcp-server 2>/dev/null; go get k8s.io/client-go@v0.34.2 k8s.io/api@v0.34.2 k8s.io/apimachinery@v0.34.2 google.golang.org/grpc@v1.71.0 google.golang.org/protobuf@v1.36.6`

- [ ] **Step 3: Build**

Run: `cd mcp-server && go build ./...`
Expected: Build succeeds

- [ ] **Step 4: Commit**

```bash
git add mcp-server/internal/k8s/ mcp-server/go.mod mcp-server/go.sum
git commit -m "feat(mcp): implement real K8S client using BearerToken for per-request auth"

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

### Task 3.4: MCP Server gRPC Client

**Files:**
- Create: `mcp-server/internal/rpc/client.go`

- [ ] **Step 1: 实现 gRPC Client**

```go
package rpc

import (
    "fmt"
    "sync"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "k8s-ai-ops/backend/api" // 注意：需要在 mcp-server 中引用 backend 的 proto 包
)

// 更实用的做法：将 proto 文件复制到 mcp-server 共用，或单独建 shared 包
// 简化方案：MCP Server 通过 HTTP 调用 Backend REST API 获取 token
// 这里先声明接口，实际使用 HTTP 兜底

type TokenProvider struct {
    grpcAddr string
    httpAddr string
    conn     *grpc.ClientConn
    client   api.K8SAuthServiceClient
    mu       sync.RWMutex
}

func NewTokenProvider(grpcAddr, httpAddr string) *TokenProvider {
    return &TokenProvider{grpcAddr: grpcAddr, httpAddr: httpAddr}
}

func (p *TokenProvider) GetToken(userID string) (string, string, error) {
    // 尝试 gRPC
    token, ns, err := p.grpcGetToken(userID)
    if err == nil {
        return token, ns, nil
    }
    // 降级到 HTTP
    return p.httpGetToken(userID)
}

func (p *TokenProvider) grpcGetToken(userID string) (string, string, error) {
    p.mu.RLock()
    if p.client == nil {
        p.mu.RUnlock()
        p.mu.Lock()
        conn, err := grpc.Dial(p.grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
        if err != nil {
            p.mu.Unlock()
            return "", "", fmt.Errorf("grpc dial: %w", err)
        }
        p.conn = conn
        p.client = api.NewK8SAuthServiceClient(conn)
        p.mu.Unlock()
    } else {
        p.mu.RUnlock()
    }

    resp, err := p.client.GetServiceAccountToken(nil, &api.TokenRequest{UserId: userID})
    if err != nil {
        return "", "", fmt.Errorf("grpc get token: %w", err)
    }
    return resp.Token, resp.Namespace, nil
}

func (p *TokenProvider) httpGetToken(userID string) (string, string, error) {
    // HTTP 降级实现（略）
    return "", "", fmt.Errorf("http fallback not implemented")
}
```

- [ ] **Step 2: 将 proto 文件复制到 MCP Server 项目**

```bash
mkdir -p mcp-server/api
cp backend/api/auth.proto mcp-server/api/
# 修改 option go_package 为 mcp-server 的 module 路径
# 或者在 mcp-server 中直接引用 backend 的包（不推荐，会引入循环依赖）
```

最佳实践：proto 包单独拆分。但为简化，MCP Server 自己维护一份 proto 副本，或者直接用 HTTP REST 查询 Backend token（更简单）。

**简化方案决定：** MCP Server 通过 HTTP REST 调用 Backend 获取 token，而不是 gRPC。这样避免了 proto 跨项目共享的复杂度。

```go
// 简化版：MCP Server 通过 HTTP 调用 Backend 内部接口
func (p *TokenProvider) GetToken(userID string) (string, string, error) {
    resp, err := http.Get(fmt.Sprintf("%s/internal/rpc/k8s-token/%s", p.httpAddr, userID))
    if err != nil {
        return "", "", err
    }
    defer resp.Body.Close()
    var result struct {
        Token     string `json:"token"`
        Namespace string `json:"namespace"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", "", err
    }
    return result.Token, result.Namespace, nil
}
```

或者保留 gRPC 方案：把 proto 生成的代码单独放到一个共享包（如 `pkg/api/`）中，Backend 和 MCP Server 都引用它。

- [ ] **Step 3: Commit**

选择方案后提交。这里推荐使用 gRPC + 共享 proto 代码的方案：

```bash
mkdir -p pkg/api
# proto 文件和生成的 .pb.go 放在 pkg/api/
# backend 和 mcp-server 都通过 replace directive 或 symlink 引用
```

### Task 3.5: 实现 MCP 工具

**Files:**
- Create: `mcp-server/internal/tools/list_pods.go`
- Create: `mcp-server/internal/tools/get_pod.go`
- Create: `mcp-server/internal/tools/get_pod_logs.go`
- Create: `mcp-server/internal/tools/list_events.go`
- Create: `mcp-server/internal/tools/list_deployments.go`
- Create: `mcp-server/internal/tools/restart_deployment.go`
- Create: `mcp-server/internal/tools/list_namespaces.go`
- Modify: `mcp-server/cmd/server/main.go`

- [ ] **Step 1: 实现 list_pods 工具**

```go
package tools

import (
    "context"
    "encoding/json"

    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
)

type ListPodsInput struct {
    Namespace string `json:"namespace"`
}

type PodResult struct {
    Namespace    string `json:"namespace"`
    Name         string `json:"name"`
    Phase        string `json:"phase"`
    Reason       string `json:"reason"`
    Message      string `json:"message"`
    RestartCount int    `json:"restartCount"`
    Node         string `json:"node"`
    PodIP        string `json:"podIP,omitempty"`
}

func ListPods(ctx context.Context, client kubernetes.Interface, input json.RawMessage) (any, error) {
    var params ListPodsInput
    if err := json.Unmarshal(input, &params); err != nil {
        return nil, err
    }

    podList, err := client.CoreV1().Pods(params.Namespace).List(ctx, metav1.ListOptions{})
    if err != nil {
        return nil, err
    }

    pods := make([]PodResult, 0, len(podList.Items))
    for _, pod := range podList.Items {
        reason := string(pod.Status.Phase)
        message := ""
        var restartCount int32
        if len(pod.Status.Conditions) > 0 {
            message = pod.Status.Conditions[len(pod.Status.Conditions)-1].Message
        }
        for _, containerStatus := range pod.Status.ContainerStatuses {
            restartCount += containerStatus.RestartCount
            if containerStatus.State.Waiting != nil {
                reason = containerStatus.State.Waiting.Reason
                message = containerStatus.State.Waiting.Message
            }
        }

        pods = append(pods, PodResult{
            Namespace:    pod.Namespace,
            Name:         pod.Name,
            Phase:        string(pod.Status.Phase),
            Reason:       reason,
            Message:      message,
            RestartCount: int(restartCount),
            Node:         pod.Spec.NodeName,
            PodIP:        pod.Status.PodIP,
        })
    }
    return map[string]any{"pods": pods}, nil
}
```

- [ ] **Step 2: 实现 get_pod 工具**

```go
package tools

import (
    "context"
    "encoding/json"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
)

type GetPodInput struct {
    Namespace string `json:"namespace"`
    Name      string `json:"name"`
}

func GetPod(ctx context.Context, client kubernetes.Interface, input json.RawMessage) (any, error) {
    var params GetPodInput
    if err := json.Unmarshal(input, &params); err != nil {
        return nil, err
    }
    pod, err := client.CoreV1().Pods(params.Namespace).Get(ctx, params.Name, metav1.GetOptions{})
    if err != nil {
        return nil, err
    }
    // 简化为 PodResult 结构
    return PodResult{
        Namespace: pod.Namespace, Name: pod.Name, Phase: string(pod.Status.Phase),
        Node: pod.Spec.NodeName, PodIP: pod.Status.PodIP,
    }, nil
}
```

- [ ] **Step 3: 实现 get_pod_logs 工具**

```go
package tools

import (
    "context"
    "encoding/json"
    "io"

    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
)

type GetPodLogsInput struct {
    Namespace string `json:"namespace"`
    Name      string `json:"name"`
    TailLines *int64 `json:"tailLines,omitempty"`
}

func GetPodLogs(ctx context.Context, client kubernetes.Interface, input json.RawMessage) (any, error) {
    var params GetPodLogsInput
    if err := json.Unmarshal(input, &params); err != nil {
        return nil, err
    }
    tail := int64(100)
    if params.TailLines != nil && *params.TailLines > 0 {
        tail = *params.TailLines
    }
    logOpts := &corev1.PodLogOptions{TailLines: &tail}
    req := client.CoreV1().Pods(params.Namespace).GetLogs(params.Name, logOpts)
    stream, err := req.Stream(ctx)
    if err != nil {
        return nil, err
    }
    defer stream.Close()
    logs, _ := io.ReadAll(stream)
    return map[string]any{"logs": string(logs), "truncated": len(logs) > 10000}, nil
}
```

- [ ] **Step 4: 实现 list_events、list_deployments、restart_deployment、list_namespaces 工具**（结构与 list_pods 类似）

- [ ] **Step 5: 重写 MCP Server main.go 注册所有工具并集成 gRPC Token 查询**

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "os"
    "strings"

    "k8s.io/client-go/kubernetes"
    "mcp-server/internal/k8s"
    mcpRpc "mcp-server/internal/rpc"
    "mcp-server/internal/tools"
)

type toolRequest struct {
    UserID string          `json:"user_id"`
    Tool   string          `json:"tool"`
    Args   json.RawMessage `json:"args"`
}

type toolHandler func(ctx context.Context, client kubernetes.Interface, args json.RawMessage) (any, error)

var toolHandlers = map[string]toolHandler{
    "list_pods":         tools.ListPods,
    "get_pod":           tools.GetPod,
    "get_pod_logs":      tools.GetPodLogs,
    "list_events":       tools.ListEvents,
    "list_deployments":  tools.ListDeployments,
    "restart_deployment": tools.RestartDeployment,
    "list_namespaces":   tools.ListNamespaces,
}

func main() {
    addr := env("HTTP_ADDR", ":8081")
    backendGRPC := env("BACKEND_GRPC_ADDR", "backend-api:8082")
    backendHTTP := env("BACKEND_HTTP_ADDR", "http://backend-api:8080")

    tokenProvider := mcpRpc.NewTokenProvider(backendGRPC, backendHTTP)

    mux := http.NewServeMux()
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
    })
    mux.HandleFunc("/tools/", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
            return
        }
        toolName := strings.TrimPrefix(r.URL.Path, "/tools/")
        handler, ok := toolHandlers[toolName]
        if !ok {
            writeResult(w, false, nil, "unknown tool: "+toolName)
            return
        }

        var req toolRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            writeResult(w, false, nil, "invalid request: "+err.Error())
            return
        }

        // 通过 gRPC 查询用户 SA Token
        token, _, err := tokenProvider.GetToken(req.UserID)
        if err != nil {
            writeResult(w, false, nil, "auth failed: "+err.Error())
            return
        }

        // 创建 K8S client
        client, err := k8s.NewClientForToken(token)
        if err != nil {
            writeResult(w, false, nil, "k8s client failed: "+err.Error())
            return
        }

        result, err := handler(r.Context(), client, req.Args)
        if err != nil {
            writeResult(w, false, nil, "tool error: "+err.Error())
            return
        }
        writeResult(w, true, result, "")
    })

    log.Printf("level=INFO component=mcp-server event=server_start addr=%s", addr)
    if err := http.ListenAndServe(addr, mux); err != nil {
        log.Fatalf("level=ERROR component=mcp-server event=server_exit error=%q", err)
    }
}

func writeResult(w http.ResponseWriter, success bool, data any, errMsg string) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{
        "success": success,
        "data":    data,
        "error":   errMsg,
    })
}

func env(key, fallback string) string {
    v := os.Getenv(key)
    if v == "" { return fallback }
    return v
}
```

- [ ] **Step 6: Build and test**

Run: `cd mcp-server && go build ./... && go test ./...`
Expected: All build and tests pass

- [ ] **Step 7: Commit**

```bash
git add mcp-server/
git commit -m "feat(mcp): implement all K8S tools (list/get pods, logs, events, deployments) with gRPC token lookup"

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

### Task 3.6: 内置管理员 SA（Helm 模板）

**Files:**
- Create: `deploy/helm/k8s-ai-ops/charts/k8s-ai-backend/templates/admin-serviceaccount.yaml`

- [ ] **Step 1: 创建 admin SA 模板**

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: k8s-ai-admin
  namespace: {{ .Values.global.namespace }}
---
apiVersion: v1
kind: Secret
metadata:
  name: k8s-ai-admin-token
  annotations:
    kubernetes.io/service-account.name: k8s-ai-admin
type: kubernetes.io/service-account-token
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: k8s-ai-admin
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: k8s-ai-admin
    namespace: {{ .Values.global.namespace }}
```

- [ ] **Step 2: 在 SeedDemoData 中添加 admin 用户**

```go
// backend/internal/store/postgres.go - SeedDemoData
func (s *PostgresStore) SeedDemoData(ctx context.Context) error {
    now := time.Now().UTC()
    // 创建 admin 用户
    adminPassHash := "$2a$10$..."  // bcrypt hash of "123456"
    s.db.Exec(`INSERT INTO users (id, keycloak_user_id, username, display_name, email, role, status, created_at, updated_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (id) DO NOTHING`,
        "admin-user", "", "admin", "System Admin", "admin@k8s-ai.local", RoleAdmin, "active", now, now)
    // 已有 demo 用户
    // ...
}
```

注意：bcrypt 密码需要在 Go 代码中使用 `golang.org/x/crypto/bcrypt` 生成。

- [ ] **Step 3: 添加 VerifyPassword 到 Store**

```go
func (s *PostgresStore) VerifyPassword(username, password string) (User, bool) {
    // 先查询用户
    row := s.db.QueryRow(`SELECT id, username, role, status FROM users WHERE username=$1 AND status='active'`, username)
    var user User
    var passwordHash string
    // 简化：从环境变量或配置校验
    // 真实场景应在 users 表加 password_hash 字段，或通过 Keycloak 认证
    if username == "admin" && password == "123456" {
        user = User{ID: "admin-user", Username: "admin", Role: RoleAdmin, Status: "active"}
        return user, true
    }
    if username == "demo" && password == "demo123" {
        user = User{ID: "demo-user", Username: "demo", Role: RoleOperator, Status: "active"}
        return user, true
    }
    return User{}, false
}
```

- [ ] **Step 4: Commit**

```bash
git add deploy/helm/k8s-ai-ops/charts/k8s-ai-backend/templates/ backend/internal/store/postgres.go
git commit -m "feat(auth): add built-in admin SA and user admin/123456 with cluster-admin permissions"

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

---

## 阶段四：Keycloak 认证

### Task 4.1: JWT 校验中间件

**Files:**
- Create: `backend/internal/auth/jwt.go`
- Modify: `backend/internal/http/router.go`
- Modify: `backend/go.mod`

- [ ] **Step 1: 实现 JWT 中间件**

```go
package auth

import (
    "context"
    "net/http"
    "strings"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
    ContextUserID contextKey = "userID"
    ContextRole   contextKey = "role"
)

type JWTMiddleware struct {
    issuer    string
    jwksURL   string
    disabled  bool
}

func NewJWTMiddleware(issuer string, disabled bool) *JWTMiddleware {
    return &JWTMiddleware{
        issuer:   issuer,
        jwksURL:  issuer + "/protocol/openid-connect/certs",
        disabled: disabled,
    }
}

func (m *JWTMiddleware) Wrap(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if m.disabled {
            // 开发模式：设置默认用户
            ctx := context.WithValue(r.Context(), ContextUserID, "demo-user")
            ctx = context.WithValue(ctx, ContextRole, "admin")
            next.ServeHTTP(w, r.WithContext(ctx))
            return
        }

        authHeader := r.Header.Get("Authorization")
        if !strings.HasPrefix(authHeader, "Bearer ") {
            http.Error(w, `{"error":{"code":"UNAUTHENTICATED","message":"Missing bearer token"}}`, http.StatusUnauthorized)
            return
        }
        tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

        token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
            // JWKS 验证（简化：直接解析 claims，真实场景需 JWKS）
            if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
                return nil, jwt.ErrSignatureInvalid
            }
            return nil, jwt.ErrSignatureInvalid // 占位，JWKS 实现
        })
        if err != nil || !token.Valid {
            http.Error(w, `{"error":{"code":"UNAUTHENTICATED","message":"Invalid token"}}", http.StatusUnauthorized)
            return
        }

        claims := token.Claims.(jwt.MapClaims)
        userID := claims["sub"].(string)
        roles := claims["realm_access"].(map[string]any)["roles"].([]any)
        role := "operator"
        for _, r := range roles {
            if r.(string) == "admin" {
                role = "admin"
                break
            }
        }

        ctx := context.WithValue(r.Context(), ContextUserID, userID)
        ctx = context.WithValue(ctx, ContextRole, role)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

- [ ] **Step 2: 添加依赖**

Run: `cd backend && go get github.com/golang-jwt/jwt/v5`

- [ ] **Step 3: 在 router 中接入中间件**

```go
// backend/internal/http/router.go 新增字段和方法
type Server struct {
    // ... 已有字段
    jwtMiddleware *auth.JWTMiddleware
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 公开路由
    if r.URL.Path == "/healthz" {
        writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
        return
    }

    // JWT 校验（除公开路由外）
    wrapped := s.jwtMiddleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        s.serveWithAuth(w, r)
    }))
    wrapped.ServeHTTP(w, r)
}

func (s *Server) serveWithAuth(w http.ResponseWriter, r *http.Request) {
    role, _ := r.Context().Value(auth.ContextRole).(string)

    // 路由权限检查
    if strings.HasPrefix(r.URL.Path, "/api/admin/") && role != "admin" {
        writeError(w, http.StatusForbidden, "FORBIDDEN", "Admin role required.")
        return
    }
    if strings.HasPrefix(r.URL.Path, "/api/operator/") && role != "operator" && role != "admin" {
        writeError(w, http.StatusForbidden, "FORBIDDEN", "Operator role required.")
        return
    }

    // 原有路由分发
    s.routeWithAuth(w, r)
}
```

- [ ] **Step 4: Build and test**

Run: `cd backend && go build ./... && go test ./internal/http -v`
Expected: Build and tests pass（AUTH_DISABLED 保持向后兼容）

- [ ] **Step 5: Commit**

```bash
git add backend/internal/auth/ backend/internal/http/ backend/go.mod backend/go.sum
git commit -m "feat(auth): add JWT middleware with AUTH_DISABLED mode and route-level role protection"

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

### Task 4.2: 内置用户登录端点

**Files:**
- Modify: `backend/internal/http/router.go`
- Modify: `backend/internal/store/store.go`

- [ ] **Step 1: 新增 login endpoint**

```go
// router.go 新增路由
case r.Method == http.MethodPost && r.URL.Path == "/api/auth/login":
    s.login(w, r)

// 实现 login
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Username string `json:"username"`
        Password string `json:"password"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request body must be valid JSON.")
        return
    }
    user, ok := s.store.VerifyPassword(req.Username, req.Password)
    if !ok {
        writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Invalid username or password.")
        return
    }
    // 返回模拟 JWT（临时方案）
    writeJSON(w, http.StatusOK, map[string]any{
        "token":      "demo-jwt-token-" + user.ID,
        "userId":     user.ID,
        "username":   user.Username,
        "role":       user.Role,
    })
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/http/router.go
git commit -m "feat(auth): add /api/auth/login endpoint for built-in user authentication"

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

---

## 阶段五：Traefik Ingress + Helm 架构重组

### Task 5.1: 创建 Umbrella Chart 结构

- [ ] **Step 1: 重组目录结构**

```bash
# 备份现有 chart
cp -r deploy/helm/k8s-ai-ops deploy/helm/k8s-ai-ops-backup

# 创建新目录结构
mkdir -p deploy/helm/k8s-ai-ops/charts/{k8s-ai-frontend,k8s-ai-backend,k8s-ai-mcp-server,postgresql,redis,keycloak,traefik}
mkdir -p deploy/helm/k8s-ai-ops/templates
```

- [ ] **Step 2: 创建 Umbrella Chart.yaml**

```yaml
# deploy/helm/k8s-ai-ops/Chart.yaml
apiVersion: v2
name: k8s-ai-ops
description: Kubernetes AI operations assistant with Keycloak, Backend API, MCP Server, and React UI.
type: application
version: 0.2.0
appVersion: "0.2.0"
dependencies:
  - name: k8s-ai-frontend
    version: "0.1.0"
  - name: k8s-ai-backend
    version: "0.1.0"
  - name: k8s-ai-mcp-server
    version: "0.1.0"
  - name: postgresql
    version: "0.1.0"
  - name: redis
    version: "0.1.0"
  - name: keycloak
    version: "0.1.0"
  - name: traefik
    version: "0.1.0"
```

- [ ] **Step 3: 将现有模板拆分到各子 chart**

每个子 chart 需要：
- `Chart.yaml` — 基础信息
- `values.yaml` — 该组件的默认 values
- `templates/` — 对应的 K8S 资源

- [ ] **Step 4: Umbrella values.yaml 覆盖**

```yaml
global:
  namespace: k8s-ai-system

images:
  source: tar
  registry: ""
  tag: local
  pullPolicy: IfNotPresent

ingress:
  enabled: false
  host: k8s-ai.local

k8s-ai-frontend:
  replicas: 1
  image:
    repository: k8s-ai-frontend
  service:
    port: 80

k8s-ai-backend:
  replicas: 1
  image:
    repository: k8s-ai-backend
  service:
    port: 8080
  encryptionKey: change-me-32-byte-development-key
  storeDriver: postgres
  cacheDriver: redis
  rbacSyncEnabled: true
  managedNamespaces:
    - dev
    - test

k8s-ai-mcp-server:
  replicas: 1
  image:
    repository: k8s-ai-mcp-server
  service:
    port: 8081
  backendGrpcAddr: k8s-ai-backend:8082
  backendHttpAddr: http://k8s-ai-backend:8080

postgresql:
  enabled: true
  image: postgres:16-alpine
  database: k8s_ai
  username: k8s_ai
  password: k8s_ai

redis:
  enabled: true
  image: redis:7-alpine

keycloak:
  enabled: true
  image: quay.io/keycloak/keycloak:26.0
  adminUser: admin
  adminPassword: admin

traefik:
  enabled: false
  image: traefik:v3.3
```

- [ ] **Step 5: 更新部署脚本**

修改 `scripts/helm-install.sh` 和 `scripts/helm-upgrade.sh` 适配新结构（`--set` 路径变化）。

- [ ] **Step 6: 创建 Ingress 模板**

```yaml
{{- if .Values.ingress.enabled }}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: k8s-ai-ingress
  namespace: {{ .Values.global.namespace }}
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: web
spec:
  ingressClassName: traefik
  rules:
    - host: {{ .Values.ingress.host }}
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: k8s-ai-frontend
                port:
                  number: 80
          - path: /api
            pathType: Prefix
            backend:
              service:
                name: k8s-ai-backend
                port:
                  number: 8080
          - path: /auth
            pathType: Prefix
            backend:
              service:
                name: keycloak
                port:
                  number: 8080
{{- end }}
```

- [ ] **Step 7: 更新 nginx.conf**

```nginx
# frontend/nginx.conf - 移除 /api/ proxy_pass
server {
    listen 80;
    root /usr/share/nginx/html;
    index index.html;
    location / {
        try_files $uri /index.html;
    }
}
```

- [ ] **Step 8: 验证 Helm 模板**

Run: `helm template deploy/helm/k8s-ai-ops --values deploy/helm/k8s-ai-ops/values-local.yaml`
Expected: YAML 输出正确，没有错误

- [ ] **Step 9: Commit**

```bash
git add deploy/helm/ frontend/nginx.conf scripts/
git commit -m "refactor(helm): restructure to umbrella chart with 7 sub-charts, add Traefik ingress support"

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

### Task 5.2: Traefik 子 chart

- [ ] **Step 1: 创建 Traefik 子 chart**（`deploy/helm/k8s-ai-ops/charts/traefik/`）

包含：
- `Chart.yaml`
- `values.yaml`（image, ports, resources）
- `templates/serviceaccount.yaml`
- `templates/clusterrole.yaml`（Traefik 需要的 RBAC）
- `templates/deployment.yaml`
- `templates/service.yaml`（type LoadBalancer/ClusterIP）
- `templates/configmap.yaml`（traefik.yml 静态配置）

---

## 阶段六：前端接入真实 API

### Task 6.1: API Client 层

**Files:**
- Create: `frontend/src/api/client.ts`
- Create: `frontend/src/api/types.ts`

- [ ] **Step 1: 创建 API client**

```typescript
// frontend/src/api/client.ts
const API_BASE = window.location.origin;

export class ApiError extends Error {
    constructor(public response: any) {
        super(response?.error?.message || 'API Error');
    }
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
    const token = localStorage.getItem('auth_token');
    const res = await fetch(`${API_BASE}${path}`, {
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
            ...options?.headers,
        },
    });
    if (!res.ok) {
        throw new ApiError(await res.json());
    }
    return res.json();
}

export const api = {
    get: <T>(path: string) => request<T>(path),
    post: <T>(path: string, body?: any) => request<T>(path, { method: 'POST', body: JSON.stringify(body) }),
    put: <T>(path: string, body?: any) => request<T>(path, { method: 'PUT', body: JSON.stringify(body) }),
};
```

- [ ] **Step 2: 创建类型定义**

```typescript
export interface User {
    id: string;
    username: string;
    displayName: string;
    email: string;
    role: 'admin' | 'operator';
    status: string;
}

export interface Permission {
    id: string;
    namespace: string;
    apiGroup: string;
    resource: string;
    verbs: string[];
    enabled: boolean;
}

export interface LLMModel {
    modelId: string;
    displayName: string;
    providerId: string;
    isDefault: boolean;
    supportsTools: boolean;
    supportsStreaming: boolean;
}

export interface LLMProvider {
    id: string;
    name: string;
    protocol: string;
    baseUrl: string;
    enabled: boolean;
    apiKeyConfigured: boolean;
}

export interface ChatMessageResponse {
    messageId: string;
    summary: string;
    resources: any[];
}

export interface ChatSession {
    id: string;
    status: string;
}

export interface AbnormalPod {
    namespace: string;
    name: string;
    phase: string;
    reason: string;
    message: string;
    restartCount: number;
    node: string;
}

export interface AuditLog {
    id: string;
    actorUserId: string;
    action: string;
    targetType: string;
    targetId: string;
    namespace: string;
    resource: string;
    verb: string;
    allowed: boolean;
    reason: string;
    createdAt: string;
}
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/api/
git commit -m "feat(frontend): add API client layer and TypeScript type definitions"

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

### Task 6.2: App.tsx 接入真实 API

- [ ] **Step 1: 修改 App.tsx 从 API 获取数据**（替换所有硬编码 mock）

### Task 6.3: 登录页面

- [ ] **Step 1: 创建 Login.tsx**
- [ ] **Step 2: 在 main.tsx 中添加路由**

---

## 阶段七：收尾优化

### Task 7.1: 缺失 API 端点

- `GET /api/admin/users/:id`
- `PUT /api/admin/users/:id/llm-models`
- `PUT /api/admin/users/:id/status`
- `GET /api/operator/chat/sessions/:id/events`

### Task 7.2: 审计日志查询参数

- 实现 AuditLogQuery 结构体和对应的 Store 查询方法

### Task 7.3: 文档同步

- 更新 `docs/architecture/system-architecture.md` — 加入 Traefik 和 gRPC
- 更新 `docs/operations/deployment-guide.md` — 新 Helm 结构和 Ingress
- 更新 `docs/architecture/chat-mcp-flow.md` — 真实调用链时序图
- 更新 `docs/architecture/data-model.md` — 新表状态
- 更新 `docs/developer/developer-guide.md` — 已完成/未完成列表
- 更新 `docs/reference/api-design.md` — 新增端点
