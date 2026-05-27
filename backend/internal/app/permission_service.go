package app

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"k8s-ai-ops/backend/internal/domain"
)

// PermissionService 管理 K8s 业务权限的变更和 RBAC 同步。
type PermissionService struct {
	repos       *domain.Repositories
	rbacApplier RBACApplier
}

func NewPermissionService(repos *domain.Repositories, rbacApplier RBACApplier) *PermissionService {
	return &PermissionService{repos: repos, rbacApplier: rbacApplier}
}

// Update 替换用户权限并同步 K8s RBAC。
func (s *PermissionService) Update(ctx context.Context, userID string, req UpdatePermissionsRequest) ([]PermissionResponse, error) {
	perms := make([]domain.Permission, 0, len(req.Permissions))
	for _, item := range req.Permissions {
		perms = append(perms, domain.Permission{
			Namespace: item.Namespace, APIGroup: item.APIGroup,
			Resource: item.Resource, Verbs: item.Verbs, Enabled: true,
		})
	}
	if err := s.repos.Permissions.Replace(ctx, userID, perms); err != nil {
		return nil, fmt.Errorf("replace permissions: %w", err)
	}
	if s.rbacApplier != nil {
		if err := s.rbacApplier.Apply(ctx, userID, perms); err != nil {
			return nil, fmt.Errorf("apply rbac: %w", err)
		}
	}
	appLog.WithFields(logrus.Fields{
		"event": "permissions_updated", "user_id": userID, "count": len(perms),
	}).Info("permissions updated")
	result := make([]PermissionResponse, 0, len(perms))
	for _, p := range perms {
		result = append(result, PermissionResponse{
			ID: p.ID, Namespace: p.Namespace, APIGroup: p.APIGroup,
			Resource: p.Resource, Verbs: p.Verbs, Enabled: p.Enabled,
		})
	}
	return result, nil
}
