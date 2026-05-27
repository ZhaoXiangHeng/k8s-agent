package postgres

import (
	"context"

	"k8s-ai-ops/backend/internal/domain"
)

type auditRepo DataStore

func (r *auditRepo) Append(ctx context.Context, l *domain.AuditLog) error {
	m := fromDomainAudit(l)
	m.CreatedAt = now()
	if m.ID == "" {
		m.ID = "audit-" + now().Format("20060102150405.000000000")
	}
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *auditRepo) FindAll(ctx context.Context) ([]domain.AuditLog, error) {
	var ms []auditModel
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]domain.AuditLog, 0, len(ms))
	for i := range ms {
		out = append(out, toDomainAudit(&ms[i]))
	}
	return out, nil
}
