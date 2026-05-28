package eino

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	agentv1 "k8s-ai-ops/proto/agent/v1"

	"k8s-ai-ops/agent-server/internal/eino/llm"
	einomcp "k8s-ai-ops/agent-server/internal/eino/mcp"
	localskill "k8s-ai-ops/agent-server/internal/eino/skill"
)

var pkgLog = logrus.WithField("component", "agent-server/eino")

// Runner 使用 Eino ADK ChatModelAgent、MCP 工具和 Skills 执行 Agent Run。
type Runner struct {
	mcpClient *einomcp.Client
	skillMW   adk.ChatModelAgentMiddleware
}

// NewRunner 创建连接到指定 MCP Server 并加载 Skills 的 Runner。
func NewRunner(ctx context.Context, mcpServerURL, skillsDir string) (*Runner, error) {
	mcpClient, err := einomcp.NewClient(ctx, mcpServerURL)
	if err != nil {
		return nil, fmt.Errorf("mcp client init: %w", err)
	}

	// 创建基于本地文件系统的 skill backend
	fsBackend := localskill.NewOSBackend()
	skillBackend, err := skill.NewBackendFromFilesystem(ctx, &skill.BackendFromFilesystemConfig{
		Backend: fsBackend,
		BaseDir: skillsDir,
	})
	if err != nil {
		return nil, fmt.Errorf("skill backend init: %w", err)
	}

	// 创建 skill 中间件（渐进式披露：先只暴露 name+description，LLM 调用时才加载完整 SKILL.md）
	skillMW, err := skill.NewMiddleware(ctx, &skill.Config{
		Backend: skillBackend,
	})
	if err != nil {
		return nil, fmt.Errorf("skill middleware init: %w", err)
	}

	return &Runner{mcpClient: mcpClient, skillMW: skillMW}, nil
}

// StreamSender 是 Runner 发送流事件所需的最小接口。
type StreamSender interface {
	Send(*agentv1.StreamEvent) error
}

// RunStream 执行 ADK ChatModelAgent 循环，并通过 sender 回传流事件。
func (r *Runner) RunStream(ctx context.Context, request *agentv1.AgentRunRequest, sender StreamSender) error {
	pkgLog.WithField("event", "agent_run_start").WithField("request_id", request.GetRequestId()).WithField("provider_id", request.GetModel().GetProviderId()).WithField("model", request.GetModel().GetModelName()).WithField("context_message_count", len(request.GetContextMessages())).Info("agent run starting")

	chatModel, err := llm.NewFromConfig(ctx, request.GetModel())
	if err != nil {
		pkgLog.WithError(err).WithField("event", "llm_model_create_failed").WithField("request_id", request.GetRequestId()).WithField("provider_id", request.GetModel().GetProviderId()).WithField("model", request.GetModel().GetModelName()).Error("failed to create chat model")
		return fmt.Errorf("llm factory: %w", err)
	}

	// user_id 只能由 agent-server 注入，不能信任 LLM 生成的工具参数。
	userID := request.GetUser().GetId()
	tools := r.mcpClient.ToolsForUser(userID)
	pkgLog.WithField("event", "agent_tools_ready").WithField("request_id", request.GetRequestId()).WithField("user_id", userID).WithField("tool_count", len(tools)).Info("agent tools ready")

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "k8s-ops-agent",
		Description: "Kubernetes AI operations assistant with skills support",
		Instruction: systemContext(request),
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
		MaxIterations: 15,
		Handlers:      []adk.ChatModelAgentMiddleware{r.skillMW},
	})
	if err != nil {
		pkgLog.WithError(err).WithField("event", "agent_create_failed").WithField("request_id", request.GetRequestId()).Error("failed to create ADK ChatModelAgent")
		return fmt.Errorf("create agent: %w", err)
	}

	messages := buildMessages(request)
	iter := agent.Run(ctx, &adk.AgentInput{
		Messages:        messages,
		EnableStreaming: true,
	})

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			pkgLog.WithError(event.Err).WithField("event", "agent_event_error").WithField("request_id", request.GetRequestId()).Error("agent event error")
			if sendErr := sender.Send(&agentv1.StreamEvent{
				EventId:   uuid.New().String(),
				Timestamp: time.Now().UnixMilli(),
				Event: &agentv1.StreamEvent_Error{
					Error: &agentv1.ErrorEvent{
						Code:    "AGENT_ERROR",
						Message: event.Err.Error(),
					},
				},
			}); sendErr != nil {
				return fmt.Errorf("send error event: %w", sendErr)
			}
			continue
		}

		if err := r.handleAgentEvent(ctx, event, request.GetRequestId(), sender); err != nil {
			return err
		}
	}

	// 发送完成事件
	if err := sender.Send(&agentv1.StreamEvent{
		EventId:   uuid.New().String(),
		Timestamp: time.Now().UnixMilli(),
		Event: &agentv1.StreamEvent_Complete{
			Complete: &agentv1.CompleteEvent{
				Summary:   "agent loop completed",
				Resources: nil,
			},
		},
	}); err != nil {
		return fmt.Errorf("send complete event: %w", err)
	}

	pkgLog.WithField("event", "agent_run_complete").WithField("request_id", request.GetRequestId()).Info("agent run complete")
	return nil
}

