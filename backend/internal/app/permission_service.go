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

func (s *PermissionService) isAdmin(ctx context.Context, userID string) bool {
	user, err := s.repos.Users.FindByID(ctx, userID)
	return err == nil && user != nil && user.Role == domain.RoleAdmin
}

// Update 替换用户权限并同步 K8s RBAC。
// admin 用户拥有集群管理员权限，不需要单独分配 namespace 级别权限。
func (s *PermissionService) Update(ctx context.Context, userID string, req UpdatePermissionsRequest) ([]PermissionResponse, error) {
	if s.isAdmin(ctx, userID) {
		return nil, fmt.Errorf("admin users have cluster-admin access, no per-user permissions needed")
	}
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
