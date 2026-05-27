package domain

import "time"

// ServiceAccountBinding 记录用户与 K8s ServiceAccount 的绑定关系。
type ServiceAccountBinding struct {
	ID                 string
	UserID             string
	Namespace          string
	ServiceAccountName string
	Status             string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// ServiceAccountToken 存储用户的 K8s ServiceAccount 运行时凭据。
// 基础设施层持久化时必须加密 Token 字段。
type ServiceAccountToken struct {
	UserID    string
	SAName    string
	Token     string
	Namespace string
	CACert    string
	APIServer string
}
