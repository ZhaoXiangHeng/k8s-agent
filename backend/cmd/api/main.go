package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"google.golang.org/grpc"

	agentclient "k8s-ai-ops/backend/internal/agent"
	"k8s-ai-ops/backend/internal/cache"
	"k8s-ai-ops/backend/internal/config"
	apihttp "k8s-ai-ops/backend/internal/http"
	"k8s-ai-ops/backend/internal/identity"
	k8sops "k8s-ai-ops/backend/internal/k8s"
	"k8s-ai-ops/backend/internal/store"
	agentv1 "k8s-ai-ops/proto/agent/v1"
	identityv1 "k8s-ai-ops/proto/identity/v1"
)

func main() {
	cfg := config.Load()
	appStore := store.Store(store.NewMemoryStore())
	if cfg.StoreDriver == "postgres" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		postgresStore, err := store.NewPostgresStore(ctx, cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("level=ERROR component=backend-api event=postgres_connect_failed error=%q", err)
		}
		defer postgresStore.Close()
		appStore = postgresStore
		log.Printf("level=INFO component=backend-api event=postgres_connected")
	}
	if cfg.CacheDriver == "redis" {
		redisClient := cache.NewRedisClient(cfg.RedisAddr)
		if err := redisClient.Ping(); err != nil {
			log.Fatalf("level=ERROR component=backend-api event=redis_connect_failed addr=%s error=%q", cfg.RedisAddr, err)
		}
		log.Printf("level=INFO component=backend-api event=redis_connected addr=%s", cfg.RedisAddr)
	}
	server := apihttp.NewServer(appStore)
	agentConn, err := agentclient.Dial(context.Background(), cfg.AgentServerAddr)
	if err != nil {
		log.Fatalf("level=ERROR component=backend-api event=agent_server_connect_failed addr=%s error=%q", cfg.AgentServerAddr, err)
	}
	defer agentConn.Close()
	server.SetAgentClient(agentclient.NewGRPCClient(agentv1.NewAgentServiceClient(agentConn)))
	log.Printf("level=INFO component=backend-api event=agent_server_connected addr=%s", cfg.AgentServerAddr)
	if cfg.K8SRBACSyncEnabled {
		client, err := k8sops.NewClientset(cfg.Kubeconfig)
		if err != nil {
			log.Fatalf("level=ERROR component=backend-api event=k8s_client_create_failed error=%q", err)
		}
		server.SetRBACApplier(k8sops.NewRBACManager(client))
		log.Printf("level=INFO component=backend-api event=k8s_rbac_sync_enabled")
	} else {
		log.Printf("level=INFO component=backend-api event=k8s_rbac_sync_disabled")
	}
	grpcAddr := os.Getenv("GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = ":8082"
	}
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("level=ERROR component=backend-api event=grpc_listen_failed addr=%s error=%q", grpcAddr, err)
	}
	grpcServer := grpc.NewServer()
	identityv1.RegisterIdentityServiceServer(grpcServer, identity.NewServer(appStore))
	go func() {
		log.Printf("level=INFO component=backend-api event=grpc_server_start addr=%s", grpcAddr)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatalf("level=ERROR component=backend-api event=grpc_server_exit error=%q", err)
		}
	}()

	log.Printf("level=INFO component=backend-api event=server_start addr=%s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, server); err != nil {
		log.Fatalf("level=ERROR component=backend-api event=server_exit error=%q", err)
	}
}
