package k8s

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// TokenProvider 从 K8s Secret 中获取 ServiceAccount 的 bearer token。
type TokenProvider struct {
	client kubernetes.Interface
}

// NewTokenProvider 创建 TokenProvider。
func NewTokenProvider(client kubernetes.Interface) *TokenProvider {
	return &TokenProvider{client: client}
}

// GetServiceAccountToken 返回指定 SA 的 token（从 kubernetes.io/service-account-token 类型 Secret 读取）。
func (p *TokenProvider) GetServiceAccountToken(ctx context.Context, namespace, saName string) (string, string, string, error) {
	// 查找绑定到该 SA 的 token Secret
	secrets, err := p.client.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", "", "", fmt.Errorf("list secrets: %w", err)
	}
	for _, secret := range secrets.Items {
		if secret.Type != "kubernetes.io/service-account-token" {
			continue
		}
		if secret.Annotations["kubernetes.io/service-account.name"] != saName {
			continue
		}
		token := string(secret.Data["token"])
		if token == "" {
			continue
		}
		caCert := string(secret.Data["ca.crt"])
		namespace := string(secret.Data["namespace"])
		return token, caCert, namespace, nil
	}
	return "", "", "", fmt.Errorf("token secret not found for service account %s/%s", namespace, saName)
}
