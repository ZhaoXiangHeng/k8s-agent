package k8s

import "fmt"

type RBACNames struct {
	ServiceAccount string
	Role           string
	RoleBinding    string
}

type PermissionSpec struct {
	APIGroup string
	Resource string
	Verbs    []string
}

type RoleRule struct {
	APIGroups []string
	Resources []string
	Verbs     []string
}

func BuildRBACNames(userID, namespace string) RBACNames {
	return RBACNames{
		ServiceAccount: fmt.Sprintf("k8s-ai-operator-%s", userID),
		Role:           fmt.Sprintf("k8s-ai-role-%s-%s", userID, namespace),
		RoleBinding:    fmt.Sprintf("k8s-ai-binding-%s-%s", userID, namespace),
	}
}

func BuildRoleRules(permissions []PermissionSpec) []RoleRule {
	rules := make([]RoleRule, 0, len(permissions))
	for _, permission := range permissions {
		rules = append(rules, RoleRule{
			APIGroups: []string{permission.APIGroup},
			Resources: []string{permission.Resource},
			Verbs:     append([]string(nil), permission.Verbs...),
		})
	}
	return rules
}
