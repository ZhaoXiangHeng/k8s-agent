package postgres

import (
	"context"

	"gorm.io/gorm/clause"

	"k8s-ai-ops/backend/internal/domain"
)

type userRepo DataStore

func (r *userRepo) FindAll(ctx context.Context) ([]domain.User, error) {
	var ms []userModel
	if err := r.db.WithContext(ctx).Order("created_at").Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]domain.User, 0, len(ms))
	for i := range ms {
		out = append(out, *toDomainUser(&ms[i]))
	}
	return out, nil
}

func (r *userRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	var m userModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		return nil, domain.ErrNotFound
	}
	return toDomainUser(&m), nil
}

func (r *userRepo) FindByUsername(ctx context.Context, name string) (*domain.User, error) {
	var m userModel
	if err := r.db.WithContext(ctx).Where("username = ?", name).First(&m).Error; err != nil {
		return nil, domain.ErrNotFound
	}
	return toDomainUser(&m), nil
}

func (r *userRepo) Save(ctx context.Context, u *domain.User) error {
	m := fromDomainUser(u)
	m.CreatedAt = now()
	m.UpdatedAt = now()
	if m.ID == "" {
		m.ID = "user-" + u.Username
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(m).Error
}
