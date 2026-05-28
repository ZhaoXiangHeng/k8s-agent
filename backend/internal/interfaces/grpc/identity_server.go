// Package grpc 提供 gRPC 服务实现。
package grpc

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"k8s-ai-ops/backend/internal/domain"
	"k8s-ai-ops/backend/internal/infra/k8s"
	identityv1 "k8s-ai-ops/proto/identity/v1"
)

var pkgLog = logrus.WithField("component", "backend-api/identity")

// IdentityServer 实现 identityv1.IdentityServiceServer。
type IdentityServer struct {
	identityv1.UnimplementedIdentityServiceServer
	saRepo       domain.ServiceAccountRepository
	userRepo     domain.UserRepository
	tokenProv    *k8s.TokenProvider
	adminSAName  string
	adminSANs    string
}

// NewIdentityServer 创建 gRPC Identity 服务。
func NewIdentityServer(
	saRepo domain.ServiceAccountRepository,
	userRepo domain.UserRepository,
	tokenProv *k8s.TokenProvider,
	adminSAName, adminSANs string,
) *IdentityServer {
	return &IdentityServer{
		saRepo: saRepo, userRepo: userRepo, tokenProv: tokenProv,
		adminSAName: adminSAName, adminSANs: adminSANs,
	}
}

// GetServiceAccount 根据 user_id 返回 K8s ServiceAccount 凭据。
// admin 用户共享集群管理员 SA，operator 用户使用各自 namespace 级别的 SA。
func (s *IdentityServer) GetServiceAccount(ctx context.Context, req *identityv1.GetServiceAccountRequest) (*identityv1.GetServiceAccountResponse, error) {
	userID := req.GetUserId()
	pkgLog.WithFields(logrus.Fields{"event": "identity_lookup", "user_id": userID}).Info("resolving service account")

	// admin 用户：返回共享的集群管理员 SA
	user, err := s.userRepo.FindByID(ctx, userID)
	if err == nil && user != nil && user.Role == domain.RoleAdmin {
		return s.resolveAdminToken(ctx, userID)
	}

	// operator 用户：返回各自 namespace 级别的 SA
	token, err := s.saRepo.FindToken(ctx, userID)
	if err != nil {
		pkgLog.WithError(err).WithFields(logrus.Fields{"event": "identity_lookup_failed", "user_id": userID}).Error("failed")
		return nil, fmt.Errorf("get service account: %w", err)
	}
	return &identityv1.GetServiceAccountResponse{
		SaName: token.SAName, Namespace: token.Namespace,
		Token: token.Token, CaCert: token.CACert, ApiServer: token.APIServer,
	}, nil
}

func (s *IdentityServer) resolveAdminToken(ctx context.Context, userID string) (*identityv1.GetServiceAccountResponse, error) {
	token, caCert, namespace, err := s.tokenProv.GetServiceAccountToken(ctx, s.adminSANs, s.adminSAName)
	if err != nil {
		pkgLog.WithError(err).WithFields(logrus.Fields{
			"event": "admin_token_failed", "user_id": userID,
			"admin_sa": fmt.Sprintf("%s/%s", s.adminSANs, s.adminSAName),
		}).Error("failed to resolve admin SA token")
		return nil, fmt.Errorf("get service account: %w", err)
	}
	pkgLog.WithFields(logrus.Fields{
		"event": "admin_token_resolved", "user_id": userID,
	}).Info("resolved admin SA token")
	return &identityv1.GetServiceAccountResponse{
		SaName: s.adminSAName, Namespace: namespace,
		Token: token, CaCert: caCert,
	}, nil
}
