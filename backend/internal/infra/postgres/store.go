package postgres

import (
	"context"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"k8s-ai-ops/backend/internal/domain"
)

// DataStore 聚合全部 PostgreSQL 仓储实现。
type DataStore struct {
	db *gorm.DB
	domain.Repositories
}

// New 创建 DataStore，执行 AutoMigrate，返回 close 函数。
func New(ctx context.Context, dsn string) (*DataStore, func() error, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}
	s := &DataStore{db: db}
	s.Repositories = domain.Repositories{
		Users:        (*userRepo)(s),
		Permissions:  (*permRepo)(s),
		Providers:    (*providerRepo)(s),
		Models:       (*modelRepo)(s),
		Bindings:     (*bindingRepo)(s),
		ChatSessions: (*sessionRepo)(s),
		ChatMessages: (*messageRepo)(s),
		ServiceAccts: (*saRepo)(s),
		Audit:        (*auditRepo)(s),
	}
	if err := s.db.AutoMigrate(allModels...); err != nil {
		return nil, nil, err
	}
	sqlDB, _ := db.DB()
	return s, sqlDB.Close, nil
}

func now() time.Time { return time.Now().UTC() }
