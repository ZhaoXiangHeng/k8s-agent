package domain

import (
	"fmt"
	"strings"
	"time"
)

// ─── 值对象 ───

// UserRole 是平台用户角色。
type UserRole string

const (
	RoleAdmin    UserRole = "admin"
	RoleOperator UserRole = "operator"
)

// NewUserRole 创建角色值对象，校验合法性。
func NewUserRole(s string) (UserRole, error) {
	r := UserRole(strings.ToLower(strings.TrimSpace(s)))
	switch r {
	case RoleAdmin, RoleOperator:
		return r, nil
	}
	return "", fmt.Errorf("%w: invalid role %q", ErrInvalidInput, s)
}

// Email 是用户邮箱值对象。
type Email struct{ value string }

// NewEmail 创建并校验邮箱。
func NewEmail(s string) (Email, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Email{}, nil // 邮箱可选
	}
	if !strings.Contains(s, "@") {
		return Email{}, fmt.Errorf("%w: invalid email %q", ErrInvalidInput, s)
	}
	return Email{value: strings.ToLower(s)}, nil
}

func (e Email) String() string { return e.value }

// ─── User 聚合根 ───

// User 是平台用户聚合根，管理用户身份和绑定的权限与 LLM 模型。
type User struct {
	ID          string
	Username    string
	DisplayName string
	Email       Email
	Role        UserRole
	Status      string       // "active" | "disabled"
	Permissions []Permission // 用户拥有的 K8s 业务权限
	LLMBindings []LLMBinding // 用户可用的 LLM 模型绑定
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewUser 创建用户聚合，确保必需字段和默认值。
func NewUser(username string, role UserRole) (*User, error) {
	if strings.TrimSpace(username) == "" {
		return nil, fmt.Errorf("%w: username required", ErrInvalidInput)
	}
	return &User{
		Username:    username,
		Role:        role,
		Status:      "active",
		Permissions: []Permission{},
		LLMBindings: []LLMBinding{},
	}, nil
}

// Disable 停用用户。
func (u *User) Disable() {
	u.Status = "disabled"
	u.UpdatedAt = time.Now()
}

// Enable 启用用户。
func (u *User) Enable() {
	u.Status = "active"
	u.UpdatedAt = time.Now()
}

// ReplacePermissions 替换用户的全部权限。
func (u *User) ReplacePermissions(permissions []Permission) {
	for i := range permissions {
		permissions[i].UserID = u.ID
	}
	u.Permissions = permissions
	u.UpdatedAt = time.Now()
}

// SetLLMBindings 设置用户绑定的 LLM 模型。
func (u *User) SetLLMBindings(bindings []LLMBinding) {
	for i := range bindings {
		bindings[i].UserID = u.ID
	}
	u.LLMBindings = bindings
	u.UpdatedAt = time.Now()
}

// HasPermission 判断用户是否可对指定 (namespace, resource) 执行 verb。
func (u *User) HasPermission(namespace, resource, verb string) bool {
	for _, p := range u.Permissions {
		if p.Allows(namespace, resource, verb) {
			return true
		}
	}
	return false
}

// LLMBinding 是用户与 LLM 模型的绑定值对象。
type LLMBinding struct {
	UserID    string
	ModelID   string
	IsDefault bool
}
