// Package grpc 提供 gRPC 服务实现。
package grpc

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"k8s-ai-ops/backend/internal/domain"
	identityv1 "k8s-ai-ops/proto/identity/v1"
)

var pkgLog = logrus.WithField("component", "backend-api/identity")

// IdentityServer 实现 identityv1.IdentityServiceServer。
type IdentityServer struct {
	identityv1.UnimplementedIdentityServiceServer
	saRepo domain.ServiceAccountRepository
}

// NewIdentityServer 创建 gRPC Identity 服务。
func NewIdentityServer(repo domain.ServiceAccountRepository) *IdentityServer {
	return &IdentityServer{saRepo: repo}
}

// GetServiceAccount 根据 user_id 返回 K8s ServiceAccount 凭据。
func (s *IdentityServer) GetServiceAccount(ctx context.Context, req *identityv1.GetServiceAccountRequest) (*identityv1.GetServiceAccountResponse, error) {
	userID := req.GetUserId()
	pkgLog.WithFields(logrus.Fields{"event": "identity_lookup", "user_id": userID}).Info("resolving service account")
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
