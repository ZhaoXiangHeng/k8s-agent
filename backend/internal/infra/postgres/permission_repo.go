package postgres

import (
	"context"

	"k8s-ai-ops/backend/internal/domain"
)

type permRepo DataStore

func (r *permRepo) FindByUser(ctx context.Context, uid string) ([]domain.Permission, error) {
	var ms []permModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", uid).Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Permission, 0, len(ms))
	for i := range ms {
		out = append(out, toDomainPerm(&ms[i]))
	}
	return out, nil
}

func (r *permRepo) Replace(ctx context.Context, uid string, perms []domain.Permission) error {
	r.db.WithContext(ctx).Where("user_id = ?", uid).Delete(&permModel{})
	for i := range perms {
		m := fromDomainPerm(&perms[i])
		m.UserID = uid
		m.CreatedAt = now()
		m.UpdatedAt = now()
		if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
			return err
		}
	}
	return nil
}