// handleAgentEvent 将一个 AgentEvent 映射为 gRPC StreamEvent 并发送。
func (r *Runner) handleAgentEvent(ctx context.Context, event *adk.AgentEvent, requestID string, sender StreamSender) error {
	if event.Output == nil || event.Output.MessageOutput == nil {
		return nil
	}

	msgOut := event.Output.MessageOutput

	// 流式输出：累积所有 chunk 得到完整消息
	if msgOut.IsStreaming && msgOut.MessageStream != nil {
		return r.handleStreamingMessage(msgOut, requestID, sender)
	}

	// 非流式消息直接处理
	if msgOut.Message != nil {
		return r.handleMessage(msgOut.Message, requestID, sender)
	}

	return nil
}

// handleStreamingMessage 读取流式消息的所有 chunk，合并后作为完整消息处理。
func (r *Runner) handleStreamingMessage(msgOut *adk.MessageVariant, requestID string, sender StreamSender) error {
	var content strings.Builder
	var toolCalls map[string]*toolCallAccumulator

	for {
		chunk, err := msgOut.MessageStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("stream recv: %w", err)
		}

		content.WriteString(chunk.Content)

		for _, tc := range chunk.ToolCalls {
			if toolCalls == nil {
				toolCalls = make(map[string]*toolCallAccumulator)
			}
			acc, exists := toolCalls[tc.ID]
			if !exists {
				acc = &toolCallAccumulator{name: tc.Function.Name}
				toolCalls[tc.ID] = acc
			}
			acc.arguments.WriteString(tc.Function.Arguments)
		}
	}

	// 先发送工具调用
	for _, acc := range toolCalls {
		pkgLog.WithField("event", "tool_call_emit").WithField("request_id", requestID).WithField("tool_name", acc.name).Info("tool call emitted")
		if err := sender.Send(&agentv1.StreamEvent{
			EventId:   uuid.New().String(),
			Timestamp: time.Now().UnixMilli(),
			Event: &agentv1.StreamEvent_ToolCall{
				ToolCall: &agentv1.ToolCallEvent{
					ToolName:      acc.name,
					ArgumentsJson: acc.arguments.String(),
				},
			},
		}); err != nil {
			return fmt.Errorf("send tool call event: %w", err)
		}
	}

	// 再发送思维内容
	if content.Len() > 0 {
		if err := sender.Send(&agentv1.StreamEvent{
			EventId:   uuid.New().String(),
			Timestamp: time.Now().UnixMilli(),
			Event: &agentv1.StreamEvent_Thinking{
				Thinking: &agentv1.ThinkingEvent{
					Content: content.String(),
				},
			},
		}); err != nil {
			return fmt.Errorf("send thinking event: %w", err)
		}
	}

	return nil
}

type toolCallAccumulator struct {
	name      string
	arguments strings.Builder
}

