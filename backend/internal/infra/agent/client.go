// Package agent 封装与 Agent Server 的 gRPC 通信。
// Backend 通过此包调用 AgentService.RunStream（服务端流式 RPC），
// 将用户上下文传递给 Agent Server 并接收流式 Agent 事件，
// 再通过 SSE 转发给前端。
package agent

import (
	"context"
	"io"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	agentv1 "k8s-ai-ops/proto/agent/v1"
)

// pkgLog 是 Agent 客户端的包级日志记录器。
var pkgLog = logrus.WithField("component", "backend-api/agent-client")

// StreamEvent 是 Agent Server 流式返回的事件的类型别名。
type StreamEvent = agentv1.StreamEvent

// Client 定义 Agent Server 的 gRPC 客户端接口。
// RunStream 发送 AgentRunRequest 并返回事件流读取器。
type Client interface {
	RunStream(context.Context, *agentv1.AgentRunRequest) (StreamReader, error)
}

// StreamReader 定义 Agent 事件流的读取接口。
type StreamReader interface {
	Recv() (*agentv1.StreamEvent, error)
}

// GRPCClient 是 Client 接口的 gRPC 实现。
type GRPCClient struct {
	client agentv1.AgentServiceClient
}

// grpcStreamReader 包装 gRPC 服务端流，实现 StreamReader 接口。
type grpcStreamReader struct {
	stream agentv1.AgentService_RunStreamClient
}

func (r *grpcStreamReader) Recv() (*agentv1.StreamEvent, error) {
	event, err := r.stream.Recv()
	if err == io.EOF {
		return nil, err
	}
	return event, err
}

// NewGRPCClient 创建一个包装了 AgentServiceClient 的 GRPCClient。
func NewGRPCClient(client agentv1.AgentServiceClient) *GRPCClient {
	return &GRPCClient{client: client}
}

// Dial 创建到 Agent Server 的 gRPC 连接。
// 使用 insecure 传输（集群内部通信，依赖 CNI 网络策略保障安全）。
// 连接超时为 10 秒。
func Dial(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	pkgLog.WithFields(logrus.Fields{
		"event": "agent_dial_start",
		"addr":  addr,
	}).Info("connecting to agent server")
	conn, err := grpc.DialContext(dialCtx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		pkgLog.WithError(err).WithFields(logrus.Fields{
			"event": "agent_dial_failed",
			"addr":  addr,
		}).Error("failed to connect to agent server")
		return nil, err
	}
	pkgLog.WithFields(logrus.Fields{
		"event": "agent_dial_success",
		"addr":  addr,
	}).Info("connected to agent server")
	return conn, nil
}

// RunStream 向 Agent Server 发送 RunStream 请求，返回事件流读取器。
// 调用方通过 StreamReader.Recv() 循环接收 StreamEvent 直到 io.EOF。
func (c *GRPCClient) RunStream(ctx context.Context, request *agentv1.AgentRunRequest) (StreamReader, error) {
	stream, err := c.client.RunStream(ctx, request)
	if err != nil {
		return nil, err
	}
	return &grpcStreamReader{stream: stream}, nil
}
