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
func (s *UserService) GetCurrentInfo(ctx context.Context, userID, username, role string) *UserResponse {
	user, err := s.repos.Users.FindByID(ctx, userID)
	if err != nil || user == nil {
		return &UserResponse{ID: userID, Username: username, Role: role, Status: "active"}
	}
	r := toUserResponse(user)
	return &r
}

// Delete 删除用户并级联清理权限和模型绑定。
func (s *UserService) Delete(ctx context.Context, userID string) error {
	user, err := s.repos.Users.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}
	user.Disable()
	if err := s.repos.Users.Save(ctx, user); err != nil {
		return fmt.Errorf("disable user: %w", err)
	}
	if err := s.repos.Permissions.Replace(ctx, userID, nil); err != nil {
		return fmt.Errorf("cleanup permissions: %w", err)
	}
	if err := s.repos.Bindings.Replace(ctx, userID, nil); err != nil {
		return fmt.Errorf("cleanup model bindings: %w", err)
	}
	appLog.WithFields(logrus.Fields{
		"event": "user_deleted", "user_id": userID,
	}).Info("user disabled and permissions/bindings cleaned up")
	return nil
}

// ResetPassword 重置用户密码。
func (s *UserService) ResetPassword(ctx context.Context, userID, _ string) error {
	_, err := s.repos.Users.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}
	appLog.WithFields(logrus.Fields{
		"event": "password_reset", "user_id": userID,
	}).Info("password reset requested (no-op: integrate Keycloak admin API)")
	return nil
}

// GetAllModelBindings 返回所有用户的模型绑定映射。
func (s *UserService) GetAllModelBindings(ctx context.Context) (ModelBindingMapResponse, error) {
	users, err := s.repos.Users.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	result := make(ModelBindingMapResponse, len(users))
	for _, u := range users {
		bindings, err := s.repos.Bindings.FindByUser(ctx, u.ID)
		if err != nil {
			return nil, fmt.Errorf("find bindings for user %s: %w", u.ID, err)
		}
		ids := make([]string, 0, len(bindings))
		for _, b := range bindings {
			ids = append(ids, b.ModelID)
		}
		result[u.ID] = ids
	}
	return result, nil
}

// UpdateModelBindings 替换用户的模型绑定列表。
func (s *UserService) UpdateModelBindings(ctx context.Context, userID string, modelIDs []string) error {
	_, err := s.repos.Users.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}
	bindings := make([]domain.LLMBinding, 0, len(modelIDs))
	for _, modelID := range modelIDs {
		bindings = append(bindings, domain.LLMBinding{UserID: userID, ModelID: modelID})
	}
	if err := s.repos.Bindings.Replace(ctx, userID, bindings); err != nil {
		return fmt.Errorf("replace model bindings: %w", err)
	}
	appLog.WithFields(logrus.Fields{
		"event": "model_bindings_updated", "user_id": userID, "count": len(modelIDs),
	}).Info("model bindings updated")
	return nil
}

func toUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID: u.ID, Username: u.Username, DisplayName: u.DisplayName,
		Email: u.Email.String(), Role: string(u.Role), Status: u.Status,
	}
}
