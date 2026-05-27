package domain

import "time"

// Permission 表示用户在特定 namespace 下对 K8s 资源的访问权限。
type Permission struct {
	ID        string
	UserID    string
	Namespace string
	APIGroup  string
	Resource  string
	Verbs     []string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Allows 判断此权限是否允许指定操作。
func (p Permission) Allows(namespace, resource, verb string) bool {
	if !p.Enabled || p.Namespace != namespace || p.Resource != resource {
		return false
	}
	for _, v := range p.Verbs {
		if v == verb || v == "*" {
			return true
		}
	}
	return false
}

// PermissionSpec 是用于构建 K8s RBAC 规则的权限规格。
type PermissionSpec struct {
	APIGroup string
	Resource string
	Verbs    []string
}
