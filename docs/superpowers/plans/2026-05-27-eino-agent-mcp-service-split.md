# Eino Agent MCP Service Split Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建 `backend-api + agent-server + 标准 MCP server` 三服务架构，引入 Eino 承接 LLM agent 编排，支持由 backend 管理历史的多轮对话，并同步更新所有受影响文档。

**Architecture:** `backend-api` 继续作为前端唯一入口、历史会话管理者和审计中心，新增 `agent-server` 使用 Eino 执行无状态 Chat Run，`mcp-server` 提供标准 MCP 风格 Kubernetes 工具。服务之间使用稳定 JSON DTO，Eino 类型只存在于 `agent-server` 内部。

**Tech Stack:** Go 1.26、CloudWeGo Eino、Eino MCP Tool、Kubernetes client-go、PostgreSQL、Redis、Helm、Docker、Bash。

---

## 文件结构

### 新增文件

- `backend/internal/agent/client.go`：`backend-api` 调用 `agent-server` 的 HTTP client。
- `backend/internal/agent/types.go`：`backend-api` 与 `agent-server` 共享的稳定 DTO。
- `backend/internal/agent/client_test.go`：agent client 单元测试。
- `agent-server/go.mod`：新服务 Go module。
- `agent-server/go.sum`：新服务依赖锁定。
- `agent-server/Dockerfile`：新服务镜像构建。
- `agent-server/cmd/server/main.go`：新服务入口。
- `agent-server/internal/config/config.go`：读取 `HTTP_ADDR`、`MCP_SERVER_URL`。
- `agent-server/internal/http/router.go`：`/healthz` 和 `/api/agent/runs`。
- `agent-server/internal/http/router_test.go`：agent HTTP API 测试。
- `agent-server/internal/agent/types.go`：agent 服务内部 DTO，字段与 backend DTO 保持一致。
- `agent-server/internal/agent/runner.go`：runner 接口和默认 runner。
- `agent-server/internal/agent/mock_runner.go`：测试和本地 demo 使用的 mock runner。
- `agent-server/internal/eino/runner.go`：Eino runner，封装 ChatModel、Tool、MCP client。
- `agent-server/internal/eino/runner_test.go`：mock LLM + mock MCP 的 tool calling 测试。
- `mcp-server/internal/mcp/types.go`：标准 MCP 风格工具定义、调用请求、调用响应。
- `mcp-server/internal/mcp/registry.go`：工具注册表和权限校验入口。
- `mcp-server/internal/mcp/registry_test.go`：工具 schema、权限拒绝和成功路径测试。
- `deploy/helm/k8s-ai-ops/templates/agent-server.yaml`：agent 服务 Deployment 和 Service。

### 修改文件

- `backend/cmd/api/main.go`：读取 `AGENT_SERVER_URL` 并注入 agent client。
- `backend/internal/config/config.go`：新增 `AgentServerURL`。
- `backend/internal/config/config_test.go`：覆盖默认值和环境变量。
- `backend/internal/http/router.go`：Chat 消息改为调用 agent-server，写工具事件审计。
- `backend/internal/http/router_test.go`：覆盖 agent 调用、多轮上下文、错误、审计。
- `mcp-server/cmd/server/main.go`：从自定义 `/tools/list_pods` 迁移为 MCP 风格 endpoint。
- `mcp-server/internal/tools/pods.go`：补充工具请求、过滤和权限映射。
- `mcp-server/internal/tools/pods_test.go`：补充参数和权限测试。
- `scripts/build-images.sh`：新增 `agent-server` 镜像和 tar。
- `deploy/helm/k8s-ai-ops/values.yaml`：新增 `agentServer`。
- `deploy/helm/k8s-ai-ops/values-local.yaml`：同步本地配置。
- `deploy/helm/k8s-ai-ops/values-prod-example.yaml`：同步生产示例配置。
- `deploy/helm/k8s-ai-ops/templates/backend.yaml`：新增 `AGENT_SERVER_URL`。
- `README.md`：更新服务列表、命令和镜像产物。
- `ARCHITECTURE.md`：更新三服务架构摘要。
- `AI_PROMPTS.md`：记录本次架构决策。
- `docs/INDEX.md`：更新阅读路径说明。
- `docs/product/overview.md`：更新产品能力边界。
- `docs/product/requirements.md`：更新需求和验收。
- `docs/product/user-journeys.md`：更新 Chat 和管理员流程。
- `docs/architecture/system-architecture.md`：更新组件拓扑。
- `docs/architecture/permission-model.md`：更新三层授权边界。
- `docs/architecture/chat-mcp-flow.md`：更新 Eino 与 MCP 流程。
- `docs/architecture/data-model.md`：更新审计和 Chat 模型说明。
- `docs/developer/developer-guide.md`：更新服务目录、开发命令、测试命令。
- `docs/operations/deployment-guide.md`：更新 Helm values、端口、镜像和环境变量。
- `docs/operations/observability-and-troubleshooting.md`：更新日志、审计、排错。
- `docs/operations/public-cloud-test-plan.md`：更新公有云测试服务。
- `docs/security/security-design.md`：更新 LLM、agent、MCP 安全边界。
- `docs/reference/api-design.md`：更新 Chat API、错误码、内部 agent 契约。
- `docs/reference/glossary.md`：补充 Eino、Agent Server、MCP Tool、Tool Allowlist。

