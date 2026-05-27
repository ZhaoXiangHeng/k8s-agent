package app

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"k8s-ai-ops/backend/internal/domain"
)

// UserService 管理用户和权限的查询与变更。
type UserService struct{ repos *domain.Repositories }

// List 返回全部用户列表。
func (s *UserService) List(ctx context.Context) ([]UserResponse, error) {
	users, err := s.repos.Users.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]UserResponse, 0, len(users))
	for _, u := range users {
		out = append(out, toUserResponse(&u))
	}
	return out, nil
}

// Create 创建新用户。
func (s *UserService) Create(ctx context.Context, req CreateUserRequest) (*UserResponse, error) {
	role, err := domain.NewUserRole(req.Role)
	if err != nil {
		return nil, err
	}
	user, err := domain.NewUser(req.Username, role)
	if err != nil {
		return nil, err
	}
	user.DisplayName = req.DisplayName
	if email, err := domain.NewEmail(req.Email); err == nil {
		user.Email = email
	}
	if err := s.repos.Users.Save(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}
	appLog.WithFields(logrus.Fields{
		"event": "user_created", "user_id": user.ID, "username": user.Username,
	}).Info("user created")
	r := toUserResponse(user)
	return &r, nil
}

// GetPermissions 返回用户的 K8s 权限列表。
func (s *UserService) GetPermissions(ctx context.Context, userID string) ([]PermissionResponse, error) {
	perms, err := s.repos.Permissions.FindByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]PermissionResponse, 0, len(perms))
	for _, p := range perms {
		out = append(out, PermissionResponse{
			ID: p.ID, Namespace: p.Namespace, APIGroup: p.APIGroup,
			Resource: p.Resource, Verbs: p.Verbs, Enabled: p.Enabled,
		})
	}
	return out, nil
}

// GetCurrentInfo 返回当前登录用户的基本信息。
func (s *UserService) GetCurrentInfo(_ context.Context, userID, username, role string) *UserResponse {
	return &UserResponse{ID: userID, Username: username, Role: role, Status: "active"}
}

func toUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID: u.ID, Username: u.Username, DisplayName: u.DisplayName,
		Email: u.Email.String(), Role: string(u.Role), Status: u.Status,
	}
}
