// Package k8s 封装了 Kubernetes client-go，提供面向业务场景的简化操作接口。
// 支持通过 ServiceAccount Token 创建 per-user 的 K8s 客户端，
// 所有查询默认限定在用户 ServiceAccount 所在的 namespace，实现租户隔离。
package k8s

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var log = logrus.WithField("component", "mcp-server/k8s")

// Client 封装了 Kubernetes clientset 和用户的默认 namespace。
type Client struct {
	clientset        kubernetes.Interface
	defaultNamespace string // 用户 ServiceAccount 所在的 namespace，空值时回落到此
}

// PodInfo 是 Pod 关键信息的精简视图，用于 AI Agent 理解和展示。
type PodInfo struct {
	Namespace   string            `json:"namespace"`
	Name        string            `json:"name"`
	Phase       string            `json:"phase"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	NodeName    string            `json:"nodeName"`
	Containers  []ContainerInfo   `json:"containers"`
	CreatedAt   string            `json:"createdAt"`
}

// ContainerInfo 描述 Pod 中单个容器的运行状态。
type ContainerInfo struct {
	Name         string `json:"name"`
	Image        string `json:"image"`
	Ready        bool   `json:"ready"`
	RestartCount int32  `json:"restartCount"`
	State        string `json:"state"`            // Running / Waiting / Terminated / Unknown
	Reason       string `json:"reason,omitempty"` // 非 Running 时的原因
	Message      string `json:"message,omitempty"`
}

// EventInfo 是 K8s Event 的简化表示。
type EventInfo struct {
	Type      string `json:"type"`
	Reason    string `json:"reason"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// DeploymentInfo 是 Deployment 关键状态的精简视图。
type DeploymentInfo struct {
	Namespace  string `json:"namespace"`
	Name       string `json:"name"`
	Replicas   int32  `json:"replicas"`
	ReadyReps  int32  `json:"readyReplicas"`
	UpdatedRep int32  `json:"updatedReplicas"`
	Available  int32  `json:"availableReplicas"`
}

// NewClientFromSA 使用 ServiceAccount Token 创建 K8s 客户端。
// 生产环境中 caCert 由 Backend 下发，本地演示环境可能为空（此时跳过 TLS 验证）。
func NewClientFromSA(saToken, apiServer, namespace, caCert string) (*Client, error) {
	config := &rest.Config{
		Host:        apiServer,
		BearerToken: saToken,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: []byte(caCert),
		},
	}
	if caCert == "" {
		// 本地演示环境可能没有 CA 证书；生产环境应由 Backend 下发 caCert。
		config.TLSClientConfig = rest.TLSClientConfig{Insecure: true}
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"operation":  "NewClientFromSA",
			"api_server": apiServer,
			"namespace":  namespace,
		}).Error("failed to create kubernetes clientset from service account")
		return nil, err
	}
	return &Client{clientset: clientset, defaultNamespace: namespace}, nil
}

// ListNamespaces 返回集群中所有 namespace 的名称列表。
func (c *Client) ListNamespaces(ctx context.Context) ([]string, error) {
	nsList, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.WithError(err).WithField("operation", "ListNamespaces").Error("failed to list namespaces")
		return nil, err
	}
	names := make([]string, 0, len(nsList.Items))
	for _, ns := range nsList.Items {
		names = append(names, ns.Name)
	}
	return names, nil
}

// ListPods 列出指定 namespace 中的 Pod，支持 label 过滤。
// namespace 为空时回落到用户默认 namespace。
func (c *Client) ListPods(ctx context.Context, namespace, labelSelector string) ([]PodInfo, error) {
	ns := c.namespaceOrDefault(namespace)
	podList, err := c.clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"operation":      "ListPods",
			"namespace":      ns,
			"label_selector": labelSelector,
		}).Error("failed to list pods")
		return nil, err
	}
	result := make([]PodInfo, 0, len(podList.Items))
	for _, pod := range podList.Items {
		result = append(result, toPodInfo(pod))
	}
	return result, nil
}

// GetPod 获取指定 Pod 的详细信息。
func (c *Client) GetPod(ctx context.Context, namespace, name string) (*PodInfo, error) {
	pod, err := c.clientset.CoreV1().Pods(c.namespaceOrDefault(namespace)).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"operation": "GetPod",
			"namespace": namespace,
			"name":      name,
		}).Error("failed to get pod")
		return nil, err
	}
	info := toPodInfo(*pod)
	return &info, nil
}

// GetPodLogs 获取指定 Pod 容器的日志，支持限制尾部行数。
func (c *Client) GetPodLogs(ctx context.Context, namespace, name, container string, tailLines int64) (string, error) {
	logs, err := c.clientset.CoreV1().Pods(c.namespaceOrDefault(namespace)).GetLogs(name, &corev1.PodLogOptions{
		Container: container,
		TailLines: &tailLines,
	}).DoRaw(ctx)
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"operation":  "GetPodLogs",
			"namespace":  namespace,
			"name":       name,
			"container":  container,
			"tail_lines": tailLines,
		}).Error("failed to get pod logs")
		return "", err
	}
	return string(logs), nil
}