---

## Task 1: 定义 backend 到 agent-server 的稳定契约

**Files:**
- Create: `backend/internal/agent/types.go`
- Create: `backend/internal/agent/client.go`
- Create: `backend/internal/agent/client_test.go`

- [ ] **Step 1: 写失败测试，验证 agent client 会发送 Chat Run 请求并解析响应**

```go
package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientRunSendsRequestAndParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/agent/runs" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var request RunRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Message != "检查 dev 异常 Pod" || request.User.ID != "demo-user" {
			t.Fatalf("unexpected request body: %#v", request)
		}
		_ = json.NewEncoder(w).Encode(RunResponse{
			Summary: "dev 中有 1 个异常 Pod",
			Resources: []ResourceResult{{
				Kind:      "Pod",
				Namespace: "dev",
				Name:      "api-7b8f9",
				Status:    "ImagePullBackOff",
			}},
			ToolEvents: []ToolEvent{{
				ToolName:  "list_pods",
				Namespace: "dev",
				Resource:  "pods",
				Verb:      "list",
				Allowed:   true,
				Reason:    "tool completed",
			}},
		})
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL)
	response, err := client.Run(context.Background(), RunRequest{
		Message: "检查 dev 异常 Pod",
		User: UserContext{
			ID:       "demo-user",
			Username: "operator-a",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Summary == "" || len(response.Resources) != 1 || len(response.ToolEvents) != 1 {
		t.Fatalf("unexpected response: %#v", response)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -Command "go test ./internal/agent"` from `backend`

Expected: FAIL，提示 `backend/internal/agent` 包或 `NewHTTPClient` 未定义。

- [ ] **Step 3: 实现 DTO 和 HTTP client**

`backend/internal/agent/types.go`:

```go
package agent

type UserContext struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type ModelRuntimeConfig struct {
	ProviderID        string `json:"providerId"`
	Protocol          string `json:"protocol"`
	BaseURL           string `json:"baseUrl"`
	APIKey            string `json:"apiKey"`
	ModelName         string `json:"modelName"`
	SupportsTools     bool   `json:"supportsTools"`
	SupportsStreaming bool   `json:"supportsStreaming"`
}

type Permission struct {
	Namespace string   `json:"namespace"`
	APIGroup  string   `json:"apiGroup"`
	Resource  string   `json:"resource"`
	Verbs     []string `json:"verbs"`
}

type ToolRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	APIGroup  string `json:"apiGroup"`
	Resource  string `json:"resource"`
	Verb      string `json:"verb"`
}

type RunRequest struct {
	RequestID   string              `json:"requestId"`
	SessionID   string              `json:"sessionId"`
	MessageID   string              `json:"messageId"`
	User        UserContext         `json:"user"`
	Model       ModelRuntimeConfig  `json:"model"`
	Messages    []Message           `json:"messages"`
	Message     string              `json:"message"`
	Permissions []Permission        `json:"permissions"`
	Tools       []ToolRef           `json:"tools"`
	RuntimeContext RuntimeContext   `json:"runtimeContext"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type RuntimeContext struct {
	CurrentUser       string             `json:"currentUser"`
	AllowedNamespaces []string           `json:"allowedNamespaces"`
	RecentResources   []ResourceRef      `json:"recentResources"`
}

type ResourceRef struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type ResourceResult struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Status    string `json:"status"`
}

type ToolEvent struct {
	ToolName  string `json:"toolName"`
	Namespace string `json:"namespace"`
	Resource  string `json:"resource"`
	Verb      string `json:"verb"`
	Allowed   bool   `json:"allowed"`
	Reason    string `json:"reason"`
}

type RunResponse struct {
	Summary    string           `json:"summary"`
	Resources  []ResourceResult `json:"resources"`
	ToolEvents []ToolEvent      `json:"toolEvents"`
}
```

`backend/internal/agent/client.go`:

```go
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client interface {
	Run(context.Context, RunRequest) (RunResponse, error)
}

type HTTPClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *HTTPClient) Run(ctx context.Context, request RunRequest) (RunResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return RunResponse{}, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/agent/runs", bytes.NewReader(body))
	if err != nil {
		return RunResponse{}, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := c.client.Do(httpRequest)
	if err != nil {
		return RunResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return RunResponse{}, fmt.Errorf("agent server returned status %d", response.StatusCode)
	}
	var runResponse RunResponse
	if err := json.NewDecoder(response.Body).Decode(&runResponse); err != nil {
		return RunResponse{}, err
	}
	return runResponse, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -Command "go test ./internal/agent"` from `backend`

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add backend/internal/agent
git commit -m "feat: add backend agent client contract"
```

---

## Task 2: 新增 agent-server HTTP 骨架和 mock runner

**Files:**
- Create: `agent-server/go.mod`
- Create: `agent-server/Dockerfile`
- Create: `agent-server/cmd/server/main.go`
- Create: `agent-server/internal/config/config.go`
- Create: `agent-server/internal/agent/types.go`
- Create: `agent-server/internal/agent/runner.go`
- Create: `agent-server/internal/agent/mock_runner.go`
- Create: `agent-server/internal/http/router.go`
- Create: `agent-server/internal/http/router_test.go`

- [ ] **Step 1: 写失败测试，验证 `/healthz` 和 `/api/agent/runs`**

```go
package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"k8s-ai-ops/agent-server/internal/agent"
)

func TestRouterHealthz(t *testing.T) {
	recorder := httptest.NewRecorder()
	NewRouter(agent.NewMockRunner()).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestRouterRunsAgent(t *testing.T) {
	payload := []byte(`{"message":"检查 dev 异常 Pod","user":{"id":"demo-user","username":"operator-a"}}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/agent/runs", bytes.NewReader(payload))

	NewRouter(agent.NewMockRunner()).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var body agent.RunResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Summary == "" || len(body.ToolEvents) == 0 {
		t.Fatalf("unexpected response: %#v", body)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -Command "go test ./..."` from `agent-server`

Expected: FAIL，提示 module 或 router 未定义。

- [ ] **Step 3: 创建 agent-server module 与骨架实现**

`agent-server/go.mod`:

```go
module k8s-ai-ops/agent-server

go 1.26
```

`agent-server/internal/agent/runner.go`:

```go
package agent

import "context"

type Runner interface {
	Run(context.Context, RunRequest) (RunResponse, error)
}
```

`agent-server/internal/agent/types.go` 使用 Task 1 中同名 DTO 字段。

`agent-server/internal/agent/mock_runner.go`:

```go
package agent

import "context"

type MockRunner struct{}

func NewMockRunner() *MockRunner {
	return &MockRunner{}
}

func (r *MockRunner) Run(_ context.Context, request RunRequest) (RunResponse, error) {
	return RunResponse{
		Summary: "dev namespace 中有 1 个异常 Pod。",
		Resources: []ResourceResult{{
			Kind:      "Pod",
			Namespace: "dev",
			Name:      "api-7b8f9",
			Status:    "ImagePullBackOff",
		}},
		ToolEvents: []ToolEvent{{
			ToolName:  "list_pods",
			Namespace: "dev",
			Resource:  "pods",
			Verb:      "list",
			Allowed:   true,
			Reason:    "mock tool completed",
		}},
	}, nil
}
```

`agent-server/internal/http/router.go`:

```go
package http

import (
	"encoding/json"
	"net/http"

	"k8s-ai-ops/agent-server/internal/agent"
)

type Server struct {
	runner agent.Runner
}

func NewRouter(runner agent.Runner) http.Handler {
	return &Server{runner: runner}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/healthz":
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	case r.Method == http.MethodPost && r.URL.Path == "/api/agent/runs":
		s.runAgent(w, r)
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{"error": map[string]string{"code": "NOT_FOUND", "message": "The requested API endpoint was not found."}})
	}
}

func (s *Server) runAgent(w http.ResponseWriter, r *http.Request) {
	var request agent.RunRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]string{"code": "INVALID_REQUEST", "message": "Request body must be valid JSON."}})
		return
	}
	response, err := s.runner.Run(r.Context(), request)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": map[string]string{"code": "AGENT_RUN_FAILED", "message": "Agent run failed."}})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
```

`agent-server/cmd/server/main.go`:

```go
package main

import (
	"log"
	"net/http"
	"os"

	"k8s-ai-ops/agent-server/internal/agent"
	apihttp "k8s-ai-ops/agent-server/internal/http"
)

