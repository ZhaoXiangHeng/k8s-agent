// Package main 是 mcp-server 的入口，启动一个 MCP over SSE 服务，
// 将 Kubernetes 操作封装为 MCP 工具供 AI Agent 调用。
// 通过 Identity Service 获取用户对应的 ServiceAccount 凭证，
// 实现 per-user 的 K8s 访问控制。
package main

import (
	"context"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"

	"k8s-ai-ops/mcp-server/internal/handler"
	"k8s-ai-ops/mcp-server/internal/identity"
)

var log = logrus.WithField("component", "mcp-server")

// init 配置结构化 JSON 日志，包含调用者信息，便于排查问题。
func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	})
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetReportCaller(true)
}

func main() {
	// 从环境变量读取监听地址，默认 :8081
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8081"
	}

	// Identity Service 地址，用于查询用户的 K8s ServiceAccount
	identityAddr := os.Getenv("IDENTITY_SERVER_ADDR")
	if identityAddr == "" {
		identityAddr = "backend:8082"
	}

	// 启动时即连接 Identity Service，失败则直接退出
	idClient, err := identity.NewClient(context.Background(), identityAddr)
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"event": "identity_connect_failed",
			"addr":  identityAddr,
		}).Fatal("failed to connect to identity service")
	}

	// 创建 MCP Server 并注册所有工具
	s := server.NewMCPServer("k8s-mcp-server", "1.0.0")

	// 注册工具：每个工具由 Tool 定义 + Handler 函数组成
	s.AddTool(handler.ListNamespacesTool(), handler.HandleListNamespaces(idClient))
	s.AddTool(handler.ListPodsTool(), handler.HandleListPods(idClient))
	s.AddTool(handler.GetPodTool(), handler.HandleGetPod(idClient))
	s.AddTool(handler.GetPodLogsTool(), handler.HandleGetPodLogs(idClient))
	s.AddTool(handler.ListEventsTool(), handler.HandleGetPodEvents(idClient))
	s.AddTool(handler.GetPodEventsTool(), handler.HandleGetPodEvents(idClient))
	s.AddTool(handler.ListDeploymentsTool(), handler.HandleListDeployments(idClient))
	s.AddTool(handler.RestartDeploymentTool(), handler.HandleRestartDeployment(idClient))

	// 通过 SSE 协议暴露 MCP 服务
	mcpServer := server.NewSSEServer(s)
	log.WithFields(logrus.Fields{
		"event":         "server_start",
		"addr":          addr,
		"identity_addr": identityAddr,
		"protocol":      "mcp+sse",
	}).Info("server starting")
	if err := mcpServer.Start(addr); err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"event": "server_exit",
		}).Fatal("server exited unexpectedly")
	}
}
