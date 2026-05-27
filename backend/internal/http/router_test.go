package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	agentclient "k8s-ai-ops/backend/internal/agent"
	k8sops "k8s-ai-ops/backend/internal/k8s"
	agentv1 "k8s-ai-ops/proto/agent/v1"
)

func TestRouterReturnsCurrentUser(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)

	NewRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["role"] != "operator" {
		t.Fatalf("expected operator role, got %#v", body["role"])
	}
}

func TestRouterReturnsOperatorPermissions(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/operator/permissions", nil)

	NewRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	var body []map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) == 0 || body[0]["namespace"] != "dev" {
		t.Fatalf("expected dev permission, got %#v", body)
	}
}

func TestRouterCreatesChatSession(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/operator/chat/sessions", nil)

	NewRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["id"] == "" {
		t.Fatalf("expected session id, got %#v", body)
	}
}

func TestRouterRejectsDisallowedModelForChatMessage(t *testing.T) {
	payload := []byte(`{"modelId":"not-allowed","content":"帮我看看现在集群里有什么异常吗？"}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/operator/chat/sessions/demo-session/messages", bytes.NewReader(payload))

	NewRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	errorBody := body["error"].(map[string]any)
	if errorBody["code"] != "LLM_MODEL_NOT_ALLOWED" {
		t.Fatalf("expected LLM_MODEL_NOT_ALLOWED, got %#v", errorBody)
	}
}

func TestRouterReturnsInspectionResultForChatMessage(t *testing.T) {
	server := NewServer()
	client := &recordingAgentClient{}
	server.SetAgentClient(client)
	payload := []byte(`{"modelId":"mock-local","content":"帮我看看现在集群里有什么异常吗？"}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/operator/chat/sessions/demo-session/messages", bytes.NewReader(payload))

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	// Parse SSE stream
	contentType := recorder.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/event-stream") {
		t.Fatalf("expected text/event-stream, got %s", contentType)
	}

	var eventCount int
	var lastSummary string
	body := recorder.Body.String()
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			eventCount++
			data := strings.TrimPrefix(line, "data: ")
			var wrapper map[string]any
			if err := json.Unmarshal([]byte(data), &wrapper); err != nil {
				continue
			}
			if eventField, ok := wrapper["Event"].(map[string]any); ok {
				if complete, ok := eventField["Complete"].(map[string]any); ok {
					lastSummary = complete["summary"].(string)
				}
			}
		}
	}
	if eventCount == 0 {
		t.Fatalf("expected at least one SSE event, got body=%s", body)
	}
	if lastSummary == "" {
		t.Fatalf("expected complete event with summary, got body=%s", body)
	}

	if len(client.requests) != 1 {
		t.Fatalf("expected one agent request, got %#v", client.requests)
	}
	if client.requests[0].GetMessage() != "帮我看看现在集群里有什么异常吗？" {
		t.Fatalf("unexpected agent request: %#v", client.requests[0])
	}
	if len(client.requests[0].GetMessages()) == 0 || client.requests[0].GetRuntimeContext().GetCurrentUser() == "" {
		t.Fatalf("expected messages and runtime context, got %#v", client.requests[0])
	}
}

func TestRouterReturnsStructuredNotFoundError(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/missing", nil)

	NewRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	errorBody := body["error"].(map[string]any)
	if errorBody["code"] != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND, got %#v", errorBody)
	}
}

