package postgres

import (
	"context"

	"gorm.io/gorm/clause"

	"k8s-ai-ops/backend/internal/domain"
)

// ─── Session ───

type sessionRepo DataStore

func (r *sessionRepo) FindByID(ctx context.Context, id string) (*domain.ChatSession, error) {
	var m sessionModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		return nil, domain.ErrNotFound
	}
	return &domain.ChatSession{
		ID: m.ID, UserID: m.UserID, ModelID: m.ModelID,
		Title: m.Title, Status: m.Status,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}, nil
}

func (r *sessionRepo) FindByUser(ctx context.Context, uid string) ([]domain.ChatSession, error) {
	var ms []sessionModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", uid).Order("created_at DESC").Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ChatSession, 0, len(ms))
	for _, m := range ms {
		out = append(out, domain.ChatSession{
			ID: m.ID, UserID: m.UserID, Title: m.Title,
			Status: m.Status, CreatedAt: m.CreatedAt,
		})
	}
	return out, nil
}

func (r *sessionRepo) Save(ctx context.Context, s *domain.ChatSession) error {
	m := &sessionModel{
		ID: s.ID, UserID: s.UserID, ModelID: s.ModelID,
		Title: s.Title, Status: s.Status,
		CreatedAt: now(), UpdatedAt: now(),
	}
	if m.ID == "" {
		m.ID = "session-" + now().Format("20060102150405")
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(m).Error
}

// ─── Message ───

type messageRepo DataStore

func (r *messageRepo) Append(ctx context.Context, msg *domain.ChatMessage) error {
	m := &messageModel{
		ID: msg.ID, SessionID: msg.SessionID, Role: string(msg.Role),
		Content: msg.Content, ToolName: msg.ToolName, CreatedAt: now(),
	}
	if m.ID == "" {
		m.ID = "msg-" + now().Format("20060102150405.000000000")
	}
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *messageRepo) FindBySession(ctx context.Context, sid string) ([]domain.ChatMessage, error) {
	var ms []messageModel
	if err := r.db.WithContext(ctx).Where("session_id = ?", sid).Order("created_at").Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ChatMessage, len(ms))
	for i, m := range ms {
		out[i] = domain.ChatMessage{
			ID: m.ID, SessionID: m.SessionID,
			Role: domain.MessageRole(m.Role), Content: m.Content,
			ToolName: m.ToolName, CreatedAt: m.CreatedAt,
		}
	}
	return out, nil
}
