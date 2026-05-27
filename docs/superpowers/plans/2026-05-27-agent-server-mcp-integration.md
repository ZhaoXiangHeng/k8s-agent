# Agent Server MCP 集成 & 流式 ReAct 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 agent-server 从硬编码 mock 改造为基于 Eino ReAct 的智能 Agent，通过标准 MCP 协议调用 mcp-server，全链路流式推送事件。

**Architecture:** Frontend ←SSE→ Backend ←gRPC Stream→ Agent Server ←SSE (MCP)→ MCP Server ←client-go→ K8s API。Agent Server 按 ModelRuntimeConfig 动态创建 LLM ChatModel，Eino ReAct 自主决定调用 MCP 工具，LLM 分析结果。

**Tech Stack:** Go 1.26, Eino v0.9.0, eino-ext MCP, mark3labs/mcp-go, client-go, gRPC streaming, protobuf

---

## File Structure

```
proto/agent/v1/
├── agent.proto              ← 修改: RunStream RPC + StreamEvent 消息

mcp-server/
├── go.mod                   ← 修改: 添加 mark3labs/mcp-go, client-go
├── cmd/server/main.go        ← 修改: MCP Server + SSE 替换 REST
├── internal/
│   ├── mcp/
│   │   ├── types.go          ← 删除
│   │   ├── registry.go       ← 删除
│   │   └── registry_test.go  ← 删除
│   ├── tools/
│   │   ├── pods.go           ← 删除 (IsAbnormalPodPhase/Reason)
│   │   └── pods_test.go      ← 删除
│   ├── k8s/
│   │   ├── client.go         ← 重写: 真实 client-go K8s 客户端
│   │   ├── client_test.go    ← 重写
│   │   └── fake.go           ← 新增: fake client 供测试
│   └── handler/
│       ├── pods.go           ← 新增: list_pods, get_pod, get_pod_logs handler
│       ├── events.go         ← 新增: get_pod_events handler
│       ├── deployments.go    ← 新增: list_deployments, restart_deployment handler
│       └── namespaces.go     ← 新增: list_namespaces handler

agent-server/
├── go.mod                    ← 修改: 添加 eino-ext MCP, openai, anthropic
├── cmd/server/main.go         ← 修改: 启动时连接 MCP server
├── internal/
│   ├── eino/
│   │   ├── runner.go          ← 重写: Eino ReAct runner + RunStream
│   │   ├── runner_test.go     ← 重写
│   │   ├── config.go          ← 新增: MCP_SERVER_URL env
│   │   ├── llm/
│   │   │   ├── factory.go     ← 新增: 动态创建 ChatModel
│   │   │   └── factory_test.go
│   │   └── mcp/
│   │       ├── client.go      ← 新增: SSE MCP 客户端 + GetTools()
│   │       └── client_test.go
│   └── server/
│       ├── server.go          ← 修改: RunStream handler
│       ├── server_test.go     ← 修改
│       └── mock_runner.go     ← 修改: 改为 mock stream

backend/
├── internal/
│   ├── agent/
│   │   ├── client.go          ← 修改: Run() → RunStream()
│   │   └── client_test.go     ← 修改
│   ├── http/
│   │   └── router.go          ← 修改: createChatMessage 改为 SSE 流式
│   └── config/
│       └── config.go          ← 修改: 删除 MCPServerURL
```

---

### Task 1: 更新 Proto 定义

**Files:**
- Modify: `proto/agent/v1/agent.proto`

- [ ] **Step 1: 将 Run 替换为 RunStream，新增 StreamEvent 消息**

将 `proto/agent/v1/agent.proto` 中的 service 改为:

```protobuf
syntax = "proto3";

package agent.v1;

option go_package = "k8s-ai-ops/proto/agent/v1;agentv1";

service AgentService {
  rpc RunStream(AgentRunRequest) returns (stream StreamEvent);
}

message AgentRunRequest {
  string request_id = 1;
  string session_id = 2;
  string message_id = 3;
  UserContext user = 4;
  ModelRuntimeConfig model = 5;
  repeated Message messages = 6;
  string message = 7;
  repeated Permission permissions = 8;
  repeated ToolRef tools = 9;
  RuntimeContext runtime_context = 10;
}

message UserContext {
  string id = 1;
  string username = 2;
}

message ModelRuntimeConfig {
  string provider_id = 1;
  string protocol = 2;
  string base_url = 3;
  string api_key = 4;
  string model_name = 5;
  bool supports_tools = 6;
  bool supports_streaming = 7;
}

message Message {
  string role = 1;
  string content = 2;
}

message Permission {
  string namespace = 1;
  string api_group = 2;
  string resource = 3;
  repeated string verbs = 4;
}

message ToolRef {
  string name = 1;
  string namespace = 2;
  string api_group = 3;
  string resource = 4;
  string verb = 5;
}

message RuntimeContext {
  string current_user = 1;
  repeated string allowed_namespaces = 2;
  repeated ResourceRef recent_resources = 3;
}

message ResourceRef {
  string kind = 1;
  string api_group = 2;
  string namespace = 3;
  string name = 4;
  string summary = 5;
}

// --- 新增 Streaming 消息 ---

message StreamEvent {
  string event_id = 1;
  int64 timestamp = 2;

  oneof event {
    ThinkingEvent   thinking    = 10;
    ToolCallEvent   tool_call   = 11;
    ToolResultEvent tool_result = 12;
    ResourceEvent   resource    = 13;
    CompleteEvent   complete    = 14;
    ErrorEvent      error       = 15;
  }
}

message ThinkingEvent {
  string content = 1;
}

message ToolCallEvent {
  string tool_name = 1;
  string arguments_json = 2;
}

message ToolResultEvent {
  string tool_name = 1;
  bool success = 2;
  string result_json = 3;
}

message ResourceEvent {
  ResourceResult resource = 1;
}

message CompleteEvent {
  string summary = 1;
  repeated ResourceResult resources = 2;
}

message ErrorEvent {
  string code = 1;
  string message = 2;
}

// 保留 ResourceResult
message ResourceResult {
  string kind = 1;
  string api_group = 2;
  string namespace = 3;
  string name = 4;
  string status = 5;
}
```