func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8082"
	}
	log.Printf("level=INFO component=agent-server event=server_start addr=%s", addr)
	if err := http.ListenAndServe(addr, apihttp.NewRouter(agent.NewMockRunner())); err != nil {
		log.Fatalf("level=ERROR component=agent-server event=server_exit error=%q", err)
	}
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -Command "go test ./..."` from `agent-server`

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add agent-server
git commit -m "feat: add agent server skeleton"
```

---

## Task 3: backend-api Chat 接入 agent-server 和多轮上下文

**Files:**
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/config/config_test.go`
- Modify: `backend/cmd/api/main.go`
- Modify: `backend/internal/http/router.go`
- Modify: `backend/internal/http/router_test.go`

- [ ] **Step 1: 写失败测试，验证 Chat 会调用 agent client 并写审计**

在 `backend/internal/http/router_test.go` 新增：

```go
type recordingAgentClient struct {
	calls []agent.RunRequest
	err   error
}

func (c *recordingAgentClient) Run(_ context.Context, request agent.RunRequest) (agent.RunResponse, error) {
	c.calls = append(c.calls, request)
	if c.err != nil {
		return agent.RunResponse{}, c.err
	}
	return agent.RunResponse{
		Summary: "agent 返回巡检结果",
		Resources: []agent.ResourceResult{{
			Kind:      "Pod",
			Namespace: "dev",
			Name:      "api-7b8f9",
			Status:    "ImagePullBackOff",
		}},
		ToolEvents: []agent.ToolEvent{{
			ToolName:  "list_pods",
			Namespace: "dev",
			Resource:  "pods",
			Verb:      "list",
			Allowed:   true,
			Reason:    "tool completed",
		}},
	}, nil
}

