package store

import "testing"

func TestPermissionAllowsMatchingNamespaceResourceAndVerb(t *testing.T) {
	permission := Permission{
		Namespace: "dev",
		APIGroup:  "",
		Resource:  "pods",
		Verbs:     []string{"get", "list", "watch"},
		Enabled:   true,
	}

	if !permission.Allows("dev", "", "pods", "list") {
		t.Fatal("expected permission to allow listing dev pods")
	}
}

func TestPermissionDeniesDifferentNamespace(t *testing.T) {
	permission := Permission{
		Namespace: "dev",
		APIGroup:  "",
		Resource:  "pods",
		Verbs:     []string{"get", "list"},
		Enabled:   true,
	}

	if permission.Allows("prod", "", "pods", "list") {
		t.Fatal("expected permission to deny prod namespace")
	}
}

func TestPermissionDeniesDisabledRule(t *testing.T) {
	permission := Permission{
		Namespace: "dev",
		APIGroup:  "",
		Resource:  "pods",
		Verbs:     []string{"list"},
		Enabled:   false,
	}

	if permission.Allows("dev", "", "pods", "list") {
		t.Fatal("expected disabled permission to deny access")
	}
}