- [ ] **Step 2: 重新生成 Go 代码**

```bash
cd e:/k8s-agent/proto && protoc --go_out=. --go-grpc_out=. agent/v1/agent.proto
```

- [ ] **Step 3: 验证编译**

```bash
cd e:/k8s-agent/agent-server && go build ./...
cd e:/k8s-agent/backend && go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add proto/agent/v1/agent.proto proto/agent/v1/*.pb.go
git commit -m "feat: replace Run with RunStream server-streaming RPC
- Add StreamEvent with oneof: Thinking/ToolCall/ToolResult/Resource/Complete/Error
- Remove AgentRunResponse and ToolEvent (replaced by stream events)
- Keep AgentRunRequest and all input messages unchanged"
```

---

### Task 2: MCP Server — 标准 MCP 协议 + K8s 客户端

**Files:**
- Modify: `mcp-server/go.mod`
- Modify: `mcp-server/cmd/server/main.go`
- Create: `mcp-server/internal/handler/namespaces.go`
- Create: `mcp-server/internal/handler/pods.go`
- Create: `mcp-server/internal/handler/events.go`
- Create: `mcp-server/internal/handler/deployments.go`
- Modify: `mcp-server/internal/k8s/client.go`
- Create: `mcp-server/internal/k8s/fake.go`
- Modify: `mcp-server/internal/k8s/client_test.go`
- Delete: `mcp-server/internal/mcp/types.go`
- Delete: `mcp-server/internal/mcp/registry.go`
- Delete: `mcp-server/internal/mcp/registry_test.go`
- Delete: `mcp-server/internal/tools/pods.go`
- Delete: `mcp-server/internal/tools/pods_test.go`

- [ ] **Step 1: 添加依赖**

```bash
cd e:/k8s-agent/mcp-server && go get github.com/mark3labs/mcp-go@latest k8s.io/client-go@v0.34.2 k8s.io/api@v0.34.2 k8s.io/apimachinery@v0.34.2
```

- [ ] **Step 2: 实现 K8s 客户端接口**

Write `mcp-server/internal/k8s/client.go`:

```go
package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/rest"
)

type Client struct {
	clientset kubernetes.Interface
}

type PodInfo struct {
	Namespace   string              `json:"namespace"`
	Name        string              `json:"name"`
	Phase       string              `json:"phase"`
	Labels      map[string]string   `json:"labels"`
	Annotations map[string]string   `json:"annotations"`
	NodeName    string              `json:"nodeName"`
	Containers  []ContainerInfo     `json:"containers"`
	CreatedAt   string              `json:"createdAt"`
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
```

- [ ] **Step 3: 实现 Fake 客户端**

Write `mcp-server/internal/k8s/fake.go`:

```go
package k8s

import (
	"context"

	"k8s.io/client-go/kubernetes/fake"
)

func NewFakeClient() *Client {
	return &Client{clientset: fake.NewSimpleClientset()}
}
```

- [ ] **Step 4: 重写 K8s 客户端测试**

Write `mcp-server/internal/k8s/client_test.go`:

```go
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
```

- [ ] **Step 5: 实现 handler — namespaces**

Write `mcp-server/internal/handler/namespaces.go`:

```go
package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"k8s-ai-ops/mcp-server/internal/k8s"
)

func ListNamespacesTool() mcp.Tool {
	return mcp.NewTool("list_namespaces",
		mcp.WithDescription("List all Kubernetes namespaces"),
	)
}

func HandleListNamespaces(client *k8s.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		names, err := client.ListNamespaces(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list namespaces: %v", err)), nil
		}
		data, _ := json.Marshal(names)
		return mcp.NewToolResultRaw(string(data)), nil
	}
}
```

- [ ] **Step 6: 实现 handler — pods**

Write `mcp-server/internal/handler/pods.go`:

