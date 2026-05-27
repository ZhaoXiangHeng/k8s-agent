package config

import "testing"

func TestLoadDefaultsK8SRBACSyncDisabled(t *testing.T) {
	t.Setenv("K8S_RBAC_SYNC_ENABLED", "")
	t.Setenv("KUBECONFIG", "")
	t.Setenv("AGENT_SERVER_ADDR", "")

	cfg := Load()

	if cfg.K8SRBACSyncEnabled {
		t.Fatal("expected k8s rbac sync disabled by default")
	}
	if cfg.Kubeconfig != "" {
		t.Fatalf("expected empty kubeconfig by default, got %q", cfg.Kubeconfig)
	}
	if cfg.AgentServerAddr != "agent-server:8082" {
		t.Fatalf("expected default agent server addr, got %q", cfg.AgentServerAddr)
	}
}

func TestLoadEnablesK8SRBACSyncFromEnv(t *testing.T) {
	t.Setenv("K8S_RBAC_SYNC_ENABLED", "true")
	t.Setenv("KUBECONFIG", "/tmp/kubeconfig")

	cfg := Load()

	if !cfg.K8SRBACSyncEnabled {
		t.Fatal("expected k8s rbac sync enabled")
	}
	if cfg.Kubeconfig != "/tmp/kubeconfig" {
		t.Fatalf("expected kubeconfig path, got %q", cfg.Kubeconfig)
	}
}

func TestLoadAgentServerAddrFromEnv(t *testing.T) {
	t.Setenv("AGENT_SERVER_ADDR", "127.0.0.1:18082")

	cfg := Load()

	if cfg.AgentServerAddr != "127.0.0.1:18082" {
		t.Fatalf("expected agent server addr from env, got %q", cfg.AgentServerAddr)
	}
}
