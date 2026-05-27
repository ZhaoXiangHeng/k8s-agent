package domain

import "time"

// AuditLog 记录系统中的安全和操作审计事件。
type AuditLog struct {
	ID          string
	ActorUserID string
	Action      string
	TargetType  string
	TargetID    string
	Namespace   string
	Resource    string
	Verb        string
	Allowed     bool
	Reason      string
	CreatedAt   time.Time
}

// NewAuditLog 创建一条审计记录。
func NewAuditLog(actorID, action, targetType, targetID string) AuditLog {
	return AuditLog{
		ActorUserID: actorID,
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		Allowed:     true,
		CreatedAt:   time.Now(),
	}
}

// Deny 标记此审计事件为拒绝并记录原因。
func (l *AuditLog) Deny(reason string) {
	l.Allowed = false
	l.Reason = reason
}
