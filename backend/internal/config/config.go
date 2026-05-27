package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPAddr           string
	KeycloakIssuer     string
	DatabaseURL        string
	RedisAddr          string
	AgentServerAddr    string
	StoreDriver        string
	CacheDriver        string
	K8SRBACSyncEnabled bool
	Kubeconfig         string
}

func Load() Config {
	return Config{
		HTTPAddr:           env("HTTP_ADDR", ":8080"),
		KeycloakIssuer:     env("KEYCLOAK_ISSUER", "http://keycloak:8080/realms/k8s-ai"),
		DatabaseURL:        env("DATABASE_URL", "postgres://k8s_ai:k8s_ai@postgresql:5432/k8s_ai?sslmode=disable"),
		RedisAddr:          env("REDIS_ADDR", "redis:6379"),
		AgentServerAddr:    env("AGENT_SERVER_ADDR", "agent-server:8082"),
		StoreDriver:        env("STORE_DRIVER", "memory"),
		CacheDriver:        env("CACHE_DRIVER", "none"),
		K8SRBACSyncEnabled: envBool("K8S_RBAC_SYNC_ENABLED", false),
		Kubeconfig:         env("KUBECONFIG", ""),
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes" || value == "on"
}
