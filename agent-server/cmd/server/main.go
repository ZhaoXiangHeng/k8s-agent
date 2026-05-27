package main

import (
	"context"
	"net"
	"os"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	einorunner "k8s-ai-ops/agent-server/internal/eino"
	"k8s-ai-ops/agent-server/internal/server"
	agentv1 "k8s-ai-ops/proto/agent/v1"
)

var log = logrus.WithField("component", "agent-server")

// init 配置结构化 JSON 日志，包含调用者信息，便于排查问题。
func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	})
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetReportCaller(true)
}

func main() {
	addr := os.Getenv("GRPC_ADDR")
	if addr == "" {
		addr = ":8082"
	}
	cfg := einorunner.LoadConfig()

	ctx := context.Background()
	runner, err := einorunner.NewRunner(ctx, cfg.MCPServerURL, cfg.SkillsDir)
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"event": "mcp_connect_failed",
			"url":   cfg.MCPServerURL,
		}).Fatal("failed to connect to MCP server")
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"event": "listen_failed",
			"addr":  addr,
		}).Fatal("failed to listen")
	}

	grpcServer := grpc.NewServer()
	agentv1.RegisterAgentServiceServer(grpcServer, server.NewAgentService(runner))
	log.WithFields(logrus.Fields{
		"event":    "server_start",
		"addr":     addr,
		"mcp_url":  cfg.MCPServerURL,
		"protocol": "grpc",
	}).Info("server starting")
	if err := grpcServer.Serve(listener); err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"event": "server_exit",
		}).Fatal("server exited unexpectedly")
	}
}
