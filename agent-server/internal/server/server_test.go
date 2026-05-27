package server

import (
	"context"
	"io"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	agentv1 "k8s-ai-ops/proto/agent/v1"
)

const testBufSize = 1024 * 1024

func TestAgentServiceRunStreamSendsEvents(t *testing.T) {
	runner := NewMockRunner()
	runner.SetEvents([]*agentv1.StreamEvent{
		{EventId: "1", Event: &agentv1.StreamEvent_Thinking{Thinking: &agentv1.ThinkingEvent{Content: "thinking..."}}},
		{EventId: "2", Event: &agentv1.StreamEvent_Complete{Complete: &agentv1.CompleteEvent{Summary: "done"}}},
	})

	service := NewAgentService(runner)
	listener := bufconn.Listen(testBufSize)
	grpcServer := grpc.NewServer()
	agentv1.RegisterAgentServiceServer(grpcServer, service)
	go grpcServer.Serve(listener)
	defer grpcServer.Stop()

	conn, _ := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	defer conn.Close()

	client := agentv1.NewAgentServiceClient(conn)
	stream, err := client.RunStream(context.Background(), &agentv1.AgentRunRequest{
		Message: "check dev",
		RuntimeContext: &agentv1.RuntimeContext{CurrentUser: "operator-a"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var events []*agentv1.StreamEvent
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, event)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}
