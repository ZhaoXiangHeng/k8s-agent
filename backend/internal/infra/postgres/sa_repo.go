package postgres

import (
	"context"

	"gorm.io/gorm/clause"

	"k8s-ai-ops/backend/internal/domain"
	"k8s-ai-ops/backend/internal/infra/crypto"
)

type saRepo DataStore

func (r *saRepo) FindToken(ctx context.Context, uid string) (*domain.ServiceAccountToken, error) {
	var m serviceAccountTokenModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", uid).First(&m).Error; err != nil {
		return nil, domain.ErrNotFound
	}
	token, err := crypto.Decrypt(m.TokenCiphertext)
	if err != nil {
		return nil, err
	}
	return toDomainServiceAccountToken(&m, token), nil
}

func (r *saRepo) SaveToken(ctx context.Context, t *domain.ServiceAccountToken) error {
	token, err := crypto.Encrypt(t.Token)
	if err != nil {
		return err
	}
	m := fromDomainServiceAccountToken(t, token)
	m.UpdatedAt = now()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now()
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(m).Error
}

func (r *saRepo) SaveBinding(ctx context.Context, b *domain.ServiceAccountBinding) error {
	m := fromDomainServiceAccountBinding(b)
	m.UpdatedAt = now()
	if m.ID == "" {
		m.ID = b.UserID + "-" + b.Namespace + "-" + b.ServiceAccountName
	}
	if m.Status == "" {
		m.Status = "active"
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now()
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(m).Error
}

func (r *saRepo) FindBindings(ctx context.Context, uid string) ([]domain.ServiceAccountBinding, error) {
	var ms []serviceAccountBindingModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", uid).Order("namespace, service_account").Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ServiceAccountBinding, 0, len(ms))
	for i := range ms {
		out = append(out, toDomainServiceAccountBinding(&ms[i]))
	}
	return out, nil
}
