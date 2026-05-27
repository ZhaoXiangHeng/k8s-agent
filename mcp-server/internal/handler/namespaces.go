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

// ListNamespacesTool 定义 list_namespaces 工具，列出用户可访问的所有 namespace。
func ListNamespacesTool() mcp.Tool {
	return mcp.NewTool("list_namespaces",
		mcp.WithDescription("List all Kubernetes namespaces"),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User ID for Kubernetes authentication")),
	)
}

// HandleListNamespaces 是 list_namespaces 工具的处理函数。
func HandleListNamespaces(idClient *identity.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID, err := req.RequireString("user_id")
		if err != nil {
			return mcp.NewToolResultError("user_id is required"), nil
		}
		log.WithFields(logrus.Fields{
			"event":     "tool_call_start",
			"tool":      "list_namespaces",
			"user_id":   userID,
			"namespace": "",
			"name":      "",
		}).Info("tool call started")

		// 获取用户对应的 K8s ServiceAccount
		sa, err := idClient.GetServiceAccount(ctx, userID)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "list_namespaces",
				"user_id":   userID,
				"namespace": "",
				"name":      "",
				"reason":    "identity_lookup_failed",
			}).Error("tool call failed")
			return mcp.NewToolResultError(fmt.Sprintf("failed to get service account: %v", err)), nil
		}

		// 创建 per-user K8s 客户端
		k8sClient, err := k8s.NewClientFromSA(sa.Token, sa.ApiServer, sa.Namespace, sa.CaCert)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "list_namespaces",
				"user_id":   userID,
				"namespace": "",
				"name":      "",
				"reason":    "k8s_client_create_failed",
			}).Error("tool call failed")
			return mcp.NewToolResultError(fmt.Sprintf("failed to create k8s client: %v", err)), nil
		}

		names, err := k8sClient.ListNamespaces(ctx)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "list_namespaces",
				"user_id":   userID,
				"namespace": "",
				"name":      "",
				"reason":    "k8s_list_namespaces_failed",
			}).Error("tool call failed")
			return mcp.NewToolResultError(fmt.Sprintf("failed to list namespaces: %v", err)), nil
		}
		data, err := json.Marshal(names)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "list_namespaces",
				"user_id":   userID,
				"namespace": "",
				"name":      "",
				"reason":    "marshal_response_failed",
			}).Error("tool call failed")
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
		}
		log.WithFields(logrus.Fields{
			"event":     "tool_call_success",
			"tool":      "list_namespaces",
			"user_id":   userID,
			"namespace": "",
			"name":      "",
		}).Info("tool call succeeded")
		return mcp.NewToolResultText(string(data)), nil
	}
}
