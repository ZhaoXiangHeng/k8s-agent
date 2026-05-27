package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sirupsen/logrus"
	"k8s-ai-ops/mcp-server/internal/identity"
	"k8s-ai-ops/mcp-server/internal/k8s"
)

// ListDeploymentsTool 定义 list_deployments 工具，列出 Deployment 及其状态。
func ListDeploymentsTool() mcp.Tool {
	return mcp.NewTool("list_deployments",
		mcp.WithDescription("List Kubernetes deployments"),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User ID for Kubernetes authentication")),
		mcp.WithString("namespace", mcp.Description("Namespace to list deployments from")),
	)
}

// HandleListDeployments 是 list_deployments 工具的处理函数。
func HandleListDeployments(idClient *identity.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID, err := req.RequireString("user_id")
		if err != nil {
			return mcp.NewToolResultError("user_id is required"), nil
		}
		namespace := req.GetString("namespace", "")
		log.WithFields(logrus.Fields{
			"event":     "tool_call_start",
			"tool":      "list_deployments",
			"user_id":   userID,
			"namespace": namespace,
			"name":      "",
		}).Info("tool call started")

		sa, err := idClient.GetServiceAccount(ctx, userID)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "list_deployments",
				"user_id":   userID,
				"namespace": namespace,
				"name":      "",
				"reason":    "identity_lookup_failed",
			}).Error("tool call failed")
			return mcp.NewToolResultError(fmt.Sprintf("failed to get service account: %v", err)), nil
		}

		k8sClient, err := k8s.NewClientFromSA(sa.Token, sa.ApiServer, sa.Namespace, sa.CaCert)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "list_deployments",
				"user_id":   userID,
				"namespace": namespace,
				"name":      "",
				"reason":    "k8s_client_create_failed",
			}).Error("tool call failed")
			return mcp.NewToolResultError(fmt.Sprintf("failed to create k8s client: %v", err)), nil
		}

		deps, err := k8sClient.ListDeployments(ctx, namespace)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "list_deployments",
				"user_id":   userID,
				"namespace": namespace,
				"name":      "",
				"reason":    "k8s_list_deployments_failed",
			}).Error("tool call failed")
			return mcp.NewToolResultError(fmt.Sprintf("failed to list deployments: %v", err)), nil
		}
		data, err := json.Marshal(deps)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "list_deployments",
				"user_id":   userID,
				"namespace": namespace,
				"name":      "",
				"reason":    "marshal_response_failed",
			}).Error("tool call failed")
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
		}
		log.WithFields(logrus.Fields{
			"event":     "tool_call_success",
			"tool":      "list_deployments",
			"user_id":   userID,
			"namespace": namespace,
			"name":      "",
		}).Info("tool call succeeded")
		return mcp.NewToolResultText(string(data)), nil
	}
}

// RestartDeploymentTool 定义 restart_deployment 工具，通过触发滚动重启来重启 Deployment。
func RestartDeploymentTool() mcp.Tool {
	return mcp.NewTool("restart_deployment",
		mcp.WithDescription("Restart a Kubernetes deployment by triggering a rollout restart"),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User ID for Kubernetes authentication")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Deployment namespace")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Deployment name")),
	)
}

// HandleRestartDeployment 是 restart_deployment 工具的处理函数。
// 通过 Patch restartedAt 注解触发滚动重启，比直接 Update 更安全且与权限模型兼容。
func HandleRestartDeployment(idClient *identity.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID, err := req.RequireString("user_id")
		if err != nil {
			return mcp.NewToolResultError("user_id is required"), nil
		}

		sa, err := idClient.GetServiceAccount(ctx, userID)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "restart_deployment",
				"user_id":   userID,
				"namespace": "",
				"name":      "",
				"reason":    "identity_lookup_failed",
			}).Error("tool call failed")
			return mcp.NewToolResultError(fmt.Sprintf("failed to get service account: %v", err)), nil
		}

		k8sClient, err := k8s.NewClientFromSA(sa.Token, sa.ApiServer, sa.Namespace, sa.CaCert)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "restart_deployment",
				"user_id":   userID,
				"namespace": "",
				"name":      "",
				"reason":    "k8s_client_create_failed",
			}).Error("tool call failed")
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
		log.WithFields(logrus.Fields{
			"event":     "tool_call_start",
			"tool":      "restart_deployment",
			"user_id":   userID,
			"namespace": namespace,
			"name":      name,
		}).Info("tool call started")
		if err := k8sClient.RestartDeployment(ctx, namespace, name); err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "restart_deployment",
				"user_id":   userID,
				"namespace": namespace,
				"name":      name,
				"reason":    "k8s_restart_deployment_failed",
			}).Error("tool call failed")
			return mcp.NewToolResultError(fmt.Sprintf("failed to restart deployment: %v", err)), nil
		}
		log.WithFields(logrus.Fields{
			"event":     "tool_call_success",
			"tool":      "restart_deployment",
			"user_id":   userID,
			"namespace": namespace,
			"name":      name,
		}).Info("tool call succeeded")
		return mcp.NewToolResultText(fmt.Sprintf(`{"success":true,"message":"restarted deployment %s/%s"}`, namespace, name)), nil
	}
}
