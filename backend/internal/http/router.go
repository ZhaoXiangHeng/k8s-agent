package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	agentclient "k8s-ai-ops/backend/internal/agent"
	"k8s-ai-ops/backend/internal/chat"
	k8sops "k8s-ai-ops/backend/internal/k8s"
	"k8s-ai-ops/backend/internal/llm"
	"k8s-ai-ops/backend/internal/store"
	agentv1 "k8s-ai-ops/proto/agent/v1"
)

type Server struct {
	store       store.Store
	models      []llm.ModelBinding
	pods        []chat.AbnormalPod
	rbacApplier RBACApplier
	agentClient agentclient.Client
}

type RBACApplier interface {
	ApplyUserNamespacePermissions(context.Context, k8sops.UserNamespacePermissions) error
}

type ErrorResponse struct {
	Error APIError `json:"error"`
}

type APIError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

type chatMessageRequest struct {
	ModelID string `json:"modelId"`
	Content string `json:"content"`
}

type createUserRequest struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	DisplayName string `json:"displayName"`
}

type updatePermissionsRequest struct {
	Permissions []permissionRequest `json:"permissions"`
}

type permissionRequest struct {
	Namespace string   `json:"namespace"`
	APIGroup  string   `json:"apiGroup"`
	Resource  string   `json:"resource"`
	Verbs     []string `json:"verbs"`
}

type createProviderRequest struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	BaseURL  string `json:"baseUrl"`
	APIKey   string `json:"apiKey"`
	Enabled  bool   `json:"enabled"`
}

type createModelRequest struct {
	ProviderID        string `json:"providerId"`
	ModelName         string `json:"modelName"`
	DisplayName       string `json:"displayName"`
	SupportsTools     bool   `json:"supportsTools"`
	SupportsStreaming bool   `json:"supportsStreaming"`
	Enabled           bool   `json:"enabled"`
}

func NewRouter() http.Handler {
	return NewServer(store.NewMemoryStore())
}

func NewServer(stores ...store.Store) *Server {
	activeStore := store.Store(store.NewMemoryStore())
	if len(stores) > 0 && stores[0] != nil {
		activeStore = stores[0]
	}
	return &Server{
		store: activeStore,
		models: []llm.ModelBinding{
			{ModelID: "mock-local", DisplayName: "Mock Local", ProviderID: "mock", IsDefault: true, SupportsTools: true, SupportsStreaming: true},
		},
		pods: []chat.AbnormalPod{
			{
				Namespace:    "dev",
				Name:         "api-7b8f9",
				Phase:        "Pending",
				Reason:       "ImagePullBackOff",
				Message:      "Back-off pulling image",
				RestartCount: 0,
				Node:         "kind-worker",
			},
			{
				Namespace:    "dev",
				Name:         "worker-5d9c7",
				Phase:        "Running",
				Reason:       "CrashLoopBackOff",
				Message:      "Container exits after startup",
				RestartCount: 6,
				Node:         "kind-worker2",
			},
		},
	}
}

func (s *Server) SetRBACApplier(applier RBACApplier) {
	s.rbacApplier = applier
}

