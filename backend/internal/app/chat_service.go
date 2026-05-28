package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"k8s-ai-ops/backend/internal/domain"
	agentv1 "k8s-ai-ops/proto/agent/v1"
)

// ChatService 编排 Chat 会话的完整用例流程。
type ChatService struct {
	repos       *domain.Repositories
	agentClient AgentClient
	cipher      SecretCipher
}

// NewChatService 创建 Chat 应用服务。
func NewChatService(repos *domain.Repositories, agentClient AgentClient, cipher SecretCipher) *ChatService {
	return &ChatService{repos: repos, agentClient: agentClient, cipher: cipher}
}

// CreateSession 创建新的 Chat 会话并持久化。
func (s *ChatService) CreateSession(ctx context.Context, userID string) (*ChatSessionResponse, error) {
	session := domain.NewChatSession(userID)
	session.ID = "session-" + uuid.New().String()
	if err := s.repos.ChatSessions.Save(ctx, session); err != nil {
		return nil, fmt.Errorf("save session: %w", err)
	}
	return &ChatSessionResponse{
		ID: session.ID, UserID: userID, Title: session.Title,
		Status: "active", CreatedAt: session.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// DeleteSession 删除会话（含所有权校验）。
func (s *ChatService) DeleteSession(ctx context.Context, sessionID, userID string) error {
	session, err := s.repos.ChatSessions.FindByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("find session: %w", err)
	}
	if session.UserID != userID {
		return fmt.Errorf("%w: session %s does not belong to user %s", domain.ErrForbidden, sessionID, userID)
	}
	session.Status = "closed"
	if err := s.repos.ChatSessions.Save(ctx, session); err != nil {
		return fmt.Errorf("close session: %w", err)
	}
	appLog.WithFields(logrus.Fields{
		"event": "session_deleted", "session_id": sessionID,
	}).Info("session closed")
	return nil
}

// ProcessMessage 处理一条 Chat 消息的完整流程：
// 校验模型 → 持久化用户消息 → 解密 Provider → 构建 AgentRequest →
// gRPC 调用 Agent Server → 流式返回事件 → 持久化响应
func (s *ChatService) ProcessMessage(ctx context.Context, input ChatMessageRequest, userID, username, role, sessionID string, sender SSEWriter) error {
	if err := s.ensureSessionOwner(ctx, sessionID, userID); err != nil {
		return err
	}
	if err := s.ensureModelBoundToUser(ctx, input.ModelID, userID, role); err != nil {
		return err
	}

	// 1. 校验模型
	model, err := s.repos.Models.FindByID(ctx, input.ModelID)
	if err != nil {
		return fmt.Errorf("model not available: %w", err)
	}
	if !model.Enabled {
		return fmt.Errorf("model %s is disabled", input.ModelID)
	}

	// 2. 解析 Provider 配置（解密 API Key）
	provider, err := s.repos.Providers.FindByID(ctx, model.ProviderID)
	if err != nil {
		return fmt.Errorf("provider not found: %w", err)
	}
	if !provider.Enabled {
		return fmt.Errorf("provider %s is disabled", provider.ID)
	}
	apiKey, err := s.cipher.Decrypt(provider.APIKeyCiphertext)
	if err != nil {
		return fmt.Errorf("decrypt api key: %w", err)
	}

	// 3. 持久化用户消息
	msgID := "msg-" + uuid.New().String()
	if err := s.repos.ChatMessages.Append(ctx, &domain.ChatMessage{
		ID: msgID, SessionID: sessionID, Role: domain.RoleUser, Content: input.Content,
	}); err != nil {
		return fmt.Errorf("append user message: %w", err)
	}

	permissions, err := s.buildProtoPermissions(ctx, userID)
	if err != nil {
		return err
	}

	// 4. 构建 Agent 请求
	req := &agentv1.AgentRunRequest{
		RequestId: uuid.New().String(),
		SessionId: sessionID,
		MessageId: msgID,
		User:      &agentv1.UserContext{Id: userID, Username: username},
		Model: &agentv1.ModelRuntimeConfig{
			ProviderId:        provider.ID,
			Protocol:          string(provider.Protocol),
			BaseUrl:           provider.BaseURL,
			ApiKey:            apiKey,
			ModelName:         model.ModelName,
			SupportsTools:     model.SupportsTools,
			SupportsStreaming: model.SupportsStreaming,
		},
		ContextMessages: []*agentv1.Message{{Role: "user", Content: input.Content}},
		CurrentInput:    input.Content,
		Permissions:     permissions,
	}

	appLog.WithFields(logrus.Fields{
		"event": "agent_run_start", "request_id": req.RequestId,
		"session_id": sessionID, "model_name": model.ModelName,
	}).Info("forwarding to agent server")

	// 5. 调用 Agent Server
	reader, err := s.agentClient.RunStream(ctx, req)
	if err != nil {
		return fmt.Errorf("agent server: %w", err)
	}

	// 6. 流式转发事件
	var lastSummary string
	for {
		event, err := reader.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			appLog.WithError(err).WithField("request_id", req.RequestId).Error("agent stream error")
			break
		}
		if sender != nil {
			data, _ := json.Marshal(event)
			sender.WriteSSE(data)
		}
		if complete := event.GetComplete(); complete != nil {
			lastSummary = complete.GetSummary()
		}
	}

	// 7. 持久化 assistant 响应
	if err := s.repos.ChatMessages.Append(ctx, &domain.ChatMessage{
		ID: "msg-" + uuid.New().String(), SessionID: sessionID, Role: domain.RoleAssistant, Content: lastSummary,
	}); err != nil {
		return fmt.Errorf("append assistant message: %w", err)
	}
	appLog.WithFields(logrus.Fields{
		"event": "agent_run_complete", "request_id": req.RequestId,
	}).Info("agent stream completed")
	return nil
}

func (s *ChatService) ensureSessionOwner(ctx context.Context, sessionID, userID string) error {
	session, err := s.repos.ChatSessions.FindByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("find chat session: %w", err)
	}
	if session.UserID != userID {
		return fmt.Errorf("%w: session %s does not belong to user %s", domain.ErrForbidden, sessionID, userID)
	}
	if session.Status != "" && session.Status != "active" {
		return fmt.Errorf("%w: session %s is not active", domain.ErrDisabled, sessionID)
	}
	return nil
}

func (s *ChatService) ensureModelBoundToUser(ctx context.Context, modelID, userID, role string) error {
	if role == "admin" {
		return nil
	}
	bindings, err := s.repos.Bindings.FindByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("find user model bindings: %w", err)
	}
	for _, binding := range bindings {
		if binding.ModelID == modelID {
			return nil
		}
	}
	return fmt.Errorf("%w: model %s is not bound to user %s", domain.ErrForbidden, modelID, userID)
}

// buildProtoPermissions 构建 proto 格式的用户权限列表。
func (s *ChatService) buildProtoPermissions(ctx context.Context, userID string) ([]*agentv1.Permission, error) {
	perms, err := s.repos.Permissions.FindByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user permissions: %w", err)
	}
	result := make([]*agentv1.Permission, 0, len(perms))
	for _, p := range perms {
		if !p.Enabled {
			continue
		}
		result = append(result, &agentv1.Permission{
			Namespace: p.Namespace, ApiGroup: p.APIGroup,
			Resource: p.Resource, Verbs: append([]string(nil), p.Verbs...),
		})
	}
	return result, nil
}

// SSEWriter 是 SSE 事件写入的抽象。
type SSEWriter interface {
	WriteSSE(data []byte)
}
