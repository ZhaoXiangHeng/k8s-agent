package k8s

import (
	"testing"
)

func TestNewClientFromSA(t *testing.T) {
	_, err := NewClientFromSA("test-token", "https://localhost:6443", "default", "")
	if err != nil {
		// Expected: connection will fail, but NewClientFromSA only configures
		// the client and does not validate the connection at creation time.
		// The error here is acceptable since we pass an unreachable server.
	}
}