```go
package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"k8s-ai-ops/mcp-server/internal/k8s"
)

func ListPodsTool() mcp.Tool {
	return mcp.NewTool("list_pods",
		mcp.WithDescription("List Kubernetes pods with optional namespace and label filters"),
		mcp.WithString("namespace",
			mcp.Description("Namespace to list pods from"),
		),
		mcp.WithString("label_selector",
			mcp.Description("Kubernetes label selector to filter pods"),
		),
	)
}

func HandleListPods(client *k8s.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments
		namespace := getStringArg(args, "namespace")
		labelSelector := getStringArg(args, "label_selector")
		pods, err := client.ListPods(ctx, namespace, labelSelector)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list pods: %v", err)), nil
		}
		data, _ := json.Marshal(pods)
		return mcp.NewToolResultRaw(string(data)), nil
	}
}

func GetPodTool() mcp.Tool {
	return mcp.NewTool("get_pod",
		mcp.WithDescription("Get detailed information about a specific Kubernetes pod"),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Pod namespace")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Pod name")),
	)
}

func HandleGetPod(client *k8s.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments
		namespace, _ := args["namespace"].(string)
		name, _ := args["name"].(string)
		pod, err := client.GetPod(ctx, namespace, name)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get pod: %v", err)), nil
		}
		data, _ := json.Marshal(pod)
		return mcp.NewToolResultRaw(string(data)), nil
	}
}

func GetPodLogsTool() mcp.Tool {
	return mcp.NewTool("get_pod_logs",
		mcp.WithDescription("Get logs from a Kubernetes pod container"),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Pod namespace")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Pod name")),
		mcp.WithString("container", mcp.Description("Container name (uses first container if not specified)")),
		mcp.WithNumber("tail_lines", mcp.Description("Number of lines from the end of the logs (default 50)")),
	)
}

func HandleGetPodLogs(client *k8s.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments
		namespace, _ := args["namespace"].(string)
		name, _ := args["name"].(string)
		container, _ := args["container"].(string)
		tailLines := int64(50)
		if v, ok := args["tail_lines"].(float64); ok {
			tailLines = int64(v)
		}
		logs, err := client.GetPodLogs(ctx, namespace, name, container, tailLines)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get pod logs: %v", err)), nil
		}
		return mcp.NewToolResultRaw(logs), nil
	}
}
```

- [ ] **Step 7: 实现 handler — events**

Write `mcp-server/internal/handler/events.go`:

```go
package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"k8s-ai-ops/mcp-server/internal/k8s"
)

func GetPodEventsTool() mcp.Tool {
	return mcp.NewTool("get_pod_events",
		mcp.WithDescription("List Kubernetes events related to a specific pod"),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Pod namespace")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Pod name")),
	)
}

func HandleGetPodEvents(client *k8s.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments
		namespace, _ := args["namespace"].(string)
		name, _ := args["name"].(string)
		events, err := client.ListEvents(ctx, namespace, name)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list events: %v", err)), nil
		}
		data, _ := json.Marshal(events)
		return mcp.NewToolResultRaw(string(data)), nil
	}
}
```

- [ ] **Step 8: 实现 handler — deployments**

Write `mcp-server/internal/handler/deployments.go`:

```go
package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"k8s-ai-ops/mcp-server/internal/k8s"
)

func ListDeploymentsTool() mcp.Tool {
	return mcp.NewTool("list_deployments",
		mcp.WithDescription("List Kubernetes deployments"),
		mcp.WithString("namespace", mcp.Description("Namespace to list deployments from")),
	)
}

func HandleListDeployments(client *k8s.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		namespace := getStringArg(req.Params.Arguments, "namespace")
		deps, err := client.ListDeployments(ctx, namespace)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list deployments: %v", err)), nil
		}
		data, _ := json.Marshal(deps)
		return mcp.NewToolResultRaw(string(data)), nil
	}
}

func RestartDeploymentTool() mcp.Tool {
	return mcp.NewTool("restart_deployment",
		mcp.WithDescription("Restart a Kubernetes deployment by triggering a rollout restart"),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Deployment namespace")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Deployment name")),
	)
}

func HandleRestartDeployment(client *k8s.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments
		namespace, _ := args["namespace"].(string)
		name, _ := args["name"].(string)
		if err := client.RestartDeployment(ctx, namespace, name); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to restart deployment: %v", err)), nil
		}
		return mcp.NewToolResultRaw(fmt.Sprintf(`{"success":true,"message":"restarted deployment %s/%s"}`, namespace, name)), nil
	}
}

func getStringArg(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}
```

- [ ] **Step 9: 重写 main.go — MCP Server + SSE**

Write `mcp-server/cmd/server/main.go`:

```go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"k8s-ai-ops/mcp-server/internal/handler"
	"k8s-ai-ops/mcp-server/internal/k8s"
)

func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8081"
	}
	kubeconfig := os.Getenv("KUBECONFIG")

	k8sClient, err := k8s.NewClient(kubeconfig)
	if err != nil {
		log.Fatalf("level=ERROR component=mcp-server event=k8s_client_create_failed error=%q", err)
	}

	s := mcp.NewServer(
		mcp.WithName("k8s-mcp-server"),
		mcp.WithVersion("1.0.0"),
	)

	s.AddTool(handler.ListNamespacesTool(), handler.HandleListNamespaces(k8sClient))
	s.AddTool(handler.ListPodsTool(), handler.HandleListPods(k8sClient))
	s.AddTool(handler.GetPodTool(), handler.HandleGetPod(k8sClient))
	s.AddTool(handler.GetPodLogsTool(), handler.HandleGetPodLogs(k8sClient))
	s.AddTool(handler.GetPodEventsTool(), handler.HandleGetPodEvents(k8sClient))
	s.AddTool(handler.ListDeploymentsTool(), handler.HandleListDeployments(k8sClient))
	s.AddTool(handler.RestartDeploymentTool(), handler.HandleRestartDeployment(k8sClient))

	mcpServer := server.NewSSEServer(s)
	log.Printf("level=INFO component=mcp-server event=server_start addr=%s protocol=mcp+sse", addr)
	if err := mcpServer.Start(addr); err != nil {
		log.Fatalf("level=ERROR component=mcp-server event=server_exit error=%q", err)
	}
}
```

注意: `server.NewSSEServer` 自动注册 `/sse` 和 `/messages` 路由，同时保留 `/healthz` 可通过 `mcpServer.ServeMux().HandleFunc()` 添加。

