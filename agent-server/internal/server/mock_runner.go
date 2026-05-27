package server

import (
	"context"

	agent "k8s-ai-ops/agent-server/internal/eino"
	agentv1 "k8s-ai-ops/proto/agent/v1"
)

// MockRunner 是测试专用 Runner，用于回放预设事件列表。
type MockRunner struct {
	events []*agentv1.StreamEvent
}

// NewMockRunner 创建不包含事件的 MockRunner。
func NewMockRunner() *MockRunner {
	return &MockRunner{}
}

// RunStream 通过 sender 逐条发送预设事件。
func (r *MockRunner) RunStream(_ context.Context, _ *agentv1.AgentRunRequest, sender agent.StreamSender) error {
	for _, event := range r.events {
		if err := sender.Send(event); err != nil {
			return err
		}
	}
	return nil
}

// SetEvents 配置需要回放的事件。
func (r *MockRunner) SetEvents(events []*agentv1.StreamEvent) {
	r.events = events
}
