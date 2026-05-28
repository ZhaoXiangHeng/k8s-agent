package auth

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestExtractRoleUsesRealmAccessRoles(t *testing.T) {
	claims := jwt.MapClaims{
		"realm_access": map[string]interface{}{
			"roles": []interface{}{"offline_access", "operator", "admin"},
		},
	}

	if got := extractRole(claims); got != "admin" {
		t.Fatalf("expected admin, got %q", got)
	}
}

func TestExtractRoleFallsBackToOperator(t *testing.T) {
	claims := jwt.MapClaims{
		"realm_access": map[string]interface{}{
			"roles": []interface{}{"offline_access", "operator"},
		},
	}

	if got := extractRole(claims); got != "operator" {
		t.Fatalf("expected operator, got %q", got)
	}
}