- [ ] **Step 10: 验证编译**

```bash
cd e:/k8s-agent/mcp-server && go build ./...
```

- [ ] **Step 11: 运行测试**

```bash
cd e:/k8s-agent/mcp-server && go test ./...
```

- [ ] **Step 12: 删除旧文件**

```bash
rm e:/k8s-agent/mcp-server/internal/mcp/types.go
rm e:/k8s-agent/mcp-server/internal/mcp/registry.go
rm e:/k8s-agent/mcp-server/internal/mcp/registry_test.go
rm e:/k8s-agent/mcp-server/internal/tools/pods.go
rm e:/k8s-agent/mcp-server/internal/tools/pods_test.go
```

- [ ] **Step 13: Commit**

```bash
git add mcp-server/
git commit -m "feat(mcp-server): replace custom REST API with standard MCP protocol
- Use mark3labs/mcp-go for MCP server with SSE transport
- Implement 7 K8s tools: list_namespaces, list_pods, get_pod, get_pod_logs, get_pod_events, list_deployments, restart_deployment
- Replace stub K8s client with real client-go implementation
- Add fake client for testing
- Remove old custom REST types, registry, and pod analysis helpers"
```

---

### Task 3: Agent Server — Eino ReAct 核心

**Files:**
- Modify: `agent-server/go.mod`
- Modify: `agent-server/cmd/server/main.go`
- Create: `agent-server/internal/eino/config.go`
- Create: `agent-server/internal/eino/llm/factory.go`
- Create: `agent-server/internal/eino/llm/factory_test.go`
- Create: `agent-server/internal/eino/mcp/client.go`
- Create: `agent-server/internal/eino/mcp/client_test.go`
- Modify: `agent-server/internal/eino/runner.go`
- Modify: `agent-server/internal/eino/runner_test.go`
- Modify: `agent-server/internal/server/server.go`
- Modify: `agent-server/internal/server/server_test.go`
- Modify: `agent-server/internal/server/mock_runner.go`

- [ ] **Step 1: 添加依赖**

```bash
cd e:/k8s-agent/agent-server && go get github.com/cloudwego/eino-ext/components/tool/mcp@latest github.com/cloudwego/eino-ext/components/model/openai@latest github.com/mark3labs/mcp-go@latest
```

- [ ] **Step 2: 创建 config.go**

Write `agent-server/internal/eino/config.go`:

```go
package eino

import "os"

type Config struct {
	MCPServerURL string
}

func LoadConfig() Config {
	url := os.Getenv("MCP_SERVER_URL")
	if url == "" {
		url = "http://localhost:8081"
	}
	return Config{MCPServerURL: url}
}
```

- [ ] **Step 3: 创建 LLM factory**

Write `agent-server/internal/eino/llm/factory.go`:

```go
package llm

import (
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	agentv1 "k8s-ai-ops/proto/agent/v1"
)

type ChatModel interface {
	Generate(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.Message, error)
	Stream(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error)
}

func NewFromConfig(cfg *agentv1.ModelRuntimeConfig) (ChatModel, error) {
	switch cfg.GetProviderId() {
	case "openai", "anthropic":
		return openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
			BaseURL:  cfg.GetBaseUrl(),
			APIKey:   cfg.GetApiKey(),
			Model:    cfg.GetModelName(),
			ByOpenAI: true,
		})
	default:
		return nil, fmt.Errorf("unsupported provider: %s", cfg.GetProviderId())
	}
}
```

注意: OpenAI 兼容 API 可以处理大多数 provider（包括 Anthropic 兼容端点）。Eino 的 `openai.NewChatModel` 通过 `ByOpenAI: true` 使用 OpenAI 兼容格式。如果后续需要原生 Anthropic adapter，从 `eino-ext/components/model/anthropic` 引入。

- [ ] **Step 4: 创建 LLM factory 测试**

Write `agent-server/internal/eino/llm/factory_test.go`:

```go
package llm

import (
	"testing"

	agentv1 "k8s-ai-ops/proto/agent/v1"
)

func TestNewFromConfigCreatesOpenAIChatModel(t *testing.T) {
	cfg := &agentv1.ModelRuntimeConfig{
		ProviderId: "openai",
		BaseUrl:    "https://api.openai.com/v1",
		ApiKey:     "sk-test",
		ModelName:  "gpt-4",
	}
	_, err := NewFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewFromConfigRejectsUnknownProvider(t *testing.T) {
	cfg := &agentv1.ModelRuntimeConfig{
		ProviderId: "unknown",
	}
	_, err := NewFromConfig(cfg)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}
```

- [ ] **Step 5: 创建 MCP 客户端**

Write `agent-server/internal/eino/mcp/client.go`:

```go
package mcp

import (
	"context"
	"fmt"

	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
)

type Client struct {
	mcpClient *client.SSEMCPClient
	tools     []tool.BaseTool
}

func NewClient(ctx context.Context, serverURL string) (*Client, error) {
	mcpClient, err := client.NewSSEMCPClient(serverURL)
	if err != nil {
		return nil, fmt.Errorf("mcp sse client: %w", err)
	}
	if err := mcpClient.Start(ctx); err != nil {
		return nil, fmt.Errorf("mcp start: %w", err)
	}

	tools, err := einomcp.GetTools(ctx, &einomcp.Config{
		Cli:          mcpClient,
		ToolNameList: nil,
	})
	if err != nil {
		return nil, fmt.Errorf("mcp get tools: %w", err)
	}

	return &Client{
		mcpClient: mcpClient,
		tools:     tools,
	}, nil
}

func (c *Client) Tools() []tool.BaseTool {
	return c.tools
}

func (c *Client) Close() {
	if c.mcpClient != nil {
		c.mcpClient.Close()
	}
}
```

