package k8s

import (
	"reflect"
	"testing"
)

func TestBuildRBACNamesUsesUserAndNamespace(t *testing.T) {
	names := BuildRBACNames("u123", "dev")

	if names.ServiceAccount != "k8s-ai-operator-u123" {
		t.Fatalf("unexpected service account name: %s", names.ServiceAccount)
	}
	if names.Role != "k8s-ai-role-u123-dev" {
		t.Fatalf("unexpected role name: %s", names.Role)
	}
	if names.RoleBinding != "k8s-ai-binding-u123-dev" {
		t.Fatalf("unexpected role binding name: %s", names.RoleBinding)
	}
}

func TestBuildRoleRulesGroupsResourcesByAPIGroupAndVerbs(t *testing.T) {
	permissions := []PermissionSpec{
		{APIGroup: "", Resource: "pods", Verbs: []string{"get", "list"}},
		{APIGroup: "", Resource: "pods/log", Verbs: []string{"get"}},
		{APIGroup: "apps", Resource: "deployments", Verbs: []string{"get", "patch"}},
	}

	rules := BuildRoleRules(permissions)

	expected := []RoleRule{
		{APIGroups: []string{""}, Resources: []string{"pods"}, Verbs: []string{"get", "list"}},
		{APIGroups: []string{""}, Resources: []string{"pods/log"}, Verbs: []string{"get"}},
		{APIGroups: []string{"apps"}, Resources: []string{"deployments"}, Verbs: []string{"get", "patch"}},
	}
	if !reflect.DeepEqual(rules, expected) {
		t.Fatalf("unexpected role rules:\nwant: %#v\n got: %#v", expected, rules)
	}
}
