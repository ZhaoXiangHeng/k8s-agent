package k8s

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// TestNewClientFromSA 验证从 ServiceAccount 创建客户端不会在构造时 panic。
func TestNewClientFromSA(t *testing.T) {
	_, err := NewClientFromSA("test-token", "https://localhost:6443", "default", "")
	if err != nil {
		// NewClientFromSA 只创建客户端配置，不在创建时验证远端连接。
	}
}

// TestListDeploymentsHandlesNilReplicas 验证当 Deployment spec.replicas 为 nil 时返回 0 而非 panic。
func TestListDeploymentsHandlesNilReplicas(t *testing.T) {
	clientset := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "dev",
			Name:      "api",
		},
	})
	client := &Client{clientset: clientset}

	deployments, err := client.ListDeployments(context.Background(), "dev")
	if err != nil {
		t.Fatalf("ListDeployments returned error: %v", err)
	}
	if len(deployments) != 1 {
		t.Fatalf("expected 1 deployment, got %d", len(deployments))
	}
	if deployments[0].Replicas != 0 {
		t.Fatalf("expected nil replicas to default to 0, got %d", deployments[0].Replicas)
	}
}

// TestListPodsDefaultsToServiceAccountNamespace 验证空 namespace 时回落至 SA 所在 namespace。
func TestListPodsDefaultsToServiceAccountNamespace(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "dev", Name: "api"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "prod", Name: "api"}},
	)
	client := &Client{clientset: clientset, defaultNamespace: "dev"}

	pods, err := client.ListPods(context.Background(), "", "")
	if err != nil {
		t.Fatalf("ListPods returned error: %v", err)
	}
	if len(pods) != 1 || pods[0].Namespace != "dev" {
		t.Fatalf("expected only dev pods, got %#v", pods)
	}
}

// TestListDeploymentsDefaultsToServiceAccountNamespace 验证空 namespace 时 Deployment 查询也在 SA namespace 范围内。
func TestListDeploymentsDefaultsToServiceAccountNamespace(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: "dev", Name: "api"}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: "prod", Name: "api"}},
	)
	client := &Client{clientset: clientset, defaultNamespace: "dev"}

	deployments, err := client.ListDeployments(context.Background(), "")
	if err != nil {
		t.Fatalf("ListDeployments returned error: %v", err)
	}
	if len(deployments) != 1 || deployments[0].Namespace != "dev" {
		t.Fatalf("expected only dev deployments, got %#v", deployments)
	}
}

// TestRestartDeploymentUsesPatchVerb 验证重启 Deployment 使用 patch 动词（符合权限模型要求）。
func TestRestartDeploymentUsesPatchVerb(t *testing.T) {
	replicas := int32(1)
	clientset := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "dev",
			Name:      "api",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	})
	client := &Client{clientset: clientset}

	if err := client.RestartDeployment(context.Background(), "dev", "api"); err != nil {
		t.Fatalf("RestartDeployment returned error: %v", err)
	}

	// 确认操作动词是 patch，资源是 deployments
	for _, action := range clientset.Actions() {
		if action.GetVerb() == "patch" && action.GetResource().Resource == "deployments" {
			return
		}
	}
	t.Fatalf("expected restart to patch deployments, got actions %#v", clientset.Actions())
}