- [ ] **Step 6: 创建 MCP 客户端测试**

Write `agent-server/internal/eino/mcp/client_test.go`:

```go
package mcp

import (
	"testing"
)

func TestNewClientRequiresValidURL(t *testing.T) {
	_, err := NewClient(t.Context(), "http://localhost:19999")
	if err == nil {
		t.Fatal("expected error for unreachable server")
	}
}
```

- [ ] **Step 7: 重写 runner.go — Eino ReAct + RunStream**

Write `agent-server/internal/eino/runner.go`:

```go
package eino

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	agentv1 "k8s-ai-ops/proto/agent/v1"

	"k8s-ai-ops/agent-server/internal/eino/llm"
	einomcp "k8s-ai-ops/agent-server/internal/eino/mcp"
)

type Runner struct {
	mcpClient *einomcp.Client
}

func NewRunner(ctx context.Context, mcpServerURL string) (*Runner, error) {
	mcpClient, err := einomcp.NewClient(ctx, mcpServerURL)
	if err != nil {
		return nil, fmt.Errorf("mcp client init: %w", err)
	}
	return &Runner{mcpClient: mcpClient}, nil
}

func (r *Runner) Close() {
	if r.mcpClient != nil {
		r.mcpClient.Close()
	}
}

func (r *Runner) RunStream(ctx context.Context, request *agentv1.AgentRunRequest, sender StreamSender) error {
	chatModel, err := llm.NewFromConfig(request.GetModel())
	if err != nil {
		return fmt.Errorf("llm factory: %w", err)
	}

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		Model:       chatModel,
		ToolsConfig: compose.ToolsNodeConfig{Tools: r.mcpClient.Tools()},
		MaxStep:     15,
	})
	if err != nil {
		return fmt.Errorf("react agent: %w", err)
	}

	messages := buildMessages(request)

	sr, err := agent.Stream(ctx, messages)
	if err != nil {
		return fmt.Errorf("agent stream: %w", err)
	}

	for {
		msg, err := sr.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			sender.Send(&agentv1.StreamEvent{
				EventId:   uuid.New().String(),
				Timestamp: time.Now().UnixMilli(),
				Event: &agentv1.StreamEvent_Error{
					Error: &agentv1.ErrorEvent{
						Code:    "STREAM_ERROR",
						Message: err.Error(),
					},
				},
			})
			continue
		}

		switch msg.Role {
		case schema.Assistant:
			sender.Send(&agentv1.StreamEvent{
				EventId:   uuid.New().String(),
				Timestamp: time.Now().UnixMilli(),
				Event: &agentv1.StreamEvent_Thinking{
					Thinking: &agentv1.ThinkingEvent{
						Content: msg.Content,
					},
				},
			})
		case schema.Tool:
			sender.Send(&agentv1.StreamEvent{
				EventId:   uuid.New().String(),
				Timestamp: time.Now().UnixMilli(),
				Event: &agentv1.StreamEvent_ToolCall{
					ToolCall: &agentv1.ToolCallEvent{
						ToolName:       msg.ToolCall.Function.Name,
						ArgumentsJson:  msg.ToolCall.Function.Arguments,
					},
				},
			})
		}
	}

	sender.Send(&agentv1.StreamEvent{
		EventId:   uuid.New().String(),
		Timestamp: time.Now().UnixMilli(),
		Event: &agentv1.StreamEvent_Complete{
			Complete: &agentv1.CompleteEvent{
				Summary:   "agent loop completed",
				Resources: nil,
			},
		},
	})

	return nil
}

type StreamSender interface {
	Send(*agentv1.StreamEvent) error
}

func buildMessages(request *agentv1.AgentRunRequest) []*schema.Message {
	messages := []*schema.Message{
		{Role: schema.System, Content: systemContext(request)},
	}
	for _, msg := range request.GetMessages() {
		role := schema.RoleType(msg.GetRole())
		if role == "" {
			role = schema.User
		}
		messages = append(messages, &schema.Message{
			Role:    role,
			Content: msg.GetContent(),
		})
	}
	if len(request.GetMessages()) == 0 && request.GetMessage() != "" {
		messages = append(messages, schema.UserMessage(request.GetMessage()))
	}
	return messages
}

func systemContext(request *agentv1.AgentRunRequest) string {
	var b strings.Builder
	b.WriteString("You are a Kubernetes AI operations assistant. ")
	b.WriteString("You have access to Kubernetes tools via MCP. ")
	b.WriteString("Use the tools to fetch real cluster data when the user asks questions about resources. ")
	b.WriteString("Analyze the returned data and provide helpful insights. ")
	if user := request.GetRuntimeContext().GetCurrentUser(); user != "" {
		b.WriteString("Current user: ")
		b.WriteString(user)
		b.WriteString(". ")
	}
	if namespaces := request.GetRuntimeContext().GetAllowedNamespaces(); len(namespaces) > 0 {
		b.WriteString("Allowed namespaces: ")
		b.WriteString(strings.Join(namespaces, ", "))
		b.WriteString(". ")
	}
	return b.String()
}
```

- [ ] **Step 8: 重写 runner_test.go**

Write `agent-server/internal/eino/runner_test.go`:

