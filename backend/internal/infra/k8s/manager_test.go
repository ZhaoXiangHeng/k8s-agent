package k8s

import (
	"context"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRBACManagerApplyUserNamespacePermissions(t *testing.T) {
	client := fake.NewSimpleClientset()
	manager := NewRBACManager(client)

	err := manager.ApplyUserNamespacePermissions(context.Background(), UserNamespacePermissions{
		UserID:    "u123",
		Namespace: "dev",
		Rules: []PermissionSpec{
			{APIGroup: "", Resource: "pods", Verbs: []string{"get", "list", "watch"}},
			{APIGroup: "apps", Resource: "deployments", Verbs: []string{"get", "patch"}},
		},
	})

	if err != nil {
		t.Fatalf("apply rbac: %v", err)
	}

	names := BuildRBACNames("u123", "dev")
	if _, err := client.CoreV1().ServiceAccounts("dev").Get(context.Background(), names.ServiceAccount, metav1.GetOptions{}); err != nil {
		t.Fatalf("expected service account: %v", err)
	}

	role, err := client.RbacV1().Roles("dev").Get(context.Background(), names.Role, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("expected role: %v", err)
	}
	if len(role.Rules) != 2 {
		t.Fatalf("expected two rules, got %#v", role.Rules)
	}
	if role.Rules[0].Resources[0] != "pods" {
		t.Fatalf("expected pods rule, got %#v", role.Rules[0])
	}

	binding, err := client.RbacV1().RoleBindings("dev").Get(context.Background(), names.RoleBinding, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("expected role binding: %v", err)
	}
	if binding.RoleRef.Kind != "Role" || binding.RoleRef.Name != names.Role {
		t.Fatalf("unexpected role ref: %#v", binding.RoleRef)
	}
	if len(binding.Subjects) != 1 || binding.Subjects[0].Kind != rbacv1.ServiceAccountKind || binding.Subjects[0].Name != names.ServiceAccount {
		t.Fatalf("unexpected subjects: %#v", binding.Subjects)
	}
}

func TestRBACManagerUpdatesExistingRole(t *testing.T) {
	client := fake.NewSimpleClientset()
	manager := NewRBACManager(client)

	first := UserNamespacePermissions{
		UserID:    "u123",
		Namespace: "dev",
		Rules: []PermissionSpec{
			{APIGroup: "", Resource: "pods", Verbs: []string{"get"}},
		},
	}
	second := UserNamespacePermissions{
		UserID:    "u123",
		Namespace: "dev",
		Rules: []PermissionSpec{
			{APIGroup: "", Resource: "services", Verbs: []string{"get", "list"}},
		},
	}

	if err := manager.ApplyUserNamespacePermissions(context.Background(), first); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	if err := manager.ApplyUserNamespacePermissions(context.Background(), second); err != nil {
		t.Fatalf("second apply: %v", err)
	}

	names := BuildRBACNames("u123", "dev")
	role, err := client.RbacV1().Roles("dev").Get(context.Background(), names.Role, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("expected role: %v", err)
	}
	if role.Rules[0].Resources[0] != "services" {
		t.Fatalf("expected updated services rule, got %#v", role.Rules)
	}
}
