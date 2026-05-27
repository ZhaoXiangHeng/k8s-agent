package postgres

import (
	"context"

	"gorm.io/gorm/clause"

	"k8s-ai-ops/backend/internal/domain"
)

// ─── Provider ───

type providerRepo DataStore

func (r *providerRepo) FindAll(ctx context.Context) ([]domain.LLMProvider, error) {
	var ms []providerModel
	if err := r.db.WithContext(ctx).Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]domain.LLMProvider, 0, len(ms))
	for i := range ms {
		out = append(out, *toDomainProvider(&ms[i]))
	}
	return out, nil
}

func (r *providerRepo) FindByID(ctx context.Context, id string) (*domain.LLMProvider, error) {
	var m providerModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		return nil, domain.ErrNotFound
	}
	return toDomainProvider(&m), nil
}

func (r *providerRepo) Save(ctx context.Context, p *domain.LLMProvider) error {
	m := fromDomainProvider(p)
	m.UpdatedAt = now()
	if m.ID == "" {
		m.ID = "provider-" + p.Name
		m.CreatedAt = now()
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(m).Error
}

// ─── Model ───

type modelRepo DataStore

func (r *modelRepo) FindAll(ctx context.Context) ([]domain.LLMModel, error) {
	var ms []modelModel
	if err := r.db.WithContext(ctx).Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]domain.LLMModel, 0, len(ms))
	for i := range ms {
		out = append(out, *toDomainModel(&ms[i]))
	}
	return out, nil
}

func (r *modelRepo) FindByID(ctx context.Context, id string) (*domain.LLMModel, error) {
	var m modelModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		return nil, domain.ErrNotFound
	}
	return toDomainModel(&m), nil
}

func (r *modelRepo) Save(ctx context.Context, m *domain.LLMModel) error {
	pm := fromDomainModel(m)
	pm.UpdatedAt = now()
	if pm.ID == "" {
		pm.ID = "model-" + m.ModelName
		pm.CreatedAt = now()
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(pm).Error
}

// ─── Binding ───

type bindingRepo DataStore

func (r *bindingRepo) FindByUser(ctx context.Context, uid string) ([]domain.LLMBinding, error) {
	var ms []bindingModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", uid).Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]domain.LLMBinding, 0, len(ms))
	for _, m := range ms {
		out = append(out, domain.LLMBinding{UserID: m.UserID, ModelID: m.ModelID, IsDefault: m.IsDefault})
	}
	return out, nil
}

func (r *bindingRepo) Replace(ctx context.Context, uid string, bindings []domain.LLMBinding) error {
	r.db.WithContext(ctx).Where("user_id = ?", uid).Delete(&bindingModel{})
	for _, b := range bindings {
		m := &bindingModel{UserID: uid, ModelID: b.ModelID, IsDefault: b.IsDefault, CreatedAt: now()}
		m.ID = uid + "-" + b.ModelID
		if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
			return err
		}
	}
	return nil
}