func (s *Server) SetAgentClient(client agentclient.Client) {
	s.agentClient = client
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/healthz":
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	case r.Method == http.MethodGet && r.URL.Path == "/api/me":
		s.writeCurrentUser(w)
	case r.Method == http.MethodGet && r.URL.Path == "/api/operator/permissions":
		s.writeOperatorPermissions(w)
	case r.Method == http.MethodGet && r.URL.Path == "/api/operator/llm-models":
		s.writeOperatorModels(w)
	case r.Method == http.MethodGet && r.URL.Path == "/api/admin/users":
		s.listAdminUsers(w)
	case r.Method == http.MethodPost && r.URL.Path == "/api/admin/users":
		s.createAdminUser(w, r)
	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/api/admin/users/") && strings.HasSuffix(r.URL.Path, "/permissions"):
		s.updateUserPermissions(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/api/admin/audit-logs":
		s.listAuditLogs(w)
	case r.Method == http.MethodGet && r.URL.Path == "/api/admin/llm/providers":
		s.listLLMProviders(w)
	case r.Method == http.MethodPost && r.URL.Path == "/api/admin/llm/providers":
		s.createLLMProvider(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/api/admin/llm/models":
		s.listLLMModels(w)
	case r.Method == http.MethodPost && r.URL.Path == "/api/admin/llm/models":
		s.createLLMModel(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/operator/chat/sessions":
		s.createChatSession(w)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/operator/chat/sessions/") && strings.HasSuffix(r.URL.Path, "/messages"):
		s.createChatMessage(w, r)
	default:
		writeError(w, http.StatusNotFound, "NOT_FOUND", "The requested API endpoint was not found.")
	}
}

func (s *Server) writeCurrentUser(w http.ResponseWriter) {
	currentUser := s.store.CurrentDemoUser()
	writeJSON(w, http.StatusOK, map[string]any{
		"id":          currentUser.ID,
		"username":    currentUser.Username,
		"displayName": currentUser.DisplayName,
		"email":       currentUser.Email,
		"role":        currentUser.Role,
		"status":      currentUser.Status,
	})
}

func (s *Server) writeOperatorPermissions(w http.ResponseWriter) {
	permissions := s.store.ListUserPermissions(s.store.CurrentDemoUser().ID)
	response := make([]map[string]any, 0, len(permissions))
	for _, permission := range permissions {
		response = append(response, map[string]any{
			"id":        permission.ID,
			"namespace": permission.Namespace,
			"apiGroup":  permission.APIGroup,
			"resource":  permission.Resource,
			"verbs":     permission.Verbs,
			"enabled":   permission.Enabled,
		})
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) writeOperatorModels(w http.ResponseWriter) {
	response := make([]map[string]any, 0, len(s.models))
	for _, model := range s.models {
		response = append(response, map[string]any{
			"modelId":           model.ModelID,
			"displayName":       model.DisplayName,
			"providerId":        model.ProviderID,
			"isDefault":         model.IsDefault,
			"supportsTools":     model.SupportsTools,
			"supportsStreaming": model.SupportsStreaming,
		})
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) listAdminUsers(w http.ResponseWriter) {
	users := s.store.ListUsers()
	response := make([]map[string]any, 0, len(users))
	for _, user := range users {
		response = append(response, userResponse(user))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) createAdminUser(w http.ResponseWriter, r *http.Request) {
	var request createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request body must be valid JSON.")
		return
	}
	role := store.UserRole(request.Role)
	if role != store.RoleAdmin && role != store.RoleOperator {
		writeError(w, http.StatusBadRequest, "INVALID_USER_ROLE", "User role must be admin or operator.")
		return
	}
	user := s.store.CreateUser(store.User{
		Username:    request.Username,
		DisplayName: request.DisplayName,
		Email:       request.Email,
		Role:        role,
		Status:      "active",
	})
	s.audit("admin.user.create", "user", user.ID, "", "", "", true, "user created")
	writeJSON(w, http.StatusCreated, userResponse(user))
}

func (s *Server) updateUserPermissions(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/admin/users/"), "/permissions")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "INVALID_USER_ID", "User id is required.")
		return
	}
	var request updatePermissionsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request body must be valid JSON.")
		return
	}
	permissions := make([]store.Permission, 0, len(request.Permissions))
	for _, item := range request.Permissions {
		permissions = append(permissions, store.Permission{
			Namespace: item.Namespace,
			APIGroup:  item.APIGroup,
			Resource:  item.Resource,
			Verbs:     item.Verbs,
		})
	}
	savedPermissions := s.store.ReplaceUserPermissions(userID, permissions)
	if s.rbacApplier != nil {
		if err := s.applyRBAC(r.Context(), userID, savedPermissions); err != nil {
			s.audit("admin.user.permissions.update", "user", userID, "", "", "", false, "k8s rbac apply failed")
			writeError(w, http.StatusInternalServerError, "K8S_RBAC_APPLY_FAILED", "Failed to apply Kubernetes RBAC resources.")
			return
		}
	}
	s.audit("admin.user.permissions.update", "user", userID, "", "", "", true, "permissions replaced")
	s.writeOperatorPermissions(w)
}

func (s *Server) applyRBAC(ctx context.Context, userID string, permissions []store.Permission) error {
	for namespace, specs := range groupPermissionsByNamespace(permissions) {
		if err := s.rbacApplier.ApplyUserNamespacePermissions(ctx, k8sops.UserNamespacePermissions{
			UserID:    userID,
			Namespace: namespace,
			Rules:     specs,
		}); err != nil {
			return err
		}
	}
	return nil
}

func groupPermissionsByNamespace(permissions []store.Permission) map[string][]k8sops.PermissionSpec {
	grouped := map[string][]k8sops.PermissionSpec{}
	for _, permission := range permissions {
		if !permission.Enabled || permission.Namespace == "" || permission.Resource == "" || len(permission.Verbs) == 0 {
			continue
		}
		grouped[permission.Namespace] = append(grouped[permission.Namespace], k8sops.PermissionSpec{
			APIGroup: permission.APIGroup,
			Resource: permission.Resource,
			Verbs:    append([]string(nil), permission.Verbs...),
		})
	}
	return grouped
}

func (s *Server) listAuditLogs(w http.ResponseWriter) {
	logs := s.store.ListAuditLogs()
	response := make([]map[string]any, 0, len(logs))
	for _, log := range logs {
		response = append(response, auditLogResponse(log))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) listLLMProviders(w http.ResponseWriter) {
	providers := s.store.ListLLMProviders()
	response := make([]map[string]any, 0, len(providers))
	for _, provider := range providers {
		response = append(response, providerResponse(provider))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) createLLMProvider(w http.ResponseWriter, r *http.Request) {
	var request createProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request body must be valid JSON.")
		return
	}
	if request.Protocol != string(llm.ProtocolOpenAI) && request.Protocol != string(llm.ProtocolAnthropic) {
		writeError(w, http.StatusBadRequest, "INVALID_LLM_PROTOCOL", "LLM protocol must be openai or anthropic.")
		return
	}
	provider := s.store.CreateLLMProvider(store.LLMProvider{
		Name:             request.Name,
		Protocol:         request.Protocol,
		BaseURL:          request.BaseURL,
		APIKeyCiphertext: request.APIKey,
		Enabled:          request.Enabled,
	})
	s.audit("admin.llm.provider.create", "llm_provider", provider.ID, "", "", "", true, "llm provider created")
	writeJSON(w, http.StatusCreated, providerResponse(provider))
}

func (s *Server) listLLMModels(w http.ResponseWriter) {
	models := s.store.ListLLMModels()
	response := make([]map[string]any, 0, len(models))
	for _, model := range models {
		response = append(response, modelResponse(model))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) createLLMModel(w http.ResponseWriter, r *http.Request) {
	var request createModelRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request body must be valid JSON.")
		return
	}
	model := s.store.CreateLLMModel(store.LLMModel{
		ProviderID:        request.ProviderID,
		ModelName:         request.ModelName,
		DisplayName:       request.DisplayName,
		SupportsTools:     request.SupportsTools,
		SupportsStreaming: request.SupportsStreaming,
		Enabled:           request.Enabled,
	})
	s.audit("admin.llm.model.create", "llm_model", model.ID, "", "", "", true, "llm model created")
	writeJSON(w, http.StatusCreated, modelResponse(model))
}

func (s *Server) createChatSession(w http.ResponseWriter) {
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":        "demo-session",
		"userId":    s.store.CurrentDemoUser().ID,
		"status":    "active",
		"createdAt": time.Now().UTC().Format(time.RFC3339),
	})
}

func userResponse(user store.User) map[string]any {
	return map[string]any{
		"id":          user.ID,
		"username":    user.Username,
		"displayName": user.DisplayName,
		"email":       user.Email,
		"role":        user.Role,
		"status":      user.Status,
	}
}

func providerResponse(provider store.LLMProvider) map[string]any {
	return map[string]any{
		"id":               provider.ID,
		"name":             provider.Name,
		"protocol":         provider.Protocol,
		"baseUrl":          provider.BaseURL,
		"enabled":          provider.Enabled,
		"apiKeyConfigured": provider.APIKeyCiphertext != "",
	}
}

func modelResponse(model store.LLMModel) map[string]any {
	return map[string]any{
		"id":                model.ID,
		"providerId":        model.ProviderID,
		"modelName":         model.ModelName,
		"displayName":       model.DisplayName,
		"supportsTools":     model.SupportsTools,
		"supportsStreaming": model.SupportsStreaming,
		"enabled":           model.Enabled,
	}
}

func auditLogResponse(log store.AuditLog) map[string]any {
	return map[string]any{
		"id":          log.ID,
		"actorUserId": log.ActorUserID,
		"action":      log.Action,
		"targetType":  log.TargetType,
		"targetId":    log.TargetID,
		"namespace":   log.Namespace,
		"resource":    log.Resource,
		"verb":        log.Verb,
		"allowed":     log.Allowed,
		"reason":      log.Reason,
		"createdAt":   log.CreatedAt,
	}
}

func (s *Server) createChatMessage(w http.ResponseWriter, r *http.Request) {
	var request chatMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request body must be valid JSON.")
		return
	}
	if !s.isModelAllowed(request.ModelID) {
		s.audit("operator.chat.message.create", "chat_message", "demo-message", "", "", "", false, "model not allowed")
		writeError(w, http.StatusForbidden, "LLM_MODEL_NOT_ALLOWED", "The selected model is not assigned to the current user.")
		return
	}

	if s.agentClient == nil {
		s.agentClient = localAgentClient{pods: s.pods}
	}

	reader, err := s.agentClient.RunStream(r.Context(), s.buildAgentRunRequest(request))
	if err != nil {
		s.audit("operator.chat.message.create", "chat_message", "demo-message", "", "", "", false, "agent server unavailable")
		writeError(w, http.StatusBadGateway, "AGENT_SERVER_UNAVAILABLE", "Agent server is unavailable.")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Streaming not supported.")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {
		event, err := reader.Recv()
		if err != nil {
			break
		}
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	s.audit("operator.chat.message.create", "chat_message", "demo-message", "", "", "", true, "inspection completed")
}

func (s *Server) buildAgentRunRequest(request chatMessageRequest) *agentv1.AgentRunRequest {
	currentUser := s.store.CurrentDemoUser()
	return &agentv1.AgentRunRequest{
		RequestId: "req-demo",
		SessionId: "demo-session",
		MessageId: "demo-message",
		User: &agentv1.UserContext{
			Id:       currentUser.ID,
			Username: currentUser.Username,
		},
		Model: &agentv1.ModelRuntimeConfig{
			ProviderId:        "mock",
			Protocol:          string(llm.ProtocolMock),
			ModelName:         request.ModelID,
			SupportsTools:     true,
			SupportsStreaming: true,
		},
		Messages: []*agentv1.Message{
			{Role: "user", Content: request.Content},
		},
		Message:        request.Content,
		Permissions:    s.agentPermissionsForCurrentUser(),
		Tools:          s.agentToolsForCurrentUser(),
		RuntimeContext: s.agentRuntimeContextForCurrentUser(),
	}
}

func (s *Server) agentPermissionsForCurrentUser() []*agentv1.Permission {
	permissions := s.store.ListUserPermissions(s.store.CurrentDemoUser().ID)
	result := make([]*agentv1.Permission, 0, len(permissions))
	for _, permission := range permissions {
		if !permission.Enabled {
			continue
		}
		result = append(result, &agentv1.Permission{
			Namespace: permission.Namespace,
			ApiGroup:  permission.APIGroup,
			Resource:  permission.Resource,
			Verbs:     append([]string(nil), permission.Verbs...),
		})
	}
	return result
}

func (s *Server) agentToolsForCurrentUser() []*agentv1.ToolRef {
	permissions := s.store.ListUserPermissions(s.store.CurrentDemoUser().ID)
	tools := []*agentv1.ToolRef{}
	for _, permission := range permissions {
		if !permission.Enabled {
			continue
		}
		for _, verb := range permission.Verbs {
			tools = append(tools, &agentv1.ToolRef{
				Name:      toolNameFor(permission.Resource, verb),
				Namespace: permission.Namespace,
				ApiGroup:  permission.APIGroup,
				Resource:  permission.Resource,
				Verb:      verb,
			})
		}
	}
	return tools
}

func (s *Server) agentRuntimeContextForCurrentUser() *agentv1.RuntimeContext {
	permissions := s.store.ListUserPermissions(s.store.CurrentDemoUser().ID)
	namespaceSet := map[string]bool{}
	namespaces := []string{}
	for _, permission := range permissions {
		if permission.Enabled && permission.Namespace != "" && !namespaceSet[permission.Namespace] {
			namespaceSet[permission.Namespace] = true
			namespaces = append(namespaces, permission.Namespace)
		}
	}
	recentResources := make([]*agentv1.ResourceRef, 0, len(s.pods))
	for _, pod := range s.pods {
		recentResources = append(recentResources, &agentv1.ResourceRef{
			Kind:      "Pod",
			Namespace: pod.Namespace,
			Name:      pod.Name,
			Summary:   pod.Reason,
		})
	}
	return &agentv1.RuntimeContext{
		CurrentUser:       s.store.CurrentDemoUser().Username,
		AllowedNamespaces: namespaces,
		RecentResources:   recentResources,
	}
}

func toolNameFor(resource, verb string) string {
	switch {
	case resource == "pods" && verb == "list":
		return "list_pods"
	case resource == "pods" && verb == "get":
		return "get_pod"
	case resource == "pods/log" && verb == "get":
		return "get_pod_logs"
	case resource == "events" && verb == "list":
		return "list_events"
	case resource == "deployments" && verb == "list":
		return "list_deployments"
	case resource == "deployments" && verb == "patch":
		return "restart_deployment"
	default:
		return resource + "_" + verb
	}
}

type eventStreamReader struct {
	events []*agentv1.StreamEvent
	pos    int
}

func (r *eventStreamReader) Recv() (*agentv1.StreamEvent, error) {
	if r.pos >= len(r.events) {
		return nil, io.EOF
	}
	event := r.events[r.pos]
	r.pos++
	return event, nil
}

type localAgentClient struct {
	pods []chat.AbnormalPod
}

func (c localAgentClient) RunStream(_ context.Context, request *agentv1.AgentRunRequest) (agentclient.StreamReader, error) {
	var resources []*agentv1.ResourceResult
	for _, pod := range c.pods {
		resources = append(resources, &agentv1.ResourceResult{
			Kind:      "Pod",
			Namespace: pod.Namespace,
			Name:      pod.Name,
			Status:    pod.Reason,
		})
	}
	return &eventStreamReader{events: []*agentv1.StreamEvent{
		{EventId: "1", Event: &agentv1.StreamEvent_Thinking{Thinking: &agentv1.ThinkingEvent{Content: "正在检查 dev 命名空间..."}}},
		{EventId: "2", Event: &agentv1.StreamEvent_ToolCall{ToolCall: &agentv1.ToolCallEvent{ToolName: "list_pods", ArgumentsJson: `{"namespace":"dev"}`}}},
		{EventId: "3", Event: &agentv1.StreamEvent_Complete{Complete: &agentv1.CompleteEvent{Summary: "dev namespace 中有 2 个异常 Pod。", Resources: resources}}},
	}}, nil
}

func (s *Server) isModelAllowed(modelID string) bool {
	for _, model := range s.models {
		if model.ModelID == modelID {
			return true
		}
	}
	return false
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{
		Error: APIError{
			Code:      code,
			Message:   message,
			RequestID: "req-demo",
		},
	})
}

func (s *Server) audit(action, targetType, targetID, namespace, resource, verb string, allowed bool, reason string) {
	s.store.AppendAuditLog(store.AuditLog{
		ActorUserID: s.store.CurrentDemoUser().ID,
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		Namespace:   namespace,
		Resource:    resource,
		Verb:        verb,
		Allowed:     allowed,
		Reason:      reason,
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
