package chat

import (
	"errors"
	"fmt"
	"strings"
)

type Permission struct {
	Namespace string
	APIGroup  string
	Resource  string
	Verbs     []string
}

func (p Permission) Allows(namespace, apiGroup, resource, verb string) bool {
	if p.Namespace != namespace || p.APIGroup != apiGroup || p.Resource != resource {
		return false
	}
	for _, allowedVerb := range p.Verbs {
		if allowedVerb == verb {
			return true
		}
	}
	return false
}

type UserContext struct {
	UserID      string
	Username    string
	Permissions []Permission
}

type ToolRequest struct {
	Name      string
	Namespace string
	APIGroup  string
	Resource  string
	Verb      string
}

func BuildSystemPrompt(ctx UserContext) string {
	var builder strings.Builder
	builder.WriteString("You are a Kubernetes AI operations assistant.\n")
	builder.WriteString("Current user: ")
	builder.WriteString(ctx.Username)
	builder.WriteString("\nAllowed Kubernetes permissions:\n")
	for _, permission := range ctx.Permissions {
		builder.WriteString(fmt.Sprintf(
			"- namespace=%s apiGroup=%s resource=%s verbs=%s\n",
			permission.Namespace,
			permission.APIGroup,
			permission.Resource,
			strings.Join(permission.Verbs, ","),
		))
	}
	builder.WriteString("Never request tools outside these permissions. Backend authorization and Kubernetes RBAC will deny unauthorized calls.")
	return builder.String()
}

func AuthorizeTool(ctx UserContext, request ToolRequest) error {
	for _, permission := range ctx.Permissions {
		if permission.Allows(request.Namespace, request.APIGroup, request.Resource, request.Verb) {
			return nil
		}
	}
	return errors.New("tool call denied by user Kubernetes permissions")
}

type AbnormalPod struct {
	Namespace    string `json:"namespace"`
	Name         string `json:"name"`
	Phase        string `json:"phase"`
	Reason       string `json:"reason"`
	Message      string `json:"message"`
	RestartCount int    `json:"restartCount"`
	Node         string `json:"node"`
}

type InspectionResult struct {
	Summary string        `json:"summary"`
	Pods    []AbnormalPod `json:"pods"`
}
