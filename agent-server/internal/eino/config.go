package eino

import "os"

// Config holds runtime configuration for the agent server.
type Config struct {
	MCPServerURL string
}

// LoadConfig reads configuration from environment variables with sensible defaults.
func LoadConfig() Config {
	url := os.Getenv("MCP_SERVER_URL")
	if url == "" {
		url = "http://localhost:8081/sse"
	}
	return Config{MCPServerURL: url}
}