func TestRouterCreatesAdminUser(t *testing.T) {
	payload := []byte(`{"username":"operator-b","email":"operator-b@example.com","role":"operator","displayName":"Operator B"}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/admin/users", bytes.NewReader(payload))

	NewRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["username"] != "operator-b" || body["role"] != "operator" {
		t.Fatalf("unexpected created user: %#v", body)
	}
}

func TestRouterRejectsInvalidUserRole(t *testing.T) {
	payload := []byte(`{"username":"bad","email":"bad@example.com","role":"owner"}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/admin/users", bytes.NewReader(payload))

	NewRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	errorBody := body["error"].(map[string]any)
	if errorBody["code"] != "INVALID_USER_ROLE" {
		t.Fatalf("expected INVALID_USER_ROLE, got %#v", errorBody)
	}
}

func TestRouterUpdatesUserPermissions(t *testing.T) {
	server := NewServer()
	payload := []byte(`{"permissions":[{"namespace":"test","apiGroup":"","resource":"pods","verbs":["get","list"]}]}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/admin/users/demo-user/permissions", bytes.NewReader(payload))

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	listRecorder := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/operator/permissions", nil)
	server.ServeHTTP(listRecorder, listRequest)

	var body []map[string]any
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 1 || body[0]["namespace"] != "test" {
		t.Fatalf("expected updated test permission, got %#v", body)
	}
}

func TestRouterAppliesRBACWhenUpdatingUserPermissions(t *testing.T) {
	server := NewServer()
	applier := &recordingRBACApplier{}
	server.SetRBACApplier(applier)
	payload := []byte(`{"permissions":[{"namespace":"dev","apiGroup":"","resource":"pods","verbs":["get","list"]},{"namespace":"test","apiGroup":"apps","resource":"deployments","verbs":["get"]}]}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/admin/users/demo-user/permissions", bytes.NewReader(payload))

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if len(applier.calls) != 2 {
		t.Fatalf("expected rbac apply per namespace, got %#v", applier.calls)
	}
	seen := map[string]bool{}
	for _, call := range applier.calls {
		seen[call.Namespace] = true
		if call.UserID != "demo-user" {
			t.Fatalf("expected demo-user, got %#v", call)
		}
		if len(call.Rules) != 1 {
			t.Fatalf("expected one rule per namespace, got %#v", call)
		}
	}
	if !seen["dev"] || !seen["test"] {
		t.Fatalf("expected dev and test namespace calls, got %#v", applier.calls)
	}
}

func TestRouterReturnsErrorWhenRBACApplyFails(t *testing.T) {
	server := NewServer()
	server.SetRBACApplier(&recordingRBACApplier{err: errors.New("k8s unavailable")})
	payload := []byte(`{"permissions":[{"namespace":"dev","apiGroup":"","resource":"pods","verbs":["get"]}]}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/admin/users/demo-user/permissions", bytes.NewReader(payload))

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	errorBody := body["error"].(map[string]any)
	if errorBody["code"] != "K8S_RBAC_APPLY_FAILED" {
		t.Fatalf("expected K8S_RBAC_APPLY_FAILED, got %#v", errorBody)
	}
}

func TestRouterCreatesLLMProviderWithoutReturningAPIKey(t *testing.T) {
	server := NewServer()
	payload := []byte(`{"name":"OpenAI","protocol":"openai","baseUrl":"https://api.openai.com/v1","apiKey":"secret-key","enabled":true}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/admin/llm/providers", bytes.NewReader(payload))

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if bytes.Contains(recorder.Body.Bytes(), []byte("secret-key")) {
		t.Fatalf("response leaked api key: %s", recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["protocol"] != "openai" || body["apiKeyConfigured"] != true {
		t.Fatalf("unexpected provider response: %#v", body)
	}
}

type recordingRBACApplier struct {
	calls []k8sops.UserNamespacePermissions
	err   error
}

type recordingAgentClient struct {
	requests []*agentv1.AgentRunRequest
	err      error
}

type recordingStreamReader struct {
	events []*agentv1.StreamEvent
	pos    int
}

func (r *recordingStreamReader) Recv() (*agentv1.StreamEvent, error) {
	if r.pos >= len(r.events) {
		return nil, io.EOF
	}
	event := r.events[r.pos]
	r.pos++
	return event, nil
}

func (c *recordingAgentClient) RunStream(_ context.Context, request *agentv1.AgentRunRequest) (agentclient.StreamReader, error) {
	c.requests = append(c.requests, request)
	if c.err != nil {
		return nil, c.err
	}
	return &recordingStreamReader{events: []*agentv1.StreamEvent{
		{EventId: "1", Event: &agentv1.StreamEvent_Thinking{Thinking: &agentv1.ThinkingEvent{Content: "analyzing..."}}},
		{EventId: "2", Event: &agentv1.StreamEvent_Complete{Complete: &agentv1.CompleteEvent{
			Summary: "agent 返回巡检结果",
			Resources: []*agentv1.ResourceResult{{
				Kind:      "Pod",
				Namespace: "dev",
				Name:      "api-7b8f9",
				Status:    "ImagePullBackOff",
			}},
		}}},
	}}, nil
}

func (a *recordingRBACApplier) ApplyUserNamespacePermissions(_ context.Context, permissions k8sops.UserNamespacePermissions) error {
	a.calls = append(a.calls, permissions)
	return a.err
}

func TestRouterCreatesLLMModel(t *testing.T) {
	server := NewServer()
	providerPayload := []byte(`{"name":"OpenAI","protocol":"openai","baseUrl":"https://api.openai.com/v1","apiKey":"secret-key","enabled":true}`)
	server.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/admin/llm/providers", bytes.NewReader(providerPayload)))

	modelPayload := []byte(`{"providerId":"provider-openai","modelName":"gpt-4.1","displayName":"GPT 4.1","supportsTools":true,"supportsStreaming":true,"enabled":true}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/admin/llm/models", bytes.NewReader(modelPayload))

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["modelName"] != "gpt-4.1" || body["supportsTools"] != true {
		t.Fatalf("unexpected model response: %#v", body)
	}
}

func TestRouterListsAuditLogsAfterAdminAction(t *testing.T) {
	server := NewServer()
	payload := []byte(`{"username":"operator-c","email":"operator-c@example.com","role":"operator","displayName":"Operator C"}`)
	server.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/admin/users", bytes.NewReader(payload)))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/admin/audit-logs", nil)
	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var body []map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) == 0 {
		t.Fatal("expected at least one audit log")
	}
	if body[0]["action"] != "admin.user.create" {
		t.Fatalf("expected admin.user.create audit log, got %#v", body[0])
	}
}
