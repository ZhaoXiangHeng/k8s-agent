package postgres

import (
	"testing"
	"time"

	"k8s-ai-ops/backend/internal/domain"
)

func TestServiceAccountTokenMappingRoundTrip(t *testing.T) {
	now := time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC)
	model := &serviceAccountTokenModel{
		UserID:          "user-1",
		ServiceAccount:  "k8s-ai-operator-user-1",
		Namespace:       "default",
		TokenCiphertext: "cipher-token",
		CACert:          "ca-data",
		APIServer:       "https://kubernetes.default.svc",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	token := toDomainServiceAccountToken(model, "plain-token")
	if token.UserID != model.UserID || token.SAName != model.ServiceAccount || token.Token != "plain-token" {
		t.Fatalf("unexpected token mapping: %#v", token)
	}

	roundTrip := fromDomainServiceAccountToken(token, "cipher-token")
	if roundTrip.UserID != model.UserID || roundTrip.ServiceAccount != model.ServiceAccount || roundTrip.TokenCiphertext != model.TokenCiphertext {
		t.Fatalf("unexpected token persistence mapping: %#v", roundTrip)
	}
}

func TestServiceAccountBindingMappingRoundTrip(t *testing.T) {
	binding := &domain.ServiceAccountBinding{
		ID:                 "binding-1",
		UserID:             "user-1",
		Namespace:          "default",
		ServiceAccountName: "k8s-ai-operator-user-1",
		Status:             "active",
	}

	model := fromDomainServiceAccountBinding(binding)
	roundTrip := toDomainServiceAccountBinding(model)

	if roundTrip.UserID != binding.UserID || roundTrip.Namespace != binding.Namespace || roundTrip.ServiceAccountName != binding.ServiceAccountName {
		t.Fatalf("unexpected binding mapping: %#v", roundTrip)
	}
}

func TestServiceAccountModelsAreMigrated(t *testing.T) {
	tables := map[string]bool{}
	for _, model := range allModels {
		if named, ok := model.(interface{ TableName() string }); ok {
			tables[named.TableName()] = true
		}
	}

	for _, table := range []string{"service_account_tokens", "service_account_bindings"} {
		if !tables[table] {
			t.Fatalf("expected %s to be registered in allModels", table)
		}
	}
}