```go
package eino

import (
	"testing"

	agentv1 "k8s-ai-ops/proto/agent/v1"
)

func TestBuildMessagesAppendsSystemContext(t *testing.T) {
	request := &agentv1.AgentRunRequest{
		Messages: []*agentv1.Message{
			{Role: "user", Content: "check dev namespace"},
		},
		RuntimeContext: &agentv1.RuntimeContext{
			CurrentUser:       "operator-a",
			AllowedNamespaces: []string{"dev"},
		},
	}
	messages := buildMessages(request)
	if len(messages) != 2 {
		t.Fatalf("expected system + 1 user message, got %d", len(messages))
	}
	if messages[0].Role != "system" {
		t.Fatalf("expected first message to be system, got %s", messages[0].Role)
	}
	if messages[1].Role != "user" {
		t.Fatalf("expected second message to be user, got %s", messages[1].Role)
	}
}

func TestBuildMessagesHandlesEmptyHistoryWithMessageField(t *testing.T) {
	request := &agentv1.AgentRunRequest{
		Message: "check dev namespace",
		RuntimeContext: &agentv1.RuntimeContext{
			CurrentUser: "operator-a",
		},
	}
	messages := buildMessages(request)
	if len(messages) != 2 {
		t.Fatalf("expected system + 1 user message, got %d", len(messages))
	}
}
```

- [ ] **Step 9: 更新 server/server.go**

`agentv1.AgentService_RunStreamServer` 嵌入了 `grpc.ServerStream`，它已有 `Send(*agentv1.StreamEvent) error` 方法。我们在 eino 包中定义 `StreamSender` 接口，server 包通过 import 引用：

Write `agent-server/internal/server/server.go`:

```go
package server

import (
	"context"

	agentv1 "k8s-ai-ops/proto/agent/v1"
	agent "k8s-ai-ops/agent-server/internal/eino"
)

type Runner interface {
	RunStream(context.Context, *agentv1.AgentRunRequest, agent.StreamSender) error
}

type AgentService struct {
	agentv1.UnimplementedAgentServiceServer
	runner Runner
}

func NewAgentService(runner Runner) *AgentService {
	return &AgentService{runner: runner}
}

func (s *AgentService) RunStream(request *agentv1.AgentRunRequest, stream agentv1.AgentService_RunStreamServer) error {
	return s.runner.RunStream(stream.Context(), request, stream)
}
```

`agentv1.AgentService_RunStreamServer` 的 `Send(*agentv1.StreamEvent) error` 方法签名自然满足 `agent.StreamSender` 接口，无需额外适配。

- [ ] **Step 10: 更新 mock_runner.go**

Write `agent-server/internal/server/mock_runner.go`:

```go
package server

import (
	"context"

	agentv1 "k8s-ai-ops/proto/agent/v1"
)

type MockRunner struct {
	events []*agentv1.StreamEvent
}

func NewMockRunner() *MockRunner {
	return &MockRunner{}
}

func (r *MockRunner) RunStream(_ context.Context, _ *agentv1.AgentRunRequest, sender StreamSender) error {
	for _, event := range r.events {
		if err := sender.Send(event); err != nil {
			return err
		}
	}
	return nil
}

func (r *MockRunner) SetEvents(events []*agentv1.StreamEvent) {
	r.events = events
}
```

- [ ] **Step 11: 更新 server_test.go**

Write `agent-server/internal/server/server_test.go`:

```go
package server

import (
	"context"
	"io"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	agentv1 "k8s-ai-ops/proto/agent/v1"
)

const testBufSize = 1024 * 1024

func TestAgentServiceRunStreamSendsEvents(t *testing.T) {
	runner := NewMockRunner()
	runner.SetEvents([]*agentv1.StreamEvent{
		{
			EventId: "1",
			Event: &agentv1.StreamEvent_Thinking{
				Thinking: &agentv1.ThinkingEvent{Content: "thinking..."},
			},
		},
		{
			EventId: "2",
			Event: &agentv1.StreamEvent_Complete{
				Complete: &agentv1.CompleteEvent{Summary: "done"},
			},
		},
	})

	service := NewAgentService(runner)
	listener := bufconn.Listen(testBufSize)
	grpcServer := grpc.NewServer()
	agentv1.RegisterAgentServiceServer(grpcServer, service)
	go grpcServer.Serve(listener)
	defer grpcServer.Stop()

	conn, _ := grpc.DialContext(
		context.Background(), "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithInsecure(),
	)
	defer conn.Close()

	client := agentv1.NewAgentServiceClient(conn)
	stream, err := client.RunStream(context.Background(), &agentv1.AgentRunRequest{
		Message: "check dev",
		RuntimeContext: &agentv1.RuntimeContext{
			CurrentUser: "operator-a",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var events []*agentv1.StreamEvent
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, event)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}
```

- [ ] **Step 12: 更新 cmd/server/main.go**

Write `agent-server/cmd/server/main.go`:

```go
package main

import (
	"context"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	einorunner "k8s-ai-ops/agent-server/internal/eino"
	"k8s-ai-ops/agent-server/internal/server"
	agentv1 "k8s-ai-ops/proto/agent/v1"
)

func main() {
	addr := os.Getenv("GRPC_ADDR")
	if addr == "" {
		addr = ":8082"
	}
	cfg := einorunner.LoadConfig()

	ctx := context.Background()
	runner, err := einorunner.NewRunner(ctx, cfg.MCPServerURL)
	if err != nil {
		log.Fatalf("level=ERROR component=agent-server event=mcp_connect_failed url=%s error=%q", cfg.MCPServerURL, err)
	}
	defer runner.Close()

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("level=ERROR component=agent-server event=listen_failed addr=%s error=%q", addr, err)
	}

	grpcServer := grpc.NewServer()
	agentv1.RegisterAgentServiceServer(grpcServer, server.NewAgentService(runner))
	log.Printf("level=INFO component=agent-server event=server_start addr=%s mcp_url=%s protocol=grpc", addr, cfg.MCPServerURL)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("level=ERROR component=agent-server event=server_exit error=%q", err)
	}
}
```

