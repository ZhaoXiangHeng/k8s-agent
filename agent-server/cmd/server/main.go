package main

import (
	"context"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	einorunner "k8s-ai-ops/agent-server/internal/eino"
	"k8s-ai-ops/agent-server/internal/server"
	agentv1 "k8s-ai-ops/proto/agent/v1"
)

func main() {
	addr := os.Getenv("GRPC_ADDR")
	if addr == "" {
		addr = ":8082"
	}
	cfg := einorunner.LoadConfig()

	ctx := context.Background()
	runner, err := einorunner.NewRunner(ctx, cfg.MCPServerURL)
	if err != nil {
		log.Fatalf("level=ERROR component=agent-server event=mcp_connect_failed url=%s error=%q", cfg.MCPServerURL, err)
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("level=ERROR component=agent-server event=listen_failed addr=%s error=%q", addr, err)
	}

	grpcServer := grpc.NewServer()
	agentv1.RegisterAgentServiceServer(grpcServer, server.NewAgentService(runner))
	log.Printf("level=INFO component=agent-server event=server_start addr=%s mcp_url=%s protocol=grpc", addr, cfg.MCPServerURL)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("level=ERROR component=agent-server event=server_exit error=%q", err)
	}
}
