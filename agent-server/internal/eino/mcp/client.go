package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	mcptransport "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// Client manages MCP tools from an mcp-server SSE endpoint.
type Client struct {
	rawClient *mcptransport.Client
	tools     []tool.BaseTool
}

// userIDToolWrapper auto-injects user_id into tool call arguments before execution.
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

// NewClient connects to the MCP server via SSE and discovers tools.
func NewClient(ctx context.Context, serverURL string) (*Client, error) {
	sseClient, err := mcptransport.NewSSEMCPClient(serverURL)
	if err != nil {
		return nil, fmt.Errorf("mcp SSE client: %w", err)
	}
	if err := sseClient.Start(ctx); err != nil {
		return nil, fmt.Errorf("mcp SSE start: %w", err)
	}
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "agent-server",
		Version: "0.1.0",
	}
	if _, err := sseClient.Initialize(ctx, initRequest); err != nil {
		return nil, fmt.Errorf("mcp initialize: %w", err)
	}

	rawTools, err := einomcp.GetTools(ctx, &einomcp.Config{
		Cli: sseClient,
	})
	if err != nil {
		return nil, fmt.Errorf("mcp get tools: %w", err)
	}
	return &Client{rawClient: sseClient, tools: rawTools}, nil
}

// ToolsForUser returns tools with user_id auto-injection for per-user K8s clients.
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

// Tools returns the raw tools without user_id injection.
func (c *Client) Tools() []tool.BaseTool {
	return c.tools
}

// Close shuts down the underlying MCP SSE connection.
func (c *Client) Close() error {
	if c.rawClient != nil {
		return c.rawClient.Close()
	}
	return nil
}
