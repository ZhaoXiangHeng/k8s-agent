package agent

import (
	"context"
	"io"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	agentv1 "k8s-ai-ops/proto/agent/v1"
)

type StreamEvent = agentv1.StreamEvent

type Client interface {
	RunStream(context.Context, *agentv1.AgentRunRequest) (StreamReader, error)
}

type StreamReader interface {
	Recv() (*agentv1.StreamEvent, error)
}

type GRPCClient struct {
	client agentv1.AgentServiceClient
}

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

func NewGRPCClient(client agentv1.AgentServiceClient) *GRPCClient {
	return &GRPCClient{client: client}
}

func Dial(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return grpc.DialContext(dialCtx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
}

func (c *GRPCClient) RunStream(ctx context.Context, request *agentv1.AgentRunRequest) (StreamReader, error) {
	stream, err := c.client.RunStream(ctx, request)
	if err != nil {
		return nil, err
	}
	return &grpcStreamReader{stream: stream}, nil
}
