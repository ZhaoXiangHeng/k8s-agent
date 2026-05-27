package server

import (
	"context"

	agentv1 "k8s-ai-ops/proto/agent/v1"
	agent "k8s-ai-ops/agent-server/internal/eino"
)

// Runner is the interface the gRPC service depends on to execute agent runs.
type Runner interface {
	RunStream(context.Context, *agentv1.AgentRunRequest, agent.StreamSender) error
}

// AgentService implements the AgentServiceServer gRPC service.
type AgentService struct {
	agentv1.UnimplementedAgentServiceServer
	runner Runner
}

// NewAgentService creates a new AgentService backed by the given Runner.
func NewAgentService(runner Runner) *AgentService {
	return &AgentService{runner: runner}
}

// RunStream handles server-streaming agent run requests.
func (s *AgentService) RunStream(request *agentv1.AgentRunRequest, stream agentv1.AgentService_RunStreamServer) error {
	return s.runner.RunStream(stream.Context(), request, stream)
}
