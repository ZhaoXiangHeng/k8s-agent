package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"k8s-ai-ops/mcp-server/internal/k8s"
)

func ListPodsTool() mcp.Tool {
	return mcp.NewTool("list_pods",
		mcp.WithDescription("List Kubernetes pods with optional namespace and label filters"),
		mcp.WithString("namespace", mcp.Description("Namespace to list pods from")),
		mcp.WithString("label_selector", mcp.Description("Kubernetes label selector to filter pods")),
	)
}

func HandleListPods(client *k8s.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		namespace := getStringArg(args, "namespace")
		labelSelector := getStringArg(args, "label_selector")
		pods, err := client.ListPods(ctx, namespace, labelSelector)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list pods: %v", err)), nil
		}
		data, _ := json.Marshal(pods)
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetPodTool() mcp.Tool {
	return mcp.NewTool("get_pod",
		mcp.WithDescription("Get detailed information about a specific Kubernetes pod"),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Pod namespace")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Pod name")),
	)
}

func HandleGetPod(client *k8s.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		namespace, _ := args["namespace"].(string)
		name, _ := args["name"].(string)
		pod, err := client.GetPod(ctx, namespace, name)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get pod: %v", err)), nil
		}
		data, _ := json.Marshal(pod)
		return mcp.NewToolResultText(string(data)), nil
	}
}

func GetPodLogsTool() mcp.Tool {
	return mcp.NewTool("get_pod_logs",
		mcp.WithDescription("Get logs from a Kubernetes pod container"),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Pod namespace")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Pod name")),
		mcp.WithString("container", mcp.Description("Container name (uses first container if not specified)")),
		mcp.WithNumber("tail_lines", mcp.Description("Number of lines from the end of the logs (default 50)")),
	)
}

func HandleGetPodLogs(client *k8s.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := getArgs(req)
		namespace, _ := args["namespace"].(string)
		name, _ := args["name"].(string)
		container, _ := args["container"].(string)
		tailLines := int64(50)
		if v, ok := args["tail_lines"].(float64); ok {
			tailLines = int64(v)
		}
		logs, err := client.GetPodLogs(ctx, namespace, name, container, tailLines)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get pod logs: %v", err)), nil
		}
		return mcp.NewToolResultText(logs), nil
	}
}