- [ ] **Step 13: 验证编译**

```bash
cd e:/k8s-agent/agent-server && go build ./...
```

- [ ] **Step 14: 运行测试**

```bash
cd e:/k8s-agent/agent-server && go test ./...
```

- [ ] **Step 15: Commit**

```bash
git add agent-server/
git commit -m "feat(agent-server): implement Eino ReAct agent with MCP integration
- Replace mock runner with Eino ReAct agent + RunStream
- Add LLM factory for dynamic ChatModel creation from ModelRuntimeConfig
- Add MCP SSE client for tool discovery via eino-ext mcp
- Add config.go for MCP_SERVER_URL env reading
- Update gRPC server to register RunStream handler"
```

---

### Task 4: 后端 — Streaming gRPC 客户端 + SSE 中转

**Files:**
- Modify: `backend/internal/agent/client.go`
- Modify: `backend/internal/agent/client_test.go`
- Modify: `backend/internal/http/router.go`
- Modify: `backend/internal/config/config.go`

- [ ] **Step 1: 更新 agent client — 改为 streaming**

Write `backend/internal/agent/client.go`:

```go
package agent

import (
	"context"
	"io"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	agentv1 "k8s-ai-ops/proto/agent/v1"
)

type StreamEvent = agentv1.StreamEvent

type Client interface {
	RunStream(context.Context, *agentv1.AgentRunRequest) (StreamReader, error)
}

type StreamReader interface {
	Recv() (*agentv1.StreamEvent, error)
}

type GRPCClient struct {
	client agentv1.AgentServiceClient
}

type grpcStreamReader struct {
	stream agentv1.AgentService_RunStreamClient
}

func (r *grpcStreamReader) Recv() (*agentv1.StreamEvent, error) {
	event, err := r.stream.Recv()
	if err == io.EOF {
		return nil, err
	}
	return event, err
}

func NewGRPCClient(client agentv1.AgentServiceClient) *GRPCClient {
	return &GRPCClient{client: client}
}

func Dial(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return grpc.DialContext(dialCtx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
}

func (c *GRPCClient) RunStream(ctx context.Context, request *agentv1.AgentRunRequest) (StreamReader, error) {
	stream, err := c.client.RunStream(ctx, request)
	if err != nil {
		return nil, err
	}
	return &grpcStreamReader{stream: stream}, nil
}
```

- [ ] **Step 2: 更新 agent client 测试**

Write `backend/internal/agent/client_test.go`:

```go
package agent

import (
	"io"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	agentv1 "k8s-ai-ops/proto/agent/v1"
)

const testBufSize = 1024 * 1024

type streamingAgentServer struct {
	agentv1.UnimplementedAgentServiceServer
	events []*agentv1.StreamEvent
}

func (s *streamingAgentServer) RunStream(request *agentv1.AgentRunRequest, stream agentv1.AgentService_RunStreamServer) error {
	for _, event := range s.events {
		if err := stream.Send(event); err != nil {
			return err
		}
	}
	return nil
}

func TestClientRunStreamReceivesEvents(t *testing.T) {
	listener := bufconn.Listen(testBufSize)
	grpcServer := grpc.NewServer()
	server := &streamingAgentServer{events: []*agentv1.StreamEvent{
		{EventId: "1", Event: &agentv1.StreamEvent_Thinking{Thinking: &agentv1.ThinkingEvent{Content: "hello"}}},
		{EventId: "2", Event: &agentv1.StreamEvent_Complete{Complete: &agentv1.CompleteEvent{Summary: "done"}}},
	}}
	agentv1.RegisterAgentServiceServer(grpcServer, server)
	go grpcServer.Serve(listener)
	defer grpcServer.Stop()

	conn, err := grpc.DialContext(
		t.Context(), "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	client := NewGRPCClient(agentv1.NewAgentServiceClient(conn))
	reader, err := client.RunStream(t.Context(), &agentv1.AgentRunRequest{
		Message: "check dev",
	})
	if err != nil {
		t.Fatal(err)
	}

	var count int
	for {
		_, err := reader.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count != 2 {
		t.Fatalf("expected 2 events, got %d", count)
	}
}
```

- [ ] **Step 3: 更新 createChatMessage handler — 改为 SSE 流式**

Modify `backend/internal/http/router.go`:

在 `ServeHTTP` 方法中，路由匹配保持不变。在 `createChatMessage` 方法需要修改 — 将单次 unary 调用改为 SSE 流式输出。

在 `router.go` 文件末尾，将 `createChatMessage` 方法替换为:

