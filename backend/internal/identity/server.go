package identity

import (
	"context"
	"fmt"

	"k8s-ai-ops/backend/internal/store"
	identityv1 "k8s-ai-ops/proto/identity/v1"
)

type Server struct {
	identityv1.UnimplementedIdentityServiceServer
	store store.Store
}

func NewServer(store store.Store) *Server {
	return &Server{store: store}
}

func (s *Server) GetServiceAccount(ctx context.Context, req *identityv1.GetServiceAccountRequest) (*identityv1.GetServiceAccountResponse, error) {
	sa, err := s.store.GetServiceAccount(req.GetUserId())
	if err != nil {
		return nil, fmt.Errorf("get service account: %w", err)
	}
	return &identityv1.GetServiceAccountResponse{
		SaName:    sa.SAName,
		Namespace: sa.Namespace,
		Token:     sa.Token,
		CaCert:    sa.CACert,
		ApiServer: sa.APIServer,
	}, nil
}
