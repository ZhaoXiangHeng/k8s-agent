package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"k8s-ai-ops/mcp-server/internal/identity"
	"k8s-ai-ops/mcp-server/internal/k8s"
)

func ListDeploymentsTool() mcp.Tool {
	return mcp.NewTool("list_deployments",
		mcp.WithDescription("List Kubernetes deployments"),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User ID for Kubernetes authentication")),
		mcp.WithString("namespace", mcp.Description("Namespace to list deployments from")),
	)
}

func HandleListDeployments(idClient *identity.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		namespace := req.GetString("namespace", "")
		deps, err := k8sClient.ListDeployments(ctx, namespace)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list deployments: %v", err)), nil
		}
		data, err := json.Marshal(deps)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func RestartDeploymentTool() mcp.Tool {
	return mcp.NewTool("restart_deployment",
		mcp.WithDescription("Restart a Kubernetes deployment by triggering a rollout restart"),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User ID for Kubernetes authentication")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Deployment namespace")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Deployment name")),
	)
}

func HandleRestartDeployment(idClient *identity.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		if err := k8sClient.RestartDeployment(ctx, namespace, name); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to restart deployment: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf(`{"success":true,"message":"restarted deployment %s/%s"}`, namespace, name)), nil
	}
}