```go
func (s *Server) createChatMessage(w http.ResponseWriter, r *http.Request) {
	var request chatMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request body must be valid JSON.")
		return
	}
	if !s.isModelAllowed(request.ModelID) {
		s.audit("operator.chat.message.create", "chat_message", "demo-message", "", "", "", false, "model not allowed")
		writeError(w, http.StatusForbidden, "LLM_MODEL_NOT_ALLOWED", "The selected model is not assigned to the current user.")
		return
	}

	if s.agentClient == nil {
		s.agentClient = localAgentClient{pods: s.pods}
	}

	reader, err := s.agentClient.RunStream(r.Context(), s.buildAgentRunRequest(request))
	if err != nil {
		s.audit("operator.chat.message.create", "chat_message", "demo-message", "", "", "", false, "agent server unavailable")
		writeError(w, http.StatusBadGateway, "AGENT_SERVER_UNAVAILABLE", "Agent server is unavailable.")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Streaming not supported.")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {
		event, err := reader.Recv()
		if err != nil {
			break
		}
		data, _ := json.Marshal(event)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	s.audit("operator.chat.message.create", "chat_message", "demo-message", "", "", "", true, "stream completed")
}
```

注意: 需要在 router.go 顶部添加 `"fmt"` 导入。

同时在 `localAgentClient` 中需要将 `Run` 替换为 `RunStream`:

```go
type localAgentClient struct {
	pods []chat.AbnormalPod
}

type eventStreamReader struct {
	events []*agentv1.StreamEvent
	pos    int
}

func (r *eventStreamReader) Recv() (*agentv1.StreamEvent, error) {
	if r.pos >= len(r.events) {
		return nil, io.EOF
	}
	event := r.events[r.pos]
	r.pos++
	return event, nil
}

func (c localAgentClient) RunStream(_ context.Context, request *agentv1.AgentRunRequest) (agentclient.StreamReader, error) {
	var resources []*agentv1.ResourceResult
	for _, pod := range c.pods {
		resources = append(resources, &agentv1.ResourceResult{
			Kind:      "Pod",
			Namespace: pod.Namespace,
			Name:      pod.Name,
			Status:    pod.Reason,
		})
	}
	resourcesJSON, _ := json.Marshal(resources)
	return &eventStreamReader{events: []*agentv1.StreamEvent{
		{
			EventId: "1",
			Event: &agentv1.StreamEvent_Thinking{
				Thinking: &agentv1.ThinkingEvent{Content: "正在检查 dev 命名空间..."},
			},
		},
		{
			EventId: "2",
			Event: &agentv1.StreamEvent_ToolCall{
				ToolCall: &agentv1.ToolCallEvent{
					ToolName:      "list_pods",
					ArgumentsJson: `{"namespace":"dev"}`,
				},
			},
		},
		{
			EventId: "3",
			Event: &agentv1.StreamEvent_ToolResult{
				ToolResult: &agentv1.ToolResultEvent{
					ToolName:  "list_pods",
					Success:   true,
					ResultJson: string(resourcesJSON),
				},
			},
		},
		{
			EventId: "4",
			Event: &agentv1.StreamEvent_Complete{
				Complete: &agentv1.CompleteEvent{
					Summary:   "dev namespace 中有 2 个异常 Pod。",
					Resources: resources,
				},
			},
		},
	}}, nil
}
```

注意: 需要添加 `"io"` 导入，以及 `agentclient "k8s-ai-ops/backend/internal/agent"` 的导入。

- [ ] **Step 4: 删除 config.MCPServerURL**

Remove the `MCPServerURL` field from `backend/internal/config/config.go`:

```go
type Config struct {
	HTTPAddr           string
	KeycloakIssuer     string
	DatabaseURL        string
	RedisAddr          string
	AgentServerAddr    string
	StoreDriver        string
	CacheDriver        string
	K8SRBACSyncEnabled bool
	Kubeconfig         string
}

func Load() Config {
	return Config{
		HTTPAddr:           env("HTTP_ADDR", ":8080"),
		KeycloakIssuer:     env("KEYCLOAK_ISSUER", "http://keycloak:8080/realms/k8s-ai"),
		DatabaseURL:        env("DATABASE_URL", "postgres://k8s_ai:k8s_ai@postgresql:5432/k8s_ai?sslmode=disable"),
		RedisAddr:          env("REDIS_ADDR", "redis:6379"),
		AgentServerAddr:    env("AGENT_SERVER_ADDR", "agent-server:8082"),
		StoreDriver:        env("STORE_DRIVER", "memory"),
		CacheDriver:        env("CACHE_DRIVER", "none"),
		K8SRBACSyncEnabled: envBool("K8S_RBAC_SYNC_ENABLED", false),
		Kubeconfig:         env("KUBECONFIG", ""),
	}
}
```

注意: 从 `Load()` 返回值中删除 `MCPServerURL` 一行。

- [ ] **Step 5: 验证编译**

```bash
cd e:/k8s-agent/backend && go build ./...
```

- [ ] **Step 6: 运行测试**

```bash
cd e:/k8s-agent/backend && go test ./...
```

- [ ] **Step 7: Commit**

```bash
git add backend/
git commit -m "feat(backend): upgrade to streaming gRPC agent client with SSE relay
- Replace agent Client.Run() with RunStream() returning StreamReader
- Change createChatMessage handler to SSE streaming output
- Remove MCPServerURL config (backend no longer connects to MCP directly)
- Add localAgentClient.RunStream() fallback for demo mode"
```

---

### Task 5: 端到端集成验证

- [ ] **Step 1: 全量构建**

```bash
cd e:/k8s-agent && go build ./agent-server/... && go build ./mcp-server/... && go build ./backend/...
```

- [ ] **Step 2: 全量测试**

```bash
cd e:/k8s-agent && go test ./mcp-server/... ./agent-server/... ./backend/... ./proto/...
```

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "chore: verify all services build and pass tests after streaming migration"
```
