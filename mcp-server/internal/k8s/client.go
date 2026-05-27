package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	clientset kubernetes.Interface
}

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

type ContainerInfo struct {
	Name         string `json:"name"`
	Image        string `json:"image"`
	Ready        bool   `json:"ready"`
	RestartCount int32  `json:"restartCount"`
	State        string `json:"state"`
	Reason       string `json:"reason,omitempty"`
	Message      string `json:"message,omitempty"`
}

type EventInfo struct {
	Type      string `json:"type"`
	Reason    string `json:"reason"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

type DeploymentInfo struct {
	Namespace  string `json:"namespace"`
	Name       string `json:"name"`
	Replicas   int32  `json:"replicas"`
	ReadyReps  int32  `json:"readyReplicas"`
	UpdatedRep int32  `json:"updatedReplicas"`
	Available  int32  `json:"availableReplicas"`
}

func NewClient(kubeconfig string) (*Client, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &Client{clientset: clientset}, nil
}

func (c *Client) ListNamespaces(ctx context.Context) ([]string, error) {
	nsList, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(nsList.Items))
	for _, ns := range nsList.Items {
		names = append(names, ns.Name)
	}
	return names, nil
}

func (c *Client) ListPods(ctx context.Context, namespace, labelSelector string) ([]PodInfo, error) {
	podList, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}
	result := make([]PodInfo, 0, len(podList.Items))
	for _, pod := range podList.Items {
		result = append(result, toPodInfo(pod))
	}
	return result, nil
}

func (c *Client) GetPod(ctx context.Context, namespace, name string) (*PodInfo, error) {
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	info := toPodInfo(*pod)
	return &info, nil
}

func (c *Client) GetPodLogs(ctx context.Context, namespace, name, container string, tailLines int64) (string, error) {
	logs, err := c.clientset.CoreV1().Pods(namespace).GetLogs(name, &corev1.PodLogOptions{
		Container: container,
		TailLines: &tailLines,
	}).DoRaw(ctx)
	if err != nil {
		return "", err
	}
	return string(logs), nil
}

func (c *Client) ListEvents(ctx context.Context, namespace, podName string) ([]EventInfo, error) {
	events, err := c.clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "involvedObject.name=" + podName,
	})
	if err != nil {
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

func (c *Client) ListDeployments(ctx context.Context, namespace string) ([]DeploymentInfo, error) {
	depList, err := c.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	result := make([]DeploymentInfo, 0, len(depList.Items))
	for _, d := range depList.Items {
		result = append(result, DeploymentInfo{
			Namespace:  d.Namespace,
			Name:       d.Name,
			Replicas:   *d.Spec.Replicas,
			ReadyReps:  d.Status.ReadyReplicas,
			UpdatedRep: d.Status.UpdatedReplicas,
			Available:  d.Status.AvailableReplicas,
		})
	}
	return result, nil
}

func (c *Client) RestartDeployment(ctx context.Context, namespace, name string) error {
	dep, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if dep.Spec.Template.Annotations == nil {
		dep.Spec.Template.Annotations = map[string]string{}
	}
	dep.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = metav1.Now().Format("2006-01-02T15:04:05Z")
	_, err = c.clientset.AppsV1().Deployments(namespace).Update(ctx, dep, metav1.UpdateOptions{})
	return err
}

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
