package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"k8s-ai-ops/backend/internal/app"
	"k8s-ai-ops/backend/internal/domain"
	"k8s-ai-ops/backend/internal/infra/agent"
	"k8s-ai-ops/backend/internal/infra/auth"
	"k8s-ai-ops/backend/internal/infra/cache"
	"k8s-ai-ops/backend/internal/infra/config"
	"k8s-ai-ops/backend/internal/infra/crypto"
	"k8s-ai-ops/backend/internal/infra/k8s"
	"k8s-ai-ops/backend/internal/infra/postgres"
	grpcserver "k8s-ai-ops/backend/internal/interfaces/grpc"
	apihttp "k8s-ai-ops/backend/internal/interfaces/http"
	"k8s.io/client-go/kubernetes"
	agentv1 "k8s-ai-ops/proto/agent/v1"
	identityv1 "k8s-ai-ops/proto/identity/v1"
)

var log = logrus.WithField("component", "backend-api")

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{TimestampFormat: "2006-01-02T15:04:05.000Z07:00"})
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetReportCaller(true)
}

func main() {
	cfg := config.Load()

	// 1. Infrastructure: PostgreSQL
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, closer, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.WithError(err).WithField("event", "postgres_connect_failed").Fatal("failed to connect to PostgreSQL")
	}
	defer closer()

	// 2. Infrastructure: Redis (optional)
	if cfg.CacheDriver == "redis" {
		rc := cache.NewRedisClient(cfg.RedisAddr)
		if err := rc.Ping(); err != nil {
			log.WithError(err).WithField("event", "redis_connect_failed").Fatal("failed to connect to Redis")
		}
	}

	// 3. Domain: Repositories (from infrastructure)
	repos := &db.Repositories

	// 4. Infrastructure: K8s client & RBAC Manager
	var k8sClient kubernetes.Interface
	var rbacApplier app.RBACApplier
	if cfg.K8SRBACSyncEnabled {
		var err error
		k8sClient, err = k8s.NewClientset(cfg.Kubeconfig)
		if err != nil {
			log.WithError(err).WithField("event", "k8s_client_failed").Fatal("failed to create K8s client")
		}
		manager := k8s.NewRBACManager(k8sClient)
		rbacApplier = app.RBACApplierFunc(func(ctx context.Context, userID string, permissions []domain.Permission) error {
			byNamespace := map[string][]k8s.PermissionSpec{}
			for _, permission := range permissions {
				byNamespace[permission.Namespace] = append(byNamespace[permission.Namespace], k8s.PermissionSpec{
					APIGroup: permission.APIGroup,
					Resource: permission.Resource,
					Verbs:    permission.Verbs,
				})
			}
			for namespace, rules := range byNamespace {
				if err := manager.ApplyUserNamespacePermissions(ctx, k8s.UserNamespacePermissions{
					UserID:    userID,
					Namespace: namespace,
					Rules:     rules,
				}); err != nil {
					return err
				}
			}
			return nil
		})
	}
	// IdentityServer 需要 K8s client 获取 admin SA token，可能独立于 RBAC sync
	if k8sClient == nil {
		var err error
		k8sClient, err = k8s.NewClientset(cfg.Kubeconfig)
		if err != nil {
			log.WithError(err).WithField("event", "k8s_client_failed").Warn("failed to create K8s client for identity service")
		}
	}

	// 5. Application: Services
	svc := app.NewServices(repos, crypto.Cipher{}, rbacApplier)

	// 6. Interface: gRPC IdentityService（先于 Agent 连接启动，mcp-server 需要此 gRPC）
	grpcAddr := os.Getenv("GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = ":8082"
	}
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.WithError(err).WithField("event", "grpc_listen_failed").Fatal("failed to listen")
	}
	grpcSrv := grpc.NewServer()
	var identityTokenProv *k8s.TokenProvider
	if k8sClient != nil {
		identityTokenProv = k8s.NewTokenProvider(k8sClient)
	}
	identityv1.RegisterIdentityServiceServer(grpcSrv, grpcserver.NewIdentityServer(
		repos.ServiceAccts, repos.Users, identityTokenProv,
		cfg.AdminSAName, cfg.AdminSANamespace,
	))
	go func() {
		if err := grpcSrv.Serve(grpcListener); err != nil {
			log.WithError(err).WithField("event", "grpc_server_exit").Fatal("gRPC server exited")
		}
	}()

	// 7. Infrastructure: Agent Server gRPC client（带重试）
	agentConn, err := agent.Dial(context.Background(), cfg.AgentServerAddr)
	if err != nil {
		log.WithError(err).WithField("event", "agent_connect_failed").Fatal("failed to connect to Agent Server")
	}
	defer agentConn.Close()
	agentClient := agent.NewGRPCClient(agentv1.NewAgentServiceClient(agentConn))
	svc.SetChatService(app.NewChatService(repos, agentClientAdapter{inner: agentClient}, crypto.Cipher{}))
	log.WithField("event", "agent_server_connected").Info("Agent Server ready")

	// 8. Interface: HTTP Server
	httpSrv := apihttp.NewServer(svc)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	jwtValidator := auth.NewJWTValidator(auth.Mode(cfg.AuthMode), cfg.KeycloakIssuer)
	apihttp.RegisterRoutes(r, httpSrv, jwtValidator)

	// 9. Start HTTP Server with graceful shutdown
	httpServer := &http.Server{Addr: cfg.HTTPAddr, Handler: r}
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		httpServer.Shutdown(shutdownCtx)
		grpcSrv.GracefulStop()
	}()

	log.WithField("event", "server_start").WithField("http_addr", cfg.HTTPAddr).Info("Backend API starting")
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.WithError(err).WithField("event", "server_exit").Fatal("HTTP server exited")
	}
}

type agentClientAdapter struct {
	inner agent.Client
}

func (a agentClientAdapter) RunStream(ctx context.Context, req *agentv1.AgentRunRequest) (app.AgentStream, error) {
	return a.inner.RunStream(ctx, req)
}
