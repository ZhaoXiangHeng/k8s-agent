package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"k8s-ai-ops/mcp-server/internal/k8s"
)

func ListNamespacesTool() mcp.Tool {
	return mcp.NewTool("list_namespaces",
		mcp.WithDescription("List all Kubernetes namespaces"),
	)
}

func HandleListNamespaces(client *k8s.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		names, err := client.ListNamespaces(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list namespaces: %v", err)), nil
		}
		data, _ := json.Marshal(names)
		return mcp.NewToolResultText(string(data)), nil
	}
}
