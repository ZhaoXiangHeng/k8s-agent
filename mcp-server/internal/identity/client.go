// Package identity 封装与 Backend Identity Service 的 gRPC 通信，
// 根据用户 ID 查询对应的 K8s ServiceAccount 凭证（Token、API Server 地址、CA 证书等）。
package identity

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	identityv1 "k8s-ai-ops/proto/identity/v1"
)

var log = logrus.WithField("component", "mcp-server/identity")

// Client 是 Identity Service 的 gRPC 客户端。
type Client struct {
	addr string
}

// NewClient 创建 Identity Service 客户端配置，不在启动阶段连接 Backend。
func NewClient(ctx context.Context, addr string) (*Client, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return &Client{addr: addr}, nil
}

// GetServiceAccount 根据用户 ID 查询对应的 K8s ServiceAccount 信息，
// 返回 Token、API Server 地址、默认 Namespace 和 CA 证书。
func (c *Client) GetServiceAccount(ctx context.Context, userID string) (*identityv1.GetServiceAccountResponse, error) {
	// 对 Backend 使用短连接，避免 mcp-server 启动时被服务依赖环阻塞。
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(dialCtx, c.addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"operation": "GetServiceAccount",
			"user_id":   userID,
			"addr":      c.addr,
		}).Error("failed to dial identity service")
		return nil, err
	}
	defer conn.Close()

	resp, err := identityv1.NewIdentityServiceClient(conn).GetServiceAccount(ctx, &identityv1.GetServiceAccountRequest{
		UserId: userID,
	})
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"operation": "GetServiceAccount",
			"user_id":   userID,
		}).Error("failed to get service account from identity service")
		return nil, err
	}
	return resp, nil
}
