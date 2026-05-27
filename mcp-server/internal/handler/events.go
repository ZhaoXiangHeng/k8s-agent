package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"k8s-ai-ops/mcp-server/internal/identity"
	"k8s-ai-ops/mcp-server/internal/k8s"
)

func GetPodEventsTool() mcp.Tool {
	return mcp.NewTool("get_pod_events",
		mcp.WithDescription("List Kubernetes events related to a specific pod"),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User ID for Kubernetes authentication")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Pod namespace")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Pod name")),
	)
}

func HandleGetPodEvents(idClient *identity.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID, err := req.RequireString("user_id")
		if err != nil {
			return mcp.NewToolResultError("user_id is required"), nil
		}

		sa, err := idClient.GetServiceAccount(ctx, userID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get service account: %v", err)), nil
		}

		k8sClient, err := k8s.NewClientFromSA(sa.Token, sa.ApiServer, sa.Namespace, sa.CaCert)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create k8s client: %v", err)), nil
		}

		namespace, err := req.RequireString("namespace")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		name, err := req.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		events, err := k8sClient.ListEvents(ctx, namespace, name)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list events: %v", err)), nil
		}
		data, err := json.Marshal(events)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
