package identity

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	identityv1 "k8s-ai-ops/proto/identity/v1"
)

type Client struct {
	grpc identityv1.IdentityServiceClient
}

func NewClient(ctx context.Context, addr string) (*Client, error) {
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(dialCtx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}
	return &Client{grpc: identityv1.NewIdentityServiceClient(conn)}, nil
}

func (c *Client) GetServiceAccount(ctx context.Context, userID string) (*identityv1.GetServiceAccountResponse, error) {
	return c.grpc.GetServiceAccount(ctx, &identityv1.GetServiceAccountRequest{
		UserId: userID,
	})
}
