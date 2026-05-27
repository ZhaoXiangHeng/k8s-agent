package eino

import "os"

// Config 保存 Agent Server 的运行时配置。
type Config struct {
	MCPServerURL string
	SkillsDir    string
}

// LoadConfig 从环境变量读取配置，并提供本地开发默认值。
func LoadConfig() Config {
	url := os.Getenv("MCP_SERVER_URL")
	if url == "" {
		url = "http://localhost:8081/sse"
	}
	skillsDir := os.Getenv("SKILLS_DIR")
	if skillsDir == "" {
		skillsDir = "./skills"
	}
	return Config{MCPServerURL: url, SkillsDir: skillsDir}
}
