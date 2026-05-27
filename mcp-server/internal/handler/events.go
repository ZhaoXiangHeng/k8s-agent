package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"k8s-ai-ops/mcp-server/internal/k8s"
)

func GetPodEventsTool() mcp.Tool {
	return mcp.NewTool("get_pod_events",
		mcp.WithDescription("List Kubernetes events related to a specific pod"),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Pod namespace")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Pod name")),
	)
}

func HandleGetPodEvents(client *k8s.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		namespace, _ := args["namespace"].(string)
		name, _ := args["name"].(string)
		events, err := client.ListEvents(ctx, namespace, name)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list events: %v", err)), nil
		}
		data, _ := json.Marshal(events)
		return mcp.NewToolResultText(string(data)), nil
	}
}
