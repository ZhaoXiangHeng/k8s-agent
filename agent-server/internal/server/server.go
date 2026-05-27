package server

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	agent "k8s-ai-ops/agent-server/internal/eino"
	agentv1 "k8s-ai-ops/proto/agent/v1"
)

var pkgLog = logrus.WithField("component", "agent-server/server")

// Runner 是 gRPC 服务执行 Agent Run 时依赖的最小接口。
type Runner interface {
	RunStream(context.Context, *agentv1.AgentRunRequest, agent.StreamSender) error
}

// AgentService 实现 proto 生成的 gRPC 服务。
type AgentService struct {
	agentv1.UnimplementedAgentServiceServer
	runner Runner
}

// NewAgentService 创建绑定指定 Runner 的 AgentService。
func NewAgentService(runner Runner) *AgentService {
	return &AgentService{runner: runner}
}

// RunStream 处理服务端流式 Agent 请求。
func (s *AgentService) RunStream(request *agentv1.AgentRunRequest, stream agentv1.AgentService_RunStreamServer) error {
	started := time.Now()
	pkgLog.WithField("event", "run_stream_start").WithField("request_id", request.GetRequestId()).WithField("session_id", request.GetSessionId()).WithField("message_id", request.GetMessageId()).WithField("user_id", request.GetUser().GetId()).Info("stream run started")
	if err := s.runner.RunStream(stream.Context(), request, stream); err != nil {
		pkgLog.WithError(err).WithField("event", "run_stream_failed").WithField("request_id", request.GetRequestId()).WithField("duration_ms", time.Since(started).Milliseconds()).Error("stream run failed")
		return err
	}
	pkgLog.WithField("event", "run_stream_complete").WithField("request_id", request.GetRequestId()).WithField("duration_ms", time.Since(started).Milliseconds()).Info("stream run complete")
	return nil
}
