package app

import (
	"context"

	"k8s-ai-ops/backend/internal/domain"
)

// AuditService 管理审计日志的记录和查询。
type AuditService struct{ repos *domain.Repositories }

// Record 记录一条审计事件。
func (s *AuditService) Record(ctx context.Context, actorID, action, targetType, targetID string, allowed bool, reason string) error {
	log := domain.NewAuditLog(actorID, action, targetType, targetID)
	if !allowed {
		log.Deny(reason)
	}
	return s.repos.Audit.Append(ctx, &log)
}

// List 返回全部审计日志。
func (s *AuditService) List(ctx context.Context) ([]AuditLogResponse, error) {
	logs, err := s.repos.Audit.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AuditLogResponse, 0, len(logs))
	for _, l := range logs {
		out = append(out, AuditLogResponse{
			ID: l.ID, ActorUserID: l.ActorUserID, Action: l.Action,
			TargetType: l.TargetType, TargetID: l.TargetID,
			Namespace: l.Namespace, Resource: l.Resource, Verb: l.Verb,
			Allowed: l.Allowed, Reason: l.Reason,
			CreatedAt: l.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}
	return out, nil
}
