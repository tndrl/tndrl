package control

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	latisv1 "github.com/shanemcd/latis/gen/go/latis/v1"
)

func TestPing(t *testing.T) {
	state := NewState("test")
	server := NewServer(state, nil)

	now := time.Now().UnixNano()
	req := &latisv1.PingRequest{Timestamp: now}

	resp, err := server.Ping(context.Background(), req)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	if resp.PingTimestamp != now {
		t.Errorf("expected ping timestamp %d, got %d", now, resp.PingTimestamp)
	}
	if resp.PongTimestamp <= now {
		t.Errorf("pong timestamp should be after ping timestamp")
	}
}

func TestGetStatus(t *testing.T) {
	state := NewState("spiffe://latis/node/test-node")
	state.SetReady()
	state.SetMetadata("version", "1.0.0")

	server := NewServer(state, nil)

	resp, err := server.GetStatus(context.Background(), &latisv1.GetStatusRequest{})
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if resp.Identity != "spiffe://latis/node/test-node" {
		t.Errorf("expected identity 'spiffe://latis/node/test-node', got %v", resp.Identity)
	}
	if resp.State != latisv1.NodeState_NODE_STATE_READY {
		t.Errorf("expected READY state, got %v", resp.State)
	}
	if resp.ActiveTasks != 0 {
		t.Errorf("expected 0 active tasks, got %v", resp.ActiveTasks)
	}
	if resp.Metadata["version"] != "1.0.0" {
		t.Errorf("expected metadata version '1.0.0', got %v", resp.Metadata["version"])
	}
}

func TestShutdown(t *testing.T) {
	state := NewState("test")
	state.SetReady()

	var shutdownCalled atomic.Bool
	var shutdownGraceful bool
	var shutdownTimeout time.Duration
	var shutdownReason string

	shutdownFn := func(graceful bool, timeout time.Duration, reason string) {
		shutdownGraceful = graceful
		shutdownTimeout = timeout
		shutdownReason = reason
		shutdownCalled.Store(true)
	}

	server := NewServer(state, shutdownFn)

	req := &latisv1.ShutdownRequest{
		Graceful:       true,
		TimeoutSeconds: 30,
		Reason:         "test shutdown",
	}

	resp, err := server.Shutdown(context.Background(), req)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	if !resp.Accepted {
		t.Errorf("expected shutdown to be accepted")
	}

	// Wait for goroutine to execute
	time.Sleep(10 * time.Millisecond)

	if !shutdownCalled.Load() {
		t.Errorf("expected shutdown function to be called")
	}
	if !shutdownGraceful {
		t.Errorf("expected graceful shutdown")
	}
	if shutdownTimeout != 30*time.Second {
		t.Errorf("expected 30s timeout, got %v", shutdownTimeout)
	}
	if shutdownReason != "test shutdown" {
		t.Errorf("expected reason 'test shutdown', got %v", shutdownReason)
	}
}

func TestShutdownAlreadyDraining(t *testing.T) {
	state := NewState("test")
	state.SetDraining()

	server := NewServer(state, nil)

	resp, err := server.Shutdown(context.Background(), &latisv1.ShutdownRequest{})
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	if resp.Accepted {
		t.Errorf("expected shutdown to be rejected when already draining")
	}
	if resp.RejectionReason == "" {
		t.Errorf("expected rejection reason")
	}
}

func TestShutdownAlreadyStopped(t *testing.T) {
	state := NewState("test")
	state.SetStopped()

	server := NewServer(state, nil)

	resp, err := server.Shutdown(context.Background(), &latisv1.ShutdownRequest{})
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	if resp.Accepted {
		t.Errorf("expected shutdown to be rejected when already stopped")
	}
}
