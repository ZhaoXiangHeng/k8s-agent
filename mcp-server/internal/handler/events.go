// Package handler 实现了 MCP 工具的定义和处理函数。
// 每个工具遵循统一模式：
//  1. 从请求中提取 user_id 参数
//  2. 通过 Identity Service 获取用户对应的 K8s ServiceAccount
//  3. 创建 per-user 的 K8s 客户端
//  4. 执行 K8s 操作并返回结果
//
// 所有操作都在用户 ServiceAccount 的权限范围内执行，实现租户隔离。
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

// ListEventsTool 定义 list_events 工具的元数据和参数。
func ListEventsTool() mcp.Tool {
	return mcp.NewTool("list_events",
		mcp.WithDescription("List Kubernetes events related to a specific pod"),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User ID for Kubernetes authentication")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Pod namespace")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Pod name")),
	)
}

// GetPodEventsTool 定义 get_pod_events 工具，作为 list_events 的向后兼容别名。
func GetPodEventsTool() mcp.Tool {
	return mcp.NewTool("get_pod_events",
		mcp.WithDescription("Backward-compatible alias for list_events"),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User ID for Kubernetes authentication")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Pod namespace")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Pod name")),
	)
}

// HandleGetPodEvents 是 list_events / get_pod_events 两个工具的共享处理函数。
// 流程：身份查询 → 创建 K8s 客户端 → 获取 Events → JSON 序列化返回。
func HandleGetPodEvents(idClient *identity.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID, err := req.RequireString("user_id")
		if err != nil {
			return mcp.NewToolResultError("user_id is required"), nil
		}

		// 通过 Identity Service 获取用户的 K8s ServiceAccount 凭证
		sa, err := idClient.GetServiceAccount(ctx, userID)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "list_events",
				"user_id":   userID,
				"namespace": "",
				"name":      "",
				"reason":    "identity_lookup_failed",
			}).Error("tool call failed")
			return mcp.NewToolResultError(fmt.Sprintf("failed to get service account: %v", err)), nil
		}

		// 使用 SA 凭证创建 per-user 的 K8s 客户端
		k8sClient, err := k8s.NewClientFromSA(sa.Token, sa.ApiServer, sa.Namespace, sa.CaCert)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "list_events",
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
			"tool":      "list_events",
			"user_id":   userID,
			"namespace": namespace,
			"name":      name,
		}).Info("tool call started")
		events, err := k8sClient.ListEvents(ctx, namespace, name)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "list_events",
				"user_id":   userID,
				"namespace": namespace,
				"name":      name,
				"reason":    "k8s_list_events_failed",
			}).Error("tool call failed")
			return mcp.NewToolResultError(fmt.Sprintf("failed to list events: %v", err)), nil
		}
		data, err := json.Marshal(events)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "list_events",
				"user_id":   userID,
				"namespace": namespace,
				"name":      name,
				"reason":    "marshal_response_failed",
			}).Error("tool call failed")
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
		}
		log.WithFields(logrus.Fields{
			"event":     "tool_call_success",
			"tool":      "list_events",
			"user_id":   userID,
			"namespace": namespace,
			"name":      name,
		}).Info("tool call succeeded")
		return mcp.NewToolResultText(string(data)), nil
	}
}
