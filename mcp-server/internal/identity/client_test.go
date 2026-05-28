package identity

import (
	"context"
	"testing"
	"time"
)

func TestNewClientDoesNotDialBackendDuringConstruction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	client, err := NewClient(ctx, "127.0.0.1:1")
	if err != nil {
		t.Fatalf("NewClient should not connect during construction: %v", err)
	}
	if client == nil {
		t.Fatal("NewClient returned nil client")
	}
}
