package server

import (
	"context"

	agentv1 "k8s-ai-ops/proto/agent/v1"
	agent "k8s-ai-ops/agent-server/internal/eino"
)

// MockRunner is a test-only Runner that replays a pre-configured event list.
type MockRunner struct {
	events []*agentv1.StreamEvent
}

// NewMockRunner creates a new MockRunner with no events.
func NewMockRunner() *MockRunner {
	return &MockRunner{}
}

// RunStream sends each pre-configured event via the sender.
func (r *MockRunner) RunStream(_ context.Context, _ *agentv1.AgentRunRequest, sender agent.StreamSender) error {
	for _, event := range r.events {
		if err := sender.Send(event); err != nil {
			return err
		}
	}
	return nil
}

// SetEvents configures the events to replay.
func (r *MockRunner) SetEvents(events []*agentv1.StreamEvent) {
	r.events = events
}
