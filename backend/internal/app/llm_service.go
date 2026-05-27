package app

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"k8s-ai-ops/backend/internal/domain"
)

// LLMService 管理 LLM Provider 和 Model 的配置。
type LLMService struct {
	repos  *domain.Repositories
	cipher SecretCipher
}

func NewLLMService(repos *domain.Repositories, cipher SecretCipher) *LLMService {
	return &LLMService{repos: repos, cipher: cipher}
}

// ─── Provider ───

func (s *LLMService) ListProviders(ctx context.Context) ([]ProviderResponse, error) {
	providers, err := s.repos.Providers.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ProviderResponse, 0, len(providers))
	for _, p := range providers {
		out = append(out, ProviderResponse{
			ID: p.ID, Name: p.Name, Protocol: string(p.Protocol),
			BaseURL: p.BaseURL, Enabled: p.Enabled,
			APIKeyConfigured: p.APIKeyCiphertext != "",
		})
	}
	return out, nil
}

func (s *LLMService) CreateProvider(ctx context.Context, req CreateProviderRequest) (*ProviderResponse, error) {
	proto, err := domain.NewProtocol(req.Protocol)
	if err != nil {
		return nil, err
	}
	encrypted, err := s.cipher.Encrypt(req.APIKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt api key: %w", err)
	}
	provider := &domain.LLMProvider{
		Name: req.Name, Protocol: proto, BaseURL: req.BaseURL,
		APIKeyCiphertext: encrypted, Enabled: req.Enabled,
	}
	if err := s.repos.Providers.Save(ctx, provider); err != nil {
		return nil, fmt.Errorf("save provider: %w", err)
	}
	appLog.WithFields(logrus.Fields{
		"event": "llm_provider_created", "provider_id": provider.ID, "name": provider.Name,
	}).Info("provider created")
	return &ProviderResponse{
		ID: provider.ID, Name: provider.Name, Protocol: string(provider.Protocol),
		BaseURL: provider.BaseURL, Enabled: provider.Enabled,
		APIKeyConfigured: provider.APIKeyCiphertext != "",
	}, nil
}

func (s *LLMService) UpdateProvider(ctx context.Context, id string, req UpdateProviderRequest) (*ProviderResponse, error) {
	p, err := s.repos.Providers.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.BaseURL != nil {
		p.BaseURL = *req.BaseURL
	}
	if req.APIKey != nil && *req.APIKey != "" {
		enc, err := s.cipher.Encrypt(*req.APIKey)
		if err != nil {
			return nil, err
		}
		p.APIKeyCiphertext = enc
	}
	if req.Enabled != nil {
		if *req.Enabled {
			p.Enable()
		} else {
			p.Disable()
		}
	}
	if err := s.repos.Providers.Save(ctx, p); err != nil {
		return nil, err
	}
	appLog.WithField("event", "llm_provider_updated").WithField("provider_id", id).Info("provider updated")
	return &ProviderResponse{
		ID: p.ID, Name: p.Name, Protocol: string(p.Protocol),
		BaseURL: p.BaseURL, Enabled: p.Enabled, APIKeyConfigured: p.APIKeyCiphertext != "",
	}, nil
}

// ─── Model ───

func (s *LLMService) ListModels(ctx context.Context, userID string, enabledOnly bool) ([]ModelResponse, error) {
	models, err := s.repos.Models.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	allowedModelIDs := map[string]bool{}
	if userID != "" {
		bindings, err := s.repos.Bindings.FindByUser(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("find user model bindings: %w", err)
		}
		for _, binding := range bindings {
			allowedModelIDs[binding.ModelID] = true
		}
	}
	out := make([]ModelResponse, 0, len(models))
	for _, m := range models {
		if enabledOnly && !m.Enabled {
			continue
		}
		if userID != "" && !allowedModelIDs[m.ID] {
			continue
		}
		out = append(out, ModelResponse{
			ID: m.ID, ProviderID: m.ProviderID, ModelName: m.ModelName,
			DisplayName: m.DisplayName, SupportsTools: m.SupportsTools,
			SupportsStreaming: m.SupportsStreaming, Enabled: m.Enabled,
		})
	}
	return out, nil
}

func (s *LLMService) CreateModel(ctx context.Context, req CreateModelRequest) (*ModelResponse, error) {
	m := &domain.LLMModel{
		ProviderID: req.ProviderID, ModelName: req.ModelName, DisplayName: req.DisplayName,
		SupportsTools: req.SupportsTools, SupportsStreaming: req.SupportsStreaming, Enabled: req.Enabled,
	}
	if err := s.repos.Models.Save(ctx, m); err != nil {
		return nil, fmt.Errorf("save model: %w", err)
	}
	appLog.WithFields(logrus.Fields{"event": "llm_model_created", "model_id": m.ID}).Info("model created")
	return &ModelResponse{
		ID: m.ID, ProviderID: m.ProviderID, ModelName: m.ModelName,
		DisplayName: m.DisplayName, SupportsTools: m.SupportsTools,
		SupportsStreaming: m.SupportsStreaming, Enabled: m.Enabled,
	}, nil
}

func (s *LLMService) UpdateModel(ctx context.Context, id string, req UpdateModelRequest) (*ModelResponse, error) {
	m, err := s.repos.Models.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.ProviderID != nil {
		m.ProviderID = *req.ProviderID
	}
	if req.DisplayName != nil {
		m.DisplayName = *req.DisplayName
	}
	if req.SupportsTools != nil {
		m.SupportsTools = *req.SupportsTools
	}
	if req.SupportsStreaming != nil {
		m.SupportsStreaming = *req.SupportsStreaming
	}
	if req.Enabled != nil {
		if *req.Enabled {
			m.Enable()
		} else {
			m.Disable()
		}
	}
	if err := s.repos.Models.Save(ctx, m); err != nil {
		return nil, err
	}
	appLog.WithField("event", "llm_model_updated").WithField("model_id", id).Info("model updated")
	return &ModelResponse{
		ID: m.ID, ProviderID: m.ProviderID, ModelName: m.ModelName,
		DisplayName: m.DisplayName, SupportsTools: m.SupportsTools,
		SupportsStreaming: m.SupportsStreaming, Enabled: m.Enabled,
	}, nil
}

func (s *LLMService) DeleteModel(ctx context.Context, id string) error {
	m, err := s.repos.Models.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("find model: %w", err)
	}
	m.Disable()
	if err := s.repos.Models.Save(ctx, m); err != nil {
		return fmt.Errorf("disable model: %w", err)
	}
	users, err := s.repos.Users.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("find users for binding cleanup: %w", err)
	}
	for _, u := range users {
		bindings, err := s.repos.Bindings.FindByUser(ctx, u.ID)
		if err != nil {
			continue
		}
		filtered := make([]domain.LLMBinding, 0, len(bindings))
		for _, b := range bindings {
			if b.ModelID != id {
				filtered = append(filtered, b)
			}
		}
		if len(filtered) != len(bindings) {
			if err := s.repos.Bindings.Replace(ctx, u.ID, filtered); err != nil {
				appLog.WithError(err).WithField("user_id", u.ID).Warn("failed to cleanup model binding")
			}
		}
	}
	appLog.WithFields(logrus.Fields{"event": "llm_model_deleted", "model_id": id}).Info("model disabled and bindings cleaned up")
	return nil
}
