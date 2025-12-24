package a2aexec

import (
	"testing"

	"github.com/a2aproject/a2a-go/a2a"
	"google.golang.org/grpc"
)

func TestRegisterWithGRPC(t *testing.T) {
	server := grpc.NewServer()

	cfg := &ServerConfig{
		Executor: NewExecutor(),
		AgentCard: &a2a.AgentCard{
			Name:               "test-node",
			Description:        "A test node",
			URL:                "localhost:4433",
			PreferredTransport: a2a.TransportProtocolGRPC,
			DefaultInputModes:  []string{"text"},
			DefaultOutputModes: []string{"text"},
			Capabilities: a2a.AgentCapabilities{
				Streaming: true,
			},
		},
	}

	// Should not panic
	RegisterWithGRPC(server, cfg)

	// Verify services were registered
	info := server.GetServiceInfo()
	if len(info) == 0 {
		t.Error("expected at least one service to be registered")
	}

	// Look for A2A service
	found := false
	for name := range info {
		t.Logf("registered service: %s", name)
		if name == "a2a.v1.A2AService" {
			found = true
		}
	}

	if !found {
		t.Error("expected a2a.v1.A2AService to be registered")
	}
}

func TestRegisterWithGRPC_NilAgentCard(t *testing.T) {
	server := grpc.NewServer()

	cfg := &ServerConfig{
		Executor:  NewExecutor(),
		AgentCard: nil, // No agent card provided
	}

	// Should not panic with nil agent card
	RegisterWithGRPC(server, cfg)

	// Verify services were registered
	info := server.GetServiceInfo()
	if len(info) == 0 {
		t.Error("expected at least one service to be registered")
	}
}
