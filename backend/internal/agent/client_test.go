package agent

import (
	"context"
	"io"
	"testing"

	"google.golang.org/grpc"
	agentv1 "k8s-ai-ops/proto/agent/v1"
)

type mockRunStreamClient struct {
	grpc.ClientStream
	events []*agentv1.StreamEvent
	pos    int
}

func (m *mockRunStreamClient) Recv() (*agentv1.StreamEvent, error) {
	if m.pos >= len(m.events) {
		return nil, io.EOF
	}
	event := m.events[m.pos]
	m.pos++
	return event, nil
}

type mockAgentServiceClient struct {
	agentv1.AgentServiceClient
	stream *mockRunStreamClient
	err    error
}

func (m *mockAgentServiceClient) RunStream(ctx context.Context, in *agentv1.AgentRunRequest, opts ...grpc.CallOption) (agentv1.AgentService_RunStreamClient, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.stream, nil
}

func TestClientRunStreamReceivesEvents(t *testing.T) {
	mockStream := &mockRunStreamClient{events: []*agentv1.StreamEvent{
		{EventId: "1", Event: &agentv1.StreamEvent_Thinking{Thinking: &agentv1.ThinkingEvent{Content: "hello"}}},
		{EventId: "2", Event: &agentv1.StreamEvent_Complete{Complete: &agentv1.CompleteEvent{Summary: "done"}}},
	}}
	mockClient := &mockAgentServiceClient{stream: mockStream}

	client := NewGRPCClient(mockClient)
	reader, err := client.RunStream(context.Background(), &agentv1.AgentRunRequest{Message: "check dev"})
	if err != nil {
		t.Fatal(err)
	}

	var count int
	for {
		_, err := reader.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count != 2 {
		t.Fatalf("expected 2 events, got %d", count)
	}
}

func TestClientRunStreamPropagatesError(t *testing.T) {
	mockClient := &mockAgentServiceClient{err: io.ErrUnexpectedEOF}

	client := NewGRPCClient(mockClient)
	_, err := client.RunStream(context.Background(), &agentv1.AgentRunRequest{Message: "check dev"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
