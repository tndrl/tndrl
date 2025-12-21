package a2aexec

import (
	"testing"

	"google.golang.org/grpc"
)

func TestRegisterWithGRPC(t *testing.T) {
	server := grpc.NewServer()

	cfg := &ServerConfig{
		Executor:  NewExecutor(),
		AgentCard: DefaultAgentCard("test-unit", "A test unit", "localhost:4433"),
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

func TestDefaultAgentCard(t *testing.T) {
	card := DefaultAgentCard("my-unit", "My unit description", "localhost:4433")

	if card.Name != "my-unit" {
		t.Errorf("expected name %q, got %q", "my-unit", card.Name)
	}

	if card.Description != "My unit description" {
		t.Errorf("expected description %q, got %q", "My unit description", card.Description)
	}

	if card.URL != "localhost:4433" {
		t.Errorf("expected URL %q, got %q", "localhost:4433", card.URL)
	}

	if len(card.Skills) == 0 {
		t.Error("expected at least one skill")
	}
}
