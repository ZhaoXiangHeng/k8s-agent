package eino

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	agentv1 "k8s-ai-ops/proto/agent/v1"

	"k8s-ai-ops/agent-server/internal/eino/llm"
	einomcp "k8s-ai-ops/agent-server/internal/eino/mcp"
)

// Runner executes AI agent runs using Eino ReAct + MCP tools.
type Runner struct {
	mcpClient *einomcp.Client
}

// NewRunner creates a Runner connected to the given MCP server.
func NewRunner(ctx context.Context, mcpServerURL string) (*Runner, error) {
	mcpClient, err := einomcp.NewClient(ctx, mcpServerURL)
	if err != nil {
		return nil, fmt.Errorf("mcp client init: %w", err)
	}
	return &Runner{mcpClient: mcpClient}, nil
}

// StreamSender is the interface the runner uses to emit stream events.
type StreamSender interface {
	Send(*agentv1.StreamEvent) error
}

// RunStream executes the ReAct agent loop and streams events back via sender.
func (r *Runner) RunStream(ctx context.Context, request *agentv1.AgentRunRequest, sender StreamSender) error {
	// Create ChatModel from request config
	chatModel, err := llm.NewFromConfig(ctx, request.GetModel())
	if err != nil {
		return fmt.Errorf("llm factory: %w", err)
	}

	// Get tools with user_id injected (LLM never sees user_id parameter)
	userID := request.GetUser().GetId()
	tools := r.mcpClient.ToolsForUser(userID)

	// Create ReAct agent with streaming
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
		MaxStep: 15,
	})
	if err != nil {
		return fmt.Errorf("react agent: %w", err)
	}

	messages := buildMessages(request)

	sr, err := agent.Stream(ctx, messages)
	if err != nil {
		return fmt.Errorf("agent stream: %w", err)
	}
	defer sr.Close()

	for {
		msg, err := sr.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			sender.Send(&agentv1.StreamEvent{
				EventId:   uuid.New().String(),
				Timestamp: time.Now().UnixMilli(),
				Event: &agentv1.StreamEvent_Error{
					Error: &agentv1.ErrorEvent{
						Code:    "STREAM_ERROR",
						Message: err.Error(),
					},
				},
			})
			continue
		}

		switch msg.Role {
		case schema.Assistant:
			if len(msg.ToolCalls) > 0 {
				// Emit one ToolCall event per tool call
				for _, tc := range msg.ToolCalls {
					sender.Send(&agentv1.StreamEvent{
						EventId:   uuid.New().String(),
						Timestamp: time.Now().UnixMilli(),
						Event: &agentv1.StreamEvent_ToolCall{
							ToolCall: &agentv1.ToolCallEvent{
								ToolName:      tc.Function.Name,
								ArgumentsJson: tc.Function.Arguments,
							},
						},
					})
				}
			}
			if msg.Content != "" {
				sender.Send(&agentv1.StreamEvent{
					EventId:   uuid.New().String(),
					Timestamp: time.Now().UnixMilli(),
					Event: &agentv1.StreamEvent_Thinking{
						Thinking: &agentv1.ThinkingEvent{
							Content: msg.Content,
						},
					},
				})
			}
		case schema.Tool:
			sender.Send(&agentv1.StreamEvent{
				EventId:   uuid.New().String(),
				Timestamp: time.Now().UnixMilli(),
				Event: &agentv1.StreamEvent_ToolResult{
					ToolResult: &agentv1.ToolResultEvent{
						ToolName:  msg.ToolName,
						ResultJson: msg.Content,
						Success:   true,
					},
				},
			})
		}
	}

	sender.Send(&agentv1.StreamEvent{
		EventId:   uuid.New().String(),
		Timestamp: time.Now().UnixMilli(),
		Event: &agentv1.StreamEvent_Complete{
			Complete: &agentv1.CompleteEvent{
				Summary:   "agent loop completed",
				Resources: nil,
			},
		},
	})

	return nil
}

// buildMessages constructs the message list for the ReAct agent from the gRPC request.
func buildMessages(request *agentv1.AgentRunRequest) []*schema.Message {
	messages := []*schema.Message{
		{Role: schema.System, Content: systemContext(request)},
	}
	for _, msg := range request.GetMessages() {
		role := schema.RoleType(msg.GetRole())
		if role == "" {
			role = schema.User
		}
		messages = append(messages, &schema.Message{
			Role:    role,
			Content: msg.GetContent(),
		})
	}
	if len(request.GetMessages()) == 0 && request.GetMessage() != "" {
		messages = append(messages, schema.UserMessage(request.GetMessage()))
	}
	return messages
}

// systemContext builds the system prompt with runtime context.
func systemContext(request *agentv1.AgentRunRequest) string {
	var b strings.Builder
	b.WriteString("You are a Kubernetes AI operations assistant. ")
	b.WriteString("You have access to Kubernetes tools via MCP. ")
	b.WriteString("Use the tools to fetch real cluster data when the user asks questions about resources. ")
	b.WriteString("Analyze the returned data and provide helpful insights. ")
	if user := request.GetRuntimeContext().GetCurrentUser(); user != "" {
		b.WriteString("Current user: ")
		b.WriteString(user)
		b.WriteString(". ")
	}
	if namespaces := request.GetRuntimeContext().GetAllowedNamespaces(); len(namespaces) > 0 {
		b.WriteString("Allowed namespaces: ")
		b.WriteString(strings.Join(namespaces, ", "))
		b.WriteString(". ")
	}
	return b.String()
}
