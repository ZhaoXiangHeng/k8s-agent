package main

import (
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"k8s-ai-ops/mcp-server/internal/handler"
	"k8s-ai-ops/mcp-server/internal/k8s"
)

func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8081"
	}
	kubeconfig := os.Getenv("KUBECONFIG")

	k8sClient, err := k8s.NewClient(kubeconfig)
	if err != nil {
		log.Fatalf("level=ERROR component=mcp-server event=k8s_client_create_failed error=%q", err)
	}

	s := server.NewMCPServer("k8s-mcp-server", "1.0.0")

	s.AddTool(handler.ListNamespacesTool(), handler.HandleListNamespaces(k8sClient))
	s.AddTool(handler.ListPodsTool(), handler.HandleListPods(k8sClient))
	s.AddTool(handler.GetPodTool(), handler.HandleGetPod(k8sClient))
	s.AddTool(handler.GetPodLogsTool(), handler.HandleGetPodLogs(k8sClient))
	s.AddTool(handler.GetPodEventsTool(), handler.HandleGetPodEvents(k8sClient))
	s.AddTool(handler.ListDeploymentsTool(), handler.HandleListDeployments(k8sClient))
	s.AddTool(handler.RestartDeploymentTool(), handler.HandleRestartDeployment(k8sClient))

	mcpServer := server.NewSSEServer(s)
	log.Printf("level=INFO component=mcp-server event=server_start addr=%s protocol=mcp+sse", addr)
	if err := mcpServer.Start(addr); err != nil {
		log.Fatalf("level=ERROR component=mcp-server event=server_exit error=%q", err)
	}
}
