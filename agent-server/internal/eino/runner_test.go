package eino

import (
	"testing"

	agentv1 "k8s-ai-ops/proto/agent/v1"
)

func TestBuildMessagesAppendsSystemContext(t *testing.T) {
	request := &agentv1.AgentRunRequest{
		Messages: []*agentv1.Message{
			{Role: "user", Content: "check dev namespace"},
		},
		RuntimeContext: &agentv1.RuntimeContext{
			CurrentUser:       "operator-a",
			AllowedNamespaces: []string{"dev"},
		},
	}
	messages := buildMessages(request)
	if len(messages) != 2 {
		t.Fatalf("expected system + 1 user message, got %d", len(messages))
	}
	if messages[0].Role != "system" || messages[0].Content == "" {
		t.Fatalf("expected system context message, got %#v", messages[0])
	}
}

func TestBuildMessagesHandlesEmptyHistoryWithMessageField(t *testing.T) {
	request := &agentv1.AgentRunRequest{
		Message: "check dev namespace",
		RuntimeContext: &agentv1.RuntimeContext{
			CurrentUser: "operator-a",
		},
	}
	messages := buildMessages(request)
	if len(messages) != 2 {
		t.Fatalf("expected system + 1 user message, got %d", len(messages))
	}
	if messages[1].Content != "check dev namespace" {
		t.Fatalf("unexpected message content: %s", messages[1].Content)
	}
}

func TestBuildMessagesPreservesConversationHistory(t *testing.T) {
	request := &agentv1.AgentRunRequest{
		Messages: []*agentv1.Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi there"},
			{Role: "user", Content: "check dev"},
		},
		RuntimeContext: &agentv1.RuntimeContext{
			CurrentUser:       "operator-a",
			AllowedNamespaces: []string{"dev"},
			RecentResources: []*agentv1.ResourceRef{{
				Kind:      "Pod",
				Namespace: "dev",
				Name:      "api-7b8f9",
			}},
		},
	}
	messages := buildMessages(request)
	if len(messages) != 4 {
		t.Fatalf("expected system + 3 history messages, got %d", len(messages))
	}
}
