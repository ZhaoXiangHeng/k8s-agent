package eino

import (
	"testing"

	agentv1 "k8s-ai-ops/proto/agent/v1"
)

func TestBuildMessagesNoContext(t *testing.T) {
	request := &agentv1.AgentRunRequest{
		CurrentInput: "check dev namespace",
		User:         &agentv1.UserContext{Username: "operator-a"},
		Permissions: []*agentv1.Permission{
			{Namespace: "dev", Resource: "pods", Verbs: []string{"get", "list"}},
		},
	}
	messages := buildMessages(request)
	if len(messages) != 1 {
		t.Fatalf("expected 1 user message, got %d", len(messages))
	}
	if messages[0].Role != "user" || messages[0].Content != "check dev namespace" {
		t.Fatalf("expected user message, got %#v", messages[0])
	}
}

func TestBuildMessagesHandlesEmptyHistoryWithCurrentInput(t *testing.T) {
	request := &agentv1.AgentRunRequest{
		CurrentInput: "check dev namespace",
		User:         &agentv1.UserContext{Username: "operator-a"},
	}
	messages := buildMessages(request)
	if len(messages) != 1 {
		t.Fatalf("expected 1 user message, got %d", len(messages))
	}
	if messages[0].Content != "check dev namespace" {
		t.Fatalf("unexpected message content: %s", messages[0].Content)
	}
}

func TestBuildMessagesPreservesConversationHistory(t *testing.T) {
	request := &agentv1.AgentRunRequest{
		ContextMessages: []*agentv1.Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi there"},
		},
		CurrentInput: "check dev",
		User:         &agentv1.UserContext{Username: "operator-a"},
	}
	messages := buildMessages(request)
	if len(messages) != 3 {
		t.Fatalf("expected 2 context messages + current input = 3, got %d", len(messages))
	}
	if messages[2].Content != "check dev" {
		t.Fatalf("expected current input appended last, got %#v", messages[2])
	}
}
