package chat

import (
	"strings"
	"testing"
)

func TestBuildSystemPromptIncludesOnlyAllowedPermissions(t *testing.T) {
	prompt := BuildSystemPrompt(UserContext{
		Username: "operator-a",
		Permissions: []Permission{
			{Namespace: "dev", APIGroup: "", Resource: "pods", Verbs: []string{"get", "list"}},
		},
	})

	if !strings.Contains(prompt, "operator-a") {
		t.Fatal("expected prompt to contain username")
	}
	if !strings.Contains(prompt, "namespace=dev apiGroup= resource=pods verbs=get,list") {
		t.Fatalf("expected prompt to contain permission summary, got: %s", prompt)
	}
	if strings.Contains(prompt, "prod") {
		t.Fatalf("prompt leaked unexpected namespace: %s", prompt)
	}
}

func TestAuthorizeToolDeniesUnauthorizedNamespace(t *testing.T) {
	ctx := UserContext{
		Permissions: []Permission{
			{Namespace: "dev", APIGroup: "", Resource: "pods", Verbs: []string{"list"}},
		},
	}

	err := AuthorizeTool(ctx, ToolRequest{
		Name:      "list_pods",
		Namespace: "prod",
		APIGroup:  "",
		Resource:  "pods",
		Verb:      "list",
	})

	if err == nil {
		t.Fatal("expected unauthorized namespace to be denied")
	}
}

func TestAuthorizeToolAllowsAuthorizedRead(t *testing.T) {
	ctx := UserContext{
		Permissions: []Permission{
			{Namespace: "dev", APIGroup: "", Resource: "pods", Verbs: []string{"list"}},
		},
	}

	err := AuthorizeTool(ctx, ToolRequest{
		Name:      "list_pods",
		Namespace: "dev",
		APIGroup:  "",
		Resource:  "pods",
		Verb:      "list",
	})

	if err != nil {
		t.Fatalf("expected authorized tool call, got %v", err)
	}
}
