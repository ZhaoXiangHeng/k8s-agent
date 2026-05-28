// Package config 从环境变量加载 Backend 的运行时配置。
// 所有配置项都有合理的本地开发默认值，生产环境通过 K8s ConfigMap/Secret 注入。
package config

import (
	"os"
	"strings"
)

// Config 保存 Backend 的全部运行时配置。
type Config struct {
	HTTPAddr           string // HTTP API 监听地址（默认 :8080）
	KeycloakIssuer     string // Keycloak OIDC Issuer URL
	AuthMode           string // 认证模式："dev"（默认）、"jwt" 或 "none"
	DatabaseURL        string // PostgreSQL 连接串
	RedisAddr          string // Redis 地址（仅 CACHE_DRIVER=redis 时使用）
	AgentServerAddr    string // Agent Server gRPC 地址
	CacheDriver        string // 缓存驱动："none"（默认）或 "redis"
	K8SRBACSyncEnabled bool   // 权限变更时是否同步 K8s RBAC 资源
	Kubeconfig         string // Kubeconfig 文件路径（空则使用 InClusterConfig）
	AdminSAName        string // 管理员共享 SA 名称（默认 k8s-ai-admin）
	AdminSANamespace   string // 管理员共享 SA 所在 namespace
}

// Load 从环境变量加载 Config，未设置的变量使用默认值。
func Load() Config {
	return Config{
		HTTPAddr:           env("HTTP_ADDR", ":8080"),
		KeycloakIssuer:     env("KEYCLOAK_ISSUER", ""),
		AuthMode:           env("AUTH_MODE", "dev"),
		DatabaseURL:        env("DATABASE_URL", "postgres://k8s_ai:k8s_ai@postgresql:5432/k8s_ai?sslmode=disable"),
		RedisAddr:          env("REDIS_ADDR", "redis:6379"),
		AgentServerAddr:    env("AGENT_SERVER_ADDR", "agent-server:8082"),
		CacheDriver:        env("CACHE_DRIVER", "none"),
		K8SRBACSyncEnabled: envBool("K8S_RBAC_SYNC_ENABLED", false),
		Kubeconfig:         env("KUBECONFIG", ""),
		AdminSAName:        env("ADMIN_SA_NAME", "k8s-ai-admin"),
		AdminSANamespace:   env("ADMIN_SA_NAMESPACE", "k8s-ai-system"),
	}
}

// env 读取环境变量，未设置时返回默认值。
func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

// envBool 读取布尔型环境变量，支持 "1"、"true"、"yes"、"on"（不区分大小写）。
func envBool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes" || value == "on"
}
