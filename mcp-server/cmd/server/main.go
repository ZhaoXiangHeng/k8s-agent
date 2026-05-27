package main

import (
	"context"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"k8s-ai-ops/mcp-server/internal/handler"
	"k8s-ai-ops/mcp-server/internal/identity"
)

func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8081"
	}

	identityAddr := os.Getenv("IDENTITY_SERVER_ADDR")
	if identityAddr == "" {
		identityAddr = "backend:8082"
	}

	idClient, err := identity.NewClient(context.Background(), identityAddr)
	if err != nil {
		log.Fatalf("level=ERROR component=mcp-server event=identity_connect_failed addr=%s error=%q", identityAddr, err)
	}

	s := server.NewMCPServer("k8s-mcp-server", "1.0.0")

	s.AddTool(handler.ListNamespacesTool(), handler.HandleListNamespaces(idClient))
	s.AddTool(handler.ListPodsTool(), handler.HandleListPods(idClient))
	s.AddTool(handler.GetPodTool(), handler.HandleGetPod(idClient))
	s.AddTool(handler.GetPodLogsTool(), handler.HandleGetPodLogs(idClient))
	s.AddTool(handler.GetPodEventsTool(), handler.HandleGetPodEvents(idClient))
	s.AddTool(handler.ListDeploymentsTool(), handler.HandleListDeployments(idClient))
	s.AddTool(handler.RestartDeploymentTool(), handler.HandleRestartDeployment(idClient))

	mcpServer := server.NewSSEServer(s)
	log.Printf("level=INFO component=mcp-server event=server_start addr=%s identity_addr=%s protocol=mcp+sse", addr, identityAddr)
	if err := mcpServer.Start(addr); err != nil {
		log.Fatalf("level=ERROR component=mcp-server event=server_exit error=%q", err)
	}
}