func TestRouterCallsAgentForChatMessage(t *testing.T) {
	server := NewServer()
	client := &recordingAgentClient{}
	server.SetAgentClient(client)

	payload := []byte(`{"modelId":"mock-local","content":"帮我看看 dev 异常 Pod"}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/operator/chat/sessions/demo-session/messages", bytes.NewReader(payload))

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if len(client.calls) != 1 {
		t.Fatalf("expected one agent call, got %#v", client.calls)
	}
	if client.calls[0].Message != "帮我看看 dev 异常 Pod" {
		t.Fatalf("unexpected agent request: %#v", client.calls[0])
	}
	if len(client.calls[0].Messages) == 0 {
		t.Fatalf("expected messages context, got %#v", client.calls[0])
	}
	if client.calls[0].RuntimeContext.CurrentUser == "" {
		t.Fatalf("expected runtime context, got %#v", client.calls[0])
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -Command "go test ./internal/http"` from `backend`

Expected: FAIL，提示 `SetAgentClient` 或 `agent` import 未定义。

- [ ] **Step 3: 实现配置和依赖注入**

`backend/internal/config/config.go` 增加字段和默认值：

```go
AgentServerURL string
```

`Load()` 增加：

```go
AgentServerURL: env("AGENT_SERVER_URL", "http://agent-server:8082"),
```

`backend/internal/http/router.go` 增加字段和 setter：

```go
agentClient agent.Client

func (s *Server) SetAgentClient(client agent.Client) {
	s.agentClient = client
}
```

`NewServer` 默认设置 mock agent client 或 nil fallback。`backend/cmd/api/main.go` 在创建 server 后注入：

```go
server.SetAgentClient(agent.NewHTTPClient(cfg.AgentServerURL))
```

- [ ] **Step 4: 将 `createChatMessage` 改为调用 agent-server**

保留模型授权检查，授权通过后构造 `agent.RunRequest`。`backend-api` 负责组装多轮上下文：`Messages` 至少包含当前用户消息，后续接入持久化 Chat 消息后按 session 读取最近历史；`RuntimeContext` 包含当前用户、授权 namespace 和最近资源引用。将 `agent.RunResponse` 转换为当前前端响应：

```go
runResponse, err := s.agentClient.Run(r.Context(), agent.RunRequest{
	RequestID: "req-demo",
	SessionID: "demo-session",
	MessageID: "demo-message",
	User: agent.UserContext{
		ID:       s.store.CurrentDemoUser().ID,
		Username: s.store.CurrentDemoUser().Username,
	},
	Model: agent.ModelRuntimeConfig{
		ProviderID:        "mock",
		Protocol:          string(llm.ProtocolMock),
		ModelName:         request.ModelID,
		SupportsTools:     true,
		SupportsStreaming: true,
	},
	Messages: []agent.Message{{
		Role:    "user",
		Content: request.Content,
	}},
	Message:     request.Content,
	Permissions: s.agentPermissionsForCurrentUser(),
	Tools:       s.agentToolsForCurrentUser(),
	RuntimeContext: s.agentRuntimeContextForCurrentUser(),
})
if err != nil {
	s.audit("operator.chat.message.create", "chat_message", "demo-message", "", "", "", false, "agent server unavailable")
	writeError(w, http.StatusBadGateway, "AGENT_SERVER_UNAVAILABLE", "Agent server is unavailable.")
	return
}
for _, event := range runResponse.ToolEvents {
	s.audit("operator.tool.call", "mcp_tool", event.ToolName, event.Namespace, event.Resource, event.Verb, event.Allowed, event.Reason)
}
writeJSON(w, http.StatusOK, map[string]any{
	"messageId": "demo-message",
	"summary":   runResponse.Summary,
	"resources": runResponse.Resources,
})
```

- [ ] **Step 5: 运行 backend 测试确认通过**

Run: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -Command "go test ./..."` from `backend`

Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add backend
git commit -m "feat: route backend chat through agent server"
```

---

## Task 4: MCP server 增加标准 MCP 风格工具注册表

**Files:**
- Create: `mcp-server/internal/mcp/types.go`
- Create: `mcp-server/internal/mcp/registry.go`
- Create: `mcp-server/internal/mcp/registry_test.go`
- Modify: `mcp-server/cmd/server/main.go`

- [ ] **Step 1: 写失败测试，验证工具列表和权限拒绝**

```go
package mcp

import "testing"

func TestRegistryListsPodToolSchema(t *testing.T) {
	registry := NewRegistry()
	tools := registry.ListTools()
	if len(tools) == 0 {
		t.Fatal("expected tools")
	}
	if tools[0].Name == "" || tools[0].Resource == "" || tools[0].Verb == "" {
		t.Fatalf("unexpected tool schema: %#v", tools[0])
	}
}

func TestRegistryDeniesUnauthorizedToolCall(t *testing.T) {
	registry := NewRegistry()
	_, err := registry.CallTool(CallRequest{
		Name:      "list_pods",
		Namespace: "prod",
		Resource:  "pods",
		Verb:      "list",
		Permissions: []Permission{{
			Namespace: "dev",
			Resource:  "pods",
			Verbs:     []string{"list"},
		}},
	})
	if err == nil {
		t.Fatal("expected unauthorized call to fail")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -Command "go test ./internal/mcp"` from `mcp-server`

Expected: FAIL，提示 package 或类型未定义。

- [ ] **Step 3: 实现 registry 和权限校验**

`mcp-server/internal/mcp/types.go`:

```go
package mcp

type ToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Namespace   string `json:"namespace"`
	APIGroup    string `json:"apiGroup"`
	Resource    string `json:"resource"`
	Verb        string `json:"verb"`
}

type Permission struct {
	Namespace string   `json:"namespace"`
	APIGroup  string   `json:"apiGroup"`
	Resource  string   `json:"resource"`
	Verbs     []string `json:"verbs"`
}

type CallRequest struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	APIGroup    string            `json:"apiGroup"`
	Resource    string            `json:"resource"`
	Verb        string            `json:"verb"`
	Arguments   map[string]string `json:"arguments"`
	Permissions []Permission      `json:"permissions"`
}

type CallResponse struct {
	Content map[string]any `json:"content"`
}
```

`mcp-server/internal/mcp/registry.go`:

```go
package mcp

import "errors"

type Registry struct {
	tools []ToolDefinition
}

func NewRegistry() *Registry {
	return &Registry{tools: []ToolDefinition{
		{Name: "list_pods", Description: "List Kubernetes pods", Resource: "pods", Verb: "list"},
		{Name: "get_pod", Description: "Get Kubernetes pod detail", Resource: "pods", Verb: "get"},
		{Name: "get_pod_logs", Description: "Get Kubernetes pod logs", Resource: "pods/log", Verb: "get"},
		{Name: "list_events", Description: "List Kubernetes events", Resource: "events", Verb: "list"},
		{Name: "list_deployments", Description: "List Kubernetes deployments", APIGroup: "apps", Resource: "deployments", Verb: "list"},
		{Name: "restart_deployment", Description: "Restart Kubernetes deployment", APIGroup: "apps", Resource: "deployments", Verb: "patch"},
	}}
}

func (r *Registry) ListTools() []ToolDefinition {
	return append([]ToolDefinition(nil), r.tools...)
}

func (r *Registry) CallTool(request CallRequest) (CallResponse, error) {
	if !isAllowed(request) {
		return CallResponse{}, errors.New("tool call denied by user Kubernetes permissions")
	}
	return CallResponse{Content: map[string]any{"status": "ok", "tool": request.Name}}, nil
}

func isAllowed(request CallRequest) bool {
	for _, permission := range request.Permissions {
		if permission.Namespace != request.Namespace || permission.APIGroup != request.APIGroup || permission.Resource != request.Resource {
			continue
		}
		for _, verb := range permission.Verbs {
			if verb == request.Verb {
				return true
			}
		}
	}
	return false
}
```

- [ ] **Step 4: 更新 MCP HTTP endpoint**

`mcp-server/cmd/server/main.go` 保留 `/healthz`，新增：

- `GET /mcp/tools`
- `POST /mcp/tools/call`

使用 `mcp.NewRegistry()` 处理请求，错误返回：

```json
{"error":{"code":"MCP_TOOL_DENIED","message":"Tool call denied by user Kubernetes permissions."}}
```

- [ ] **Step 5: 运行 MCP 测试确认通过**

Run: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -Command "go test ./..."` from `mcp-server`

Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add mcp-server
git commit -m "feat: add mcp tool registry"
```

---

## Task 5: agent-server 引入 Eino runner 边界和上下文消费

**Files:**
- Modify: `agent-server/go.mod`
- Create: `agent-server/internal/eino/runner.go`
- Create: `agent-server/internal/eino/runner_test.go`
- Modify: `agent-server/cmd/server/main.go`

- [ ] **Step 1: 写失败测试，验证 Eino runner 的 mock tool calling 流程**

```go
package eino

import (
	"context"
	"testing"

	"k8s-ai-ops/agent-server/internal/agent"
)

func TestRunnerReturnsToolEventsFromAllowedTools(t *testing.T) {
	runner := NewMockableRunner(MockLLM{
		FinalSummary: "dev namespace 中有 1 个异常 Pod。",
	}, MockMCP{
		ToolName: "list_pods",
	})

	response, err := runner.Run(context.Background(), agent.RunRequest{
		Message: "检查 dev 异常 Pod",
		Messages: []agent.Message{
			{Role: "user", Content: "帮我看看 dev 有什么异常"},
			{Role: "assistant", Content: "dev 中有 1 个异常 Pod api-7b8f9"},
			{Role: "user", Content: "看看这个 Pod 的日志"},
		},
		RuntimeContext: agent.RuntimeContext{
			CurrentUser:       "operator-a",
			AllowedNamespaces: []string{"dev"},
			RecentResources: []agent.ResourceRef{{
				Kind:      "Pod",
				Namespace: "dev",
				Name:      "api-7b8f9",
			}},
		},
		Tools: []agent.ToolRef{{
			Name:      "list_pods",
			Namespace: "dev",
			Resource:  "pods",
			Verb:      "list",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Summary == "" || len(response.ToolEvents) != 1 {
		t.Fatalf("unexpected response: %#v", response)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -Command "go test ./internal/eino"` from `agent-server`

Expected: FAIL，提示 package 或 `NewMockableRunner` 未定义。

- [ ] **Step 3: 添加 Eino 依赖**

Run: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -Command "go get github.com/cloudwego/eino@latest github.com/cloudwego/eino-ext/components/model/openai@latest github.com/cloudwego/eino-ext/components/tool/mcp@latest"` from `agent-server`

Expected: `go.mod` 和 `go.sum` 更新。

说明：Eino 官方仓库提供 `ChatModel`、`Tool`、ADK 和 graph/workflow 能力，EinoExt 提供 OpenAI 等模型与 MCP tool 扩展。实现时优先使用 `ToolCallingChatModel` 和 MCP tool 扩展；如果当前 MCP 传输细节不匹配，保留 `MCPClient` 接口适配层，避免阻塞服务拆分。

- [ ] **Step 4: 实现 mockable Eino runner 适配层**

`agent-server/internal/eino/runner.go`:

```go
package eino

import (
	"context"

	"k8s-ai-ops/agent-server/internal/agent"
)

type LLM interface {
	Generate(context.Context, agent.RunRequest, []agent.ToolRef) (string, error)
}

type MCPClient interface {
	Call(context.Context, agent.ToolRef, agent.RunRequest) (agent.ToolEvent, []agent.ResourceResult, error)
}

type Runner struct {
	llm LLM
	mcp MCPClient
}

func NewMockableRunner(llm LLM, mcp MCPClient) *Runner {
	return &Runner{llm: llm, mcp: mcp}
}

func (r *Runner) Run(ctx context.Context, request agent.RunRequest) (agent.RunResponse, error) {
	events := make([]agent.ToolEvent, 0, len(request.Tools))
	resources := []agent.ResourceResult{}
	for _, tool := range request.Tools {
		event, toolResources, err := r.mcp.Call(ctx, tool, request)
		if err != nil {
			return agent.RunResponse{}, err
		}
		events = append(events, event)
		resources = append(resources, toolResources...)
	}
	summary, err := r.llm.Generate(ctx, request, request.Tools)
	if err != nil {
		return agent.RunResponse{}, err
	}
	return agent.RunResponse{Summary: summary, Resources: resources, ToolEvents: events}, nil
}

type MockLLM struct {
	FinalSummary string
}

func (m MockLLM) Generate(context.Context, agent.RunRequest, []agent.ToolRef) (string, error) {
	return m.FinalSummary, nil
}

type MockMCP struct {
	ToolName string
}

func (m MockMCP) Call(_ context.Context, tool agent.ToolRef, _ agent.RunRequest) (agent.ToolEvent, []agent.ResourceResult, error) {
	return agent.ToolEvent{
		ToolName:  tool.Name,
		Namespace: tool.Namespace,
		Resource:  tool.Resource,
		Verb:      tool.Verb,
		Allowed:   true,
		Reason:    "tool completed",
	}, []agent.ResourceResult{{
		Kind:      "Pod",
		Namespace: tool.Namespace,
		Name:      "api-7b8f9",
		Status:    "ImagePullBackOff",
	}}, nil
}
```

- [ ] **Step 5: 将 Eino runner 接入 main**

`main.go` 使用 `eino.NewMockableRunner` 作为当前运行时 runner，LLM 和 MCP 通过接口适配器注入。当前阶段的适配器返回可预测结果，保证服务拆分、Eino 边界和工具事件链路先落地；真实 provider 配置在同一个 `LLM` 接口内替换，不改变 HTTP API。

- [ ] **Step 6: 运行 agent-server 测试确认通过**

Run: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -Command "go test ./..."` from `agent-server`

Expected: PASS。

- [ ] **Step 7: 提交**

```bash
git add agent-server
git commit -m "feat: add eino runner boundary"
```

---

## Task 6: Helm 和构建脚本加入 agent-server

**Files:**
- Create: `deploy/helm/k8s-ai-ops/templates/agent-server.yaml`
- Modify: `deploy/helm/k8s-ai-ops/values.yaml`
- Modify: `deploy/helm/k8s-ai-ops/values-local.yaml`
- Modify: `deploy/helm/k8s-ai-ops/values-prod-example.yaml`
- Modify: `deploy/helm/k8s-ai-ops/templates/backend.yaml`
- Modify: `scripts/build-images.sh`

- [ ] **Step 1: 写失败验证，确认当前 Helm 不包含 agent-server**

Run: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -Command "helm template k8s-ai deploy/helm/k8s-ai-ops | Select-String 'agent-server'"` from repo root

Expected: 没有输出或命令失败，说明模板尚未包含新服务。

- [ ] **Step 2: 新增 Helm values**

在三个 values 文件增加：

```yaml
agentServer:
  replicas: 1
  image:
    repository: k8s-ai-agent-server
  service:
    port: 8082
```

- [ ] **Step 3: 新增 agent-server 模板**

`deploy/helm/k8s-ai-ops/templates/agent-server.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agent-server
  namespace: {{ .Values.global.namespace }}
spec:
  replicas: {{ .Values.agentServer.replicas }}
  selector:
    matchLabels:
      app: agent-server
  template:
    metadata:
      labels:
        app: agent-server
    spec:
      containers:
        - name: agent-server
          image: {{ include "k8s-ai-ops.image" (list . .Values.agentServer.image.repository) }}
          imagePullPolicy: {{ .Values.images.pullPolicy }}
          ports:
            - containerPort: 8082
          env:
            - name: HTTP_ADDR
              value: ":8082"
            - name: MCP_SERVER_URL
              value: "http://mcp-server:8081"
---
apiVersion: v1
kind: Service
metadata:
  name: agent-server
  namespace: {{ .Values.global.namespace }}
spec:
  selector:
    app: agent-server
  ports:
    - port: {{ .Values.agentServer.service.port }}
      targetPort: 8082
```

- [ ] **Step 4: backend 注入 `AGENT_SERVER_URL`**

`deploy/helm/k8s-ai-ops/templates/backend.yaml` env 增加：

```yaml
- name: AGENT_SERVER_URL
  value: "http://agent-server:8082"
```

- [ ] **Step 5: 构建脚本新增镜像和 tar**

`scripts/build-images.sh` 增加：

```bash
docker build -t "k8s-ai-agent-server:$TAG" agent-server
docker save "k8s-ai-agent-server:$TAG" -o "$OUTPUT_DIR/agent-server-amd64.tar"
```

- [ ] **Step 6: 运行 Helm 验证**

Run: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -Command "helm template k8s-ai deploy/helm/k8s-ai-ops | Select-String 'agent-server'"` from repo root

Expected: 输出包含 `name: agent-server`、`k8s-ai-agent-server`、`AGENT_SERVER_URL`。

- [ ] **Step 7: 提交**

```bash
git add deploy/helm/k8s-ai-ops scripts/build-images.sh
git commit -m "feat: deploy agent server"
```

---

## Task 7: 同步更新根目录和 docs 全部受影响文档

**Files:**
- Modify all documentation files listed in the design spec.

- [ ] **Step 1: 更新根目录文档**

更新 `README.md`、`ARCHITECTURE.md`、`AI_PROMPTS.md`，必须包含：

- 三服务列表：`backend-api`、`agent-server`、`mcp-server`。
- 端口：`8080`、`8082`、`8081`。
- 镜像产物：`backend-api-amd64.tar`、`agent-server-amd64.tar`、`mcp-server-amd64.tar`、`frontend-amd64.tar`。
- 验证命令：`go test ./...` for `backend`、`agent-server`、`mcp-server`，以及 `helm template`。

- [ ] **Step 2: 更新产品文档**

更新：

- `docs/product/overview.md`
- `docs/product/requirements.md`
- `docs/product/user-journeys.md`

必须说明：

- Eino agent 负责 LLM 编排。
- `backend-api` 管理多轮会话历史，`agent-server` 只消费每次请求携带的 `messages` 和 `runtimeContext`。
- 标准 MCP 工具负责 Kubernetes 操作。
- 管理员配置 LLM Provider 和模型。
- 操作员 Chat 巡检通过权限 allowlist 限制工具范围。

- [ ] **Step 3: 更新架构文档**

更新：

- `docs/architecture/system-architecture.md`
- `docs/architecture/permission-model.md`
- `docs/architecture/chat-mcp-flow.md`
- `docs/architecture/data-model.md`

必须包含新的 mermaid 三服务架构图和 Chat 时序图，并说明审计仍由 `backend-api` 统一落库。
同时说明多轮对话历史、最近资源引用和权限摘要由 `backend-api` 组装，`agent-server` 不持久化历史。

- [ ] **Step 4: 更新开发、运维、安全、参考文档**

更新：

- `docs/developer/developer-guide.md`
- `docs/operations/deployment-guide.md`
- `docs/operations/observability-and-troubleshooting.md`
- `docs/operations/public-cloud-test-plan.md`
- `docs/security/security-design.md`
- `docs/reference/api-design.md`
- `docs/reference/glossary.md`
- `docs/INDEX.md`

必须包含新服务目录、环境变量、Helm values、错误码、日志组件名和术语。
`docs/reference/api-design.md` 必须包含 `AgentRunRequest.messages`、`AgentRunRequest.runtimeContext` 和当前用户消息 `message` 的关系。

- [ ] **Step 5: 文档一致性扫描**

Run: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -Command "Select-String -Path README.md,ARCHITECTURE.md,AI_PROMPTS.md,docs\\**\\*.md -Pattern 'agent-server|AGENT_SERVER_URL|8082|k8s-ai-agent-server'"`

Expected: 多个文档均出现新服务说明。

- [ ] **Step 6: 提交**

```bash
git add README.md ARCHITECTURE.md AI_PROMPTS.md docs
git commit -m "docs: update agent server architecture"
```

---

## Task 8: 全量验证和收尾

**Files:**
- No direct edits unless verification reveals issues.

- [ ] **Step 1: 运行 backend 测试**

Run: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -Command "go test ./..."` from `backend`

Expected: PASS。

- [ ] **Step 2: 运行 agent-server 测试**

Run: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -Command "go test ./..."` from `agent-server`

Expected: PASS。

- [ ] **Step 3: 运行 mcp-server 测试**

Run: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -Command "go test ./..."` from `mcp-server`

Expected: PASS。

- [ ] **Step 4: 运行 Helm 模板验证**

Run: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -Command "helm template k8s-ai deploy/helm/k8s-ai-ops"` from repo root

Expected: exit 0。

- [ ] **Step 5: 检查敏感信息**

Run: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -Command "Select-String -Path backend\\**\\*.go,agent-server\\**\\*.go,mcp-server\\**\\*.go,docs\\**\\*.md -Pattern 'runtime-secret|secret-key|apiKey'"`

Expected: 只在示例或测试中出现，生产日志不输出 API Key。

- [ ] **Step 6: 查看 git 状态**

Run: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -Command "git status --short"` from repo root

Expected: 只剩用户已有的无关改动，或工作区干净。
