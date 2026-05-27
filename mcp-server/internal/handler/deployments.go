package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"k8s-ai-ops/mcp-server/internal/k8s"
)

func ListDeploymentsTool() mcp.Tool {
	return mcp.NewTool("list_deployments",
		mcp.WithDescription("List Kubernetes deployments"),
		mcp.WithString("namespace", mcp.Description("Namespace to list deployments from")),
	)
}

func HandleListDeployments(client *k8s.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		namespace := getStringArg(getArgs(req), "namespace")
		deps, err := client.ListDeployments(ctx, namespace)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list deployments: %v", err)), nil
		}
		data, _ := json.Marshal(deps)
		return mcp.NewToolResultText(string(data)), nil
	}
}

func RestartDeploymentTool() mcp.Tool {
	return mcp.NewTool("restart_deployment",
		mcp.WithDescription("Restart a Kubernetes deployment by triggering a rollout restart"),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Deployment namespace")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Deployment name")),
	)
}

func HandleRestartDeployment(client *k8s.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		namespace, _ := args["namespace"].(string)
		name, _ := args["name"].(string)
		if err := client.RestartDeployment(ctx, namespace, name); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to restart deployment: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf(`{"success":true,"message":"restarted deployment %s/%s"}`, namespace, name)), nil
	}
}

func getStringArg(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}

func getArgs(req mcp.CallToolRequest) map[string]any {
	args, ok := req.Params.Arguments.(map[string]any)
	if !ok {
		return make(map[string]any)
	}
	return args
}
