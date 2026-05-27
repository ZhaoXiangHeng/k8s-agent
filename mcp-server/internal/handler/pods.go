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

// ListPodsTool 定义 list_pods 工具，支持 namespace 和 label 过滤。
func ListPodsTool() mcp.Tool {
	return mcp.NewTool("list_pods",
		mcp.WithDescription("List Kubernetes pods with optional namespace and label filters"),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User ID for Kubernetes authentication")),
		mcp.WithString("namespace", mcp.Description("Namespace to list pods from")),
		mcp.WithString("label_selector", mcp.Description("Kubernetes label selector to filter pods")),
	)
}

// HandleListPods 是 list_pods 工具的处理函数。
func HandleListPods(idClient *identity.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID, err := req.RequireString("user_id")
		if err != nil {
			return mcp.NewToolResultError("user_id is required"), nil
		}

		// namespace 和 label_selector 是可选参数
		namespace := req.GetString("namespace", "")
		labelSelector := req.GetString("label_selector", "")
		log.WithFields(logrus.Fields{
			"event":     "tool_call_start",
			"tool":      "list_pods",
			"user_id":   userID,
			"namespace": namespace,
			"name":      "",
		}).Info("tool call started")

		// 获取用户对应的 K8s ServiceAccount
		sa, err := idClient.GetServiceAccount(ctx, userID)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "list_pods",
				"user_id":   userID,
				"namespace": namespace,
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
				"tool":      "list_pods",
				"user_id":   userID,
				"namespace": namespace,
				"name":      "",
				"reason":    "k8s_client_create_failed",
			}).Error("tool call failed")
			return mcp.NewToolResultError(fmt.Sprintf("failed to create k8s client: %v", err)), nil
		}

		pods, err := k8sClient.ListPods(ctx, namespace, labelSelector)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "list_pods",
				"user_id":   userID,
				"namespace": namespace,
				"name":      "",
				"reason":    "k8s_list_pods_failed",
			}).Error("tool call failed")
			return mcp.NewToolResultError(fmt.Sprintf("failed to list pods: %v", err)), nil
		}
		data, err := json.Marshal(pods)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "list_pods",
				"user_id":   userID,
				"namespace": namespace,
				"name":      "",
				"reason":    "marshal_response_failed",
			}).Error("tool call failed")
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
		}
		log.WithFields(logrus.Fields{
			"event":     "tool_call_success",
			"tool":      "list_pods",
			"user_id":   userID,
			"namespace": namespace,
			"name":      "",
		}).Info("tool call succeeded")
		return mcp.NewToolResultText(string(data)), nil
	}
}

// GetPodTool 定义 get_pod 工具，获取单个 Pod 的详细信息。
func GetPodTool() mcp.Tool {
	return mcp.NewTool("get_pod",
		mcp.WithDescription("Get detailed information about a specific Kubernetes pod"),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User ID for Kubernetes authentication")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Pod namespace")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Pod name")),
	)
}

// HandleGetPod 是 get_pod 工具的处理函数。
func HandleGetPod(idClient *identity.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID, err := req.RequireString("user_id")
		if err != nil {
			return mcp.NewToolResultError("user_id is required"), nil
		}

		sa, err := idClient.GetServiceAccount(ctx, userID)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "get_pod",
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
				"tool":      "get_pod",
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
			"tool":      "get_pod",
			"user_id":   userID,
			"namespace": namespace,
			"name":      name,
		}).Info("tool call started")
		pod, err := k8sClient.GetPod(ctx, namespace, name)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "get_pod",
				"user_id":   userID,
				"namespace": namespace,
				"name":      name,
				"reason":    "k8s_get_pod_failed",
			}).Error("tool call failed")
			return mcp.NewToolResultError(fmt.Sprintf("failed to get pod: %v", err)), nil
		}
		data, err := json.Marshal(pod)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "get_pod",
				"user_id":   userID,
				"namespace": namespace,
				"name":      name,
				"reason":    "marshal_response_failed",
			}).Error("tool call failed")
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
		}
		log.WithFields(logrus.Fields{
			"event":     "tool_call_success",
			"tool":      "get_pod",
			"user_id":   userID,
			"namespace": namespace,
			"name":      name,
		}).Info("tool call succeeded")
		return mcp.NewToolResultText(string(data)), nil
	}
}

// GetPodLogsTool 定义 get_pod_logs 工具，获取 Pod 容器日志。
func GetPodLogsTool() mcp.Tool {
	return mcp.NewTool("get_pod_logs",
		mcp.WithDescription("Get logs from a Kubernetes pod container"),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User ID for Kubernetes authentication")),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Pod namespace")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Pod name")),
		mcp.WithString("container", mcp.Description("Container name (uses first container if not specified)")),
		mcp.WithNumber("tail_lines", mcp.Description("Number of lines from the end of the logs (default 50)")),
	)
}

// HandleGetPodLogs 是 get_pod_logs 工具的处理函数。
// 支持指定容器名和尾部行数，默认返回最后 50 行。
func HandleGetPodLogs(idClient *identity.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID, err := req.RequireString("user_id")
		if err != nil {
			return mcp.NewToolResultError("user_id is required"), nil
		}

		sa, err := idClient.GetServiceAccount(ctx, userID)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "get_pod_logs",
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
				"tool":      "get_pod_logs",
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
		container := req.GetString("container", "")
		tailLines := int64(req.GetFloat("tail_lines", 50))
		log.WithFields(logrus.Fields{
			"event":     "tool_call_start",
			"tool":      "get_pod_logs",
			"user_id":   userID,
			"namespace": namespace,
			"name":      name,
		}).Info("tool call started")
		logs, err := k8sClient.GetPodLogs(ctx, namespace, name, container, tailLines)
		if err != nil {
			log.WithError(err).WithFields(logrus.Fields{
				"event":     "tool_call_error",
				"tool":      "get_pod_logs",
				"user_id":   userID,
				"namespace": namespace,
				"name":      name,
				"reason":    "k8s_get_pod_logs_failed",
			}).Error("tool call failed")
			return mcp.NewToolResultError(fmt.Sprintf("failed to get pod logs: %v", err)), nil
		}
		log.WithFields(logrus.Fields{
			"event":     "tool_call_success",
			"tool":      "get_pod_logs",
			"user_id":   userID,
			"namespace": namespace,
			"name":      name,
		}).Info("tool call succeeded")
		return mcp.NewToolResultText(logs), nil
	}
}
