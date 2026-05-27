package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type UserNamespacePermissions struct {
	UserID    string
	Namespace string
	Rules     []PermissionSpec
}

type RBACManager struct {
	client kubernetes.Interface
}

func NewRBACManager(client kubernetes.Interface) *RBACManager {
	return &RBACManager{client: client}
}

func (m *RBACManager) ApplyUserNamespacePermissions(ctx context.Context, permissions UserNamespacePermissions) error {
	names := BuildRBACNames(permissions.UserID, permissions.Namespace)
	if err := m.applyServiceAccount(ctx, permissions.Namespace, names.ServiceAccount); err != nil {
		return err
	}
	if err := m.applyRole(ctx, permissions.Namespace, names.Role, permissions.Rules); err != nil {
		return err
	}
	return m.applyRoleBinding(ctx, permissions.Namespace, names)
}

func (m *RBACManager) applyServiceAccount(ctx context.Context, namespace, name string) error {
	desired := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    managedLabels(),
		},
	}
	_, err := m.client.CoreV1().ServiceAccounts(namespace).Create(ctx, desired, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func (m *RBACManager) applyRole(ctx context.Context, namespace, name string, permissions []PermissionSpec) error {
	desiredRules := toPolicyRules(permissions)
	existing, err := m.client.RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = m.client.RbacV1().Roles(namespace).Create(ctx, &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels:    managedLabels(),
			},
			Rules: desiredRules,
		}, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	existing.Labels = mergeLabels(existing.Labels, managedLabels())
	existing.Rules = desiredRules
	_, err = m.client.RbacV1().Roles(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

func (m *RBACManager) applyRoleBinding(ctx context.Context, namespace string, names RBACNames) error {
	desiredSubjects := []rbacv1.Subject{
		{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      names.ServiceAccount,
			Namespace: namespace,
		},
	}
	desiredRoleRef := rbacv1.RoleRef{
		APIGroup: rbacv1.GroupName,
		Kind:     "Role",
		Name:     names.Role,
	}
	existing, err := m.client.RbacV1().RoleBindings(namespace).Get(ctx, names.RoleBinding, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = m.client.RbacV1().RoleBindings(namespace).Create(ctx, &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      names.RoleBinding,
				Namespace: namespace,
				Labels:    managedLabels(),
			},
			Subjects: desiredSubjects,
			RoleRef:  desiredRoleRef,
		}, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	existing.Labels = mergeLabels(existing.Labels, managedLabels())
	existing.Subjects = desiredSubjects
	existing.RoleRef = desiredRoleRef
	_, err = m.client.RbacV1().RoleBindings(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

func toPolicyRules(permissions []PermissionSpec) []rbacv1.PolicyRule {
	rules := make([]rbacv1.PolicyRule, 0, len(permissions))
	for _, permission := range permissions {
		rules = append(rules, rbacv1.PolicyRule{
			APIGroups: []string{permission.APIGroup},
			Resources: []string{permission.Resource},
			Verbs:     append([]string(nil), permission.Verbs...),
		})
	}
	return rules
}

func managedLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "k8s-ai-ops",
		"app.kubernetes.io/managed-by": "k8s-ai-ops-backend",
	}
}

func mergeLabels(existing, extra map[string]string) map[string]string {
	merged := map[string]string{}
	for key, value := range existing {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}
