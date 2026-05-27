package postgres

import "k8s-ai-ops/backend/internal/domain"

// ─── 持久化模型 ↔ 领域实体 映射函数 ───

func toDomainUser(m *userModel) *domain.User {
	return &domain.User{
		ID: m.ID, Username: m.Username, DisplayName: m.DisplayName,
		Email: domain.Email{}, Role: domain.UserRole(m.Role), Status: m.Status,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func fromDomainUser(u *domain.User) *userModel {
	return &userModel{
		ID: u.ID, Username: u.Username, DisplayName: u.DisplayName,
		Email: u.Email.String(), Role: string(u.Role), Status: u.Status,
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
	}
}

func toDomainPerm(m *permModel) domain.Permission {
	return domain.Permission{
		ID: m.ID, UserID: m.UserID, Namespace: m.Namespace, APIGroup: m.APIGroup,
		Resource: m.Resource, Verbs: m.Verbs, Enabled: m.Enabled,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func fromDomainPerm(p *domain.Permission) *permModel {
	return &permModel{
		ID: p.ID, UserID: p.UserID, Namespace: p.Namespace, APIGroup: p.APIGroup,
		Resource: p.Resource, Verbs: p.Verbs, Enabled: p.Enabled,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toDomainProvider(m *providerModel) *domain.LLMProvider {
	return &domain.LLMProvider{
		ID: m.ID, Name: m.Name, Protocol: domain.Protocol(m.Protocol),
		BaseURL: m.BaseURL, APIKeyCiphertext: m.APIKeyCiphertext,
		Enabled: m.Enabled, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func fromDomainProvider(p *domain.LLMProvider) *providerModel {
	return &providerModel{
		ID: p.ID, Name: p.Name, Protocol: string(p.Protocol),
		BaseURL: p.BaseURL, APIKeyCiphertext: p.APIKeyCiphertext,
		Enabled: p.Enabled, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toDomainModel(m *modelModel) *domain.LLMModel {
	return &domain.LLMModel{
		ID: m.ID, ProviderID: m.ProviderID, ModelName: m.ModelName,
		DisplayName: m.DisplayName, SupportsTools: m.SupportsTools,
		SupportsStreaming: m.SupportsStreaming, Enabled: m.Enabled,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func fromDomainModel(m *domain.LLMModel) *modelModel {
	return &modelModel{
		ID: m.ID, ProviderID: m.ProviderID, ModelName: m.ModelName,
		DisplayName: m.DisplayName, SupportsTools: m.SupportsTools,
		SupportsStreaming: m.SupportsStreaming, Enabled: m.Enabled,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func toDomainAudit(m *auditModel) domain.AuditLog {
	return domain.AuditLog{
		ID: m.ID, ActorUserID: m.ActorUserID, Action: m.Action,
		TargetType: m.TargetType, TargetID: m.TargetID,
		Namespace: m.Namespace, Resource: m.Resource, Verb: m.Verb,
		Allowed: m.Allowed, Reason: m.Reason, CreatedAt: m.CreatedAt,
	}
}

func fromDomainAudit(l *domain.AuditLog) *auditModel {
	return &auditModel{
		ID: l.ID, ActorUserID: l.ActorUserID, Action: l.Action,
		TargetType: l.TargetType, TargetID: l.TargetID,
		Namespace: l.Namespace, Resource: l.Resource, Verb: l.Verb,
		Allowed: l.Allowed, Reason: l.Reason, CreatedAt: l.CreatedAt,
	}
}

func toDomainServiceAccountToken(m *serviceAccountTokenModel, plaintextToken string) *domain.ServiceAccountToken {
	return &domain.ServiceAccountToken{
		UserID:    m.UserID,
		SAName:    m.ServiceAccount,
		Token:     plaintextToken,
		Namespace: m.Namespace,
		CACert:    m.CACert,
		APIServer: m.APIServer,
	}
}

func fromDomainServiceAccountToken(t *domain.ServiceAccountToken, ciphertextToken string) *serviceAccountTokenModel {
	return &serviceAccountTokenModel{
		UserID:          t.UserID,
		ServiceAccount:  t.SAName,
		Namespace:       t.Namespace,
		TokenCiphertext: ciphertextToken,
		CACert:          t.CACert,
		APIServer:       t.APIServer,
	}
}

func toDomainServiceAccountBinding(m *serviceAccountBindingModel) domain.ServiceAccountBinding {
	return domain.ServiceAccountBinding{
		ID:                 m.ID,
		UserID:             m.UserID,
		Namespace:          m.Namespace,
		ServiceAccountName: m.ServiceAccount,
		Status:             m.Status,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}

func fromDomainServiceAccountBinding(b *domain.ServiceAccountBinding) *serviceAccountBindingModel {
	return &serviceAccountBindingModel{
		ID:             b.ID,
		UserID:         b.UserID,
		Namespace:      b.Namespace,
		ServiceAccount: b.ServiceAccountName,
		Status:         b.Status,
		CreatedAt:      b.CreatedAt,
		UpdatedAt:      b.UpdatedAt,
	}
}
