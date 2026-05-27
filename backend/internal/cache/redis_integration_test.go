package cache

import (
	"os"
	"testing"
)

func TestRedisClientIntegration(t *testing.T) {
	addr := os.Getenv("K8S_AI_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("K8S_AI_TEST_REDIS_ADDR is not set")
	}

	client := NewRedisClient(addr)
	if err := client.Ping(); err != nil {
		t.Fatalf("redis ping: %v", err)
	}
	if err := client.Set("k8s-ai:test", "ok"); err != nil {
		t.Fatalf("redis set: %v", err)
	}
	value, err := client.Get("k8s-ai:test")
	if err != nil {
		t.Fatalf("redis get: %v", err)
	}
	if value != "ok" {
		t.Fatalf("expected ok, got %s", value)
	}
}
