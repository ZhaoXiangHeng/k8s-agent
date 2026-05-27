package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/sirupsen/logrus"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	mcptransport "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

var pkgLog = logrus.WithField("component", "agent-server/mcp")

// Client 管理从 mcp-server SSE 端点发现到的 MCP 工具。
type Client struct {
	rawClient *mcptransport.Client
	tools     []tool.BaseTool
}

// userIDToolWrapper 在工具执行前自动注入 user_id，避免 LLM 伪造调用身份。
type userIDToolWrapper struct {
	inner  tool.InvokableTool
	userID string
}

func (w *userIDToolWrapper) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return w.inner.Info(ctx)
}

func (w *userIDToolWrapper) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args map[string]any
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		args = make(map[string]any)
	}
	args["user_id"] = w.userID
	newJSON, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("marshal args with user_id: %w", err)
	}
	return w.inner.InvokableRun(ctx, string(newJSON), opts...)
}

// NewClient 连接 MCP SSE 服务并发现内置工具。
func NewClient(ctx context.Context, serverURL string) (*Client, error) {
	pkgLog.WithField("event", "mcp_connect_start").WithField("url", serverURL).Info("connecting to MCP server")
	sseClient, err := mcptransport.NewSSEMCPClient(serverURL)
	if err != nil {
		pkgLog.WithError(err).WithField("event", "mcp_client_create_failed").WithField("url", serverURL).Error("failed to create MCP SSE client")
		return nil, fmt.Errorf("mcp SSE client: %w", err)
	}
	if err := sseClient.Start(ctx); err != nil {
		pkgLog.WithError(err).WithField("event", "mcp_sse_start_failed").WithField("url", serverURL).Error("failed to start MCP SSE connection")
		return nil, fmt.Errorf("mcp SSE start: %w", err)
	}
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "agent-server",
		Version: "0.1.0",
	}
	if _, err := sseClient.Initialize(ctx, initRequest); err != nil {
		pkgLog.WithError(err).WithField("event", "mcp_initialize_failed").WithField("url", serverURL).Error("failed to initialize MCP connection")
		return nil, fmt.Errorf("mcp initialize: %w", err)
	}

	rawTools, err := einomcp.GetTools(ctx, &einomcp.Config{
		Cli: sseClient,
	})
	if err != nil {
		pkgLog.WithError(err).WithField("event", "mcp_tool_discovery_failed").WithField("url", serverURL).Error("failed to discover MCP tools")
		return nil, fmt.Errorf("mcp get tools: %w", err)
	}
	pkgLog.WithField("event", "mcp_tool_discovery_complete").WithField("url", serverURL).WithField("tool_count", len(rawTools)).Info("MCP tool discovery complete")
	return &Client{rawClient: sseClient, tools: rawTools}, nil
}

// ToolsForUser 返回带 user_id 自动注入的工具集合。
func (c *Client) ToolsForUser(userID string) []tool.BaseTool {
	wrapped := make([]tool.BaseTool, len(c.tools))
	for i, t := range c.tools {
		if invokable, ok := t.(tool.InvokableTool); ok {
			wrapped[i] = &userIDToolWrapper{inner: invokable, userID: userID}
		} else {
			wrapped[i] = t
		}
	}
	return wrapped
}

// Tools 返回未注入 user_id 的原始工具，仅用于内部诊断。
func (c *Client) Tools() []tool.BaseTool {
	return c.tools
}

// Close 关闭底层 MCP SSE 连接。
func (c *Client) Close() error {
	if c.rawClient != nil {
		return c.rawClient.Close()
	}
	return nil
}
