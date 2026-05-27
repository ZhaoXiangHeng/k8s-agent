package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestListNamespacesReturnsNames(t *testing.T) {
	client := NewFakeClient()
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "dev"}}
	client.clientset.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})

	names, err := client.ListNamespaces(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 {
		t.Fatalf("expected 1 namespace, got %d", len(names))
	}
}

func TestListPodsReturnsPodInfo(t *testing.T) {
	client := NewFakeClient()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "dev"},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	client.clientset.CoreV1().Pods("dev").Create(context.Background(), pod, metav1.CreateOptions{})

	pods, err := client.ListPods(context.Background(), "dev", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(pods) != 1 || pods[0].Name != "test-pod" || pods[0].Phase != "Running" {
		t.Fatalf("unexpected pods: %#v", pods)
	}
}