// handleMessage 处理非流式消息。
func (r *Runner) handleMessage(msg *schema.Message, requestID string, sender StreamSender) error {
	switch msg.Role {
	case schema.Assistant:
		// 工具调用
		for _, tc := range msg.ToolCalls {
			pkgLog.WithField("event", "tool_call_emit").WithField("request_id", requestID).WithField("tool_name", tc.Function.Name).Info("tool call emitted")
			if err := sender.Send(&agentv1.StreamEvent{
				EventId:   uuid.New().String(),
				Timestamp: time.Now().UnixMilli(),
				Event: &agentv1.StreamEvent_ToolCall{
					ToolCall: &agentv1.ToolCallEvent{
						ToolName:      tc.Function.Name,
						ArgumentsJson: tc.Function.Arguments,
					},
				},
			}); err != nil {
				return fmt.Errorf("send tool call event: %w", err)
			}
		}
		// 文本内容
		if msg.Content != "" {
			if err := sender.Send(&agentv1.StreamEvent{
				EventId:   uuid.New().String(),
				Timestamp: time.Now().UnixMilli(),
				Event: &agentv1.StreamEvent_Thinking{
					Thinking: &agentv1.ThinkingEvent{
						Content: msg.Content,
					},
				},
			}); err != nil {
				return fmt.Errorf("send thinking event: %w", err)
			}
		}
	case schema.Tool:
		pkgLog.WithField("event", "tool_result_emit").WithField("request_id", requestID).WithField("tool_name", msg.ToolName).Info("tool result emitted")
		if err := sender.Send(&agentv1.StreamEvent{
			EventId:   uuid.New().String(),
			Timestamp: time.Now().UnixMilli(),
			Event: &agentv1.StreamEvent_ToolResult{
				ToolResult: &agentv1.ToolResultEvent{
					ToolName:   msg.ToolName,
					ResultJson: msg.Content,
					Success:    true,
				},
			},
		}); err != nil {
			return fmt.Errorf("send tool result event: %w", err)
		}
	}

	return nil
}

// buildMessages 根据 gRPC 请求构造消息列表（不含 system prompt，由 agent Instruction 负责）。
func buildMessages(request *agentv1.AgentRunRequest) []*schema.Message {
	var messages []*schema.Message
	for _, msg := range request.GetContextMessages() {
		role := schema.RoleType(msg.GetRole())
		if role == "" {
			role = schema.User
		}
		messages = append(messages, &schema.Message{
			Role:    role,
			Content: msg.GetContent(),
		})
	}
	if request.GetCurrentInput() != "" {
		messages = append(messages, schema.UserMessage(request.GetCurrentInput()))
	}
	return messages
}

// systemContext 根据运行时上下文构造系统提示词。
func systemContext(request *agentv1.AgentRunRequest) string {
	var b strings.Builder
	b.WriteString("You are a Kubernetes AI operations assistant. ")
	b.WriteString("You have access to Kubernetes tools via MCP. ")
	b.WriteString("Use the tools to fetch real cluster data when the user asks questions about resources. ")
	b.WriteString("Analyze the returned data and provide helpful insights. ")
	b.WriteString("You also have access to Skills. Use the skill tool to load specialized knowledge ")
	b.WriteString("when you need guidance for specific Kubernetes operational tasks. ")

	// 范围约束：只处理 K8s 运维相关内容
	b.WriteString("CRITICAL SCOPE CONSTRAINT: You MUST ONLY handle Kubernetes cluster operations tasks. ")
	b.WriteString("You MUST REJECT any request unrelated to K8s operations, including: general programming, ")
	b.WriteString("code review, architecture design, non-K8s cloud services, translation, document writing, ")
	b.WriteString("math problems, casual chat, entertainment, or security attacks. ")
	b.WriteString("For K8s-related shell commands (kubectl, crictl, journalctl) you may assist. ")
	b.WriteString("When rejecting, briefly state your scope and list supported K8s operations categories. ")
	b.WriteString("Do NOT attempt to help with out-of-scope requests under any circumstances. ")
	b.WriteString("The scope-guard skill provides detailed scope rules — load it if the user disputes your rejection. ")

	if user := request.GetUser().GetUsername(); user != "" {
		b.WriteString("Current user: ")
		b.WriteString(user)
		b.WriteString(". ")
	}
	if namespaces := allowedNamespaces(request.GetPermissions()); len(namespaces) > 0 {
		b.WriteString("Allowed namespaces: ")
		b.WriteString(strings.Join(namespaces, ", "))
		b.WriteString(". ")
	}
	return b.String()
}

func allowedNamespaces(permissions []*agentv1.Permission) []string {
	seen := map[string]bool{}
	namespaces := []string{}
	for _, permission := range permissions {
		namespace := permission.GetNamespace()
		if namespace == "" || seen[namespace] {
			continue
		}
		seen[namespace] = true
		namespaces = append(namespaces, namespace)
	}
	return namespaces
}