// ListEvents 获取与指定 Pod 相关的 K8s Events，用于排查问题。
func (c *Client) ListEvents(ctx context.Context, namespace, podName string) ([]EventInfo, error) {
	events, err := c.clientset.CoreV1().Events(c.namespaceOrDefault(namespace)).List(ctx, metav1.ListOptions{
		FieldSelector: "involvedObject.name=" + podName,
	})
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"operation": "ListEvents",
			"namespace": namespace,
			"pod_name":  podName,
		}).Error("failed to list events")
		return nil, err
	}
	result := make([]EventInfo, 0, len(events.Items))
	for _, e := range events.Items {
		result = append(result, EventInfo{
			Type:      e.Type,
			Reason:    e.Reason,
			Message:   e.Message,
			Timestamp: e.LastTimestamp.Format("2006-01-02T15:04:05Z"),
		})
	}
	return result, nil
}

// ListDeployments 列出指定 namespace 中的 Deployment 及其状态。
func (c *Client) ListDeployments(ctx context.Context, namespace string) ([]DeploymentInfo, error) {
	depList, err := c.clientset.AppsV1().Deployments(c.namespaceOrDefault(namespace)).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"operation": "ListDeployments",
			"namespace": namespace,
		}).Error("failed to list deployments")
		return nil, err
	}
	result := make([]DeploymentInfo, 0, len(depList.Items))
	for _, d := range depList.Items {
		replicas := int32(0)
		if d.Spec.Replicas != nil {
			replicas = *d.Spec.Replicas
		}
		result = append(result, DeploymentInfo{
			Namespace:  d.Namespace,
			Name:       d.Name,
			Replicas:   replicas,
			ReadyReps:  d.Status.ReadyReplicas,
			UpdatedRep: d.Status.UpdatedReplicas,
			Available:  d.Status.AvailableReplicas,
		})
	}
	return result, nil
}

// RestartDeployment 通过 Patch 更新 restartedAt 注解来触发滚动重启。
// 使用 Patch 而非 Update，因为权限模型只授予 deployments.apps/patch。
func (c *Client) RestartDeployment(ctx context.Context, namespace, name string) error {
	namespace = c.namespaceOrDefault(namespace)
	restartedAt := metav1.Now().Format("2006-01-02T15:04:05Z")
	payload := map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"metadata": map[string]any{
					"annotations": map[string]string{
						"kubectl.kubernetes.io/restartedAt": restartedAt,
					},
				},
			},
		},
	}
	patch, err := json.Marshal(payload)
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"operation": "RestartDeployment",
			"namespace": namespace,
			"name":      name,
		}).Error("failed to marshal restart patch payload")
		return err
	}
	// 权限模型只授予 deployments.apps/patch，不能使用 Update 触发滚动重启。
	_, err = c.clientset.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"operation": "RestartDeployment",
			"namespace": namespace,
			"name":      name,
		}).Error("failed to patch deployment for restart")
	}
	return err
}

// namespaceOrDefault 当 namespace 为空时回落到用户 ServiceAccount 所在 namespace。
// 空 namespace 在 client-go 中代表跨 namespace 查询，这里需要显式限制范围。
func (c *Client) namespaceOrDefault(namespace string) string {
	if namespace != "" {
		return namespace
	}
	// 空 namespace 在 client-go 中代表跨 namespace 查询，这里回落到用户 ServiceAccount 所在 namespace。
	return c.defaultNamespace
}

// containerState 从 ContainerStatus 中提取容器的运行状态、原因和消息。
func containerState(c corev1.ContainerStatus) (string, string, string) {
	if c.State.Running != nil {
		return "Running", "", ""
	}
	if c.State.Waiting != nil {
		return "Waiting", c.State.Waiting.Reason, c.State.Waiting.Message
	}
	if c.State.Terminated != nil {
		return "Terminated", c.State.Terminated.Reason, c.State.Terminated.Message
	}
	return "Unknown", "", ""
}

// toPodInfo 将 K8s Pod 对象转换为业务层的 PodInfo 结构。
func toPodInfo(pod corev1.Pod) PodInfo {
	containers := make([]ContainerInfo, 0, len(pod.Status.ContainerStatuses))
	for _, cs := range pod.Status.ContainerStatuses {
		state, reason, message := containerState(cs)
		containers = append(containers, ContainerInfo{
			Name:         cs.Name,
			Image:        cs.Image,
			Ready:        cs.Ready,
			RestartCount: cs.RestartCount,
			State:        state,
			Reason:       reason,
			Message:      message,
		})
	}
	return PodInfo{
		Namespace:   pod.Namespace,
		Name:        pod.Name,
		Phase:       string(pod.Status.Phase),
		Labels:      pod.Labels,
		Annotations: pod.Annotations,
		NodeName:    pod.Spec.NodeName,
		Containers:  containers,
		CreatedAt:   pod.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
	}
}
