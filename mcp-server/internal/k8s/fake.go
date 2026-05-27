package k8s

import "k8s.io/client-go/kubernetes/fake"

func NewFakeClient() *Client {
	return &Client{clientset: fake.NewSimpleClientset()}
}
