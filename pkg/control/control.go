package control

import (
	"context"
	"log/slog"
	"time"

	latisv1 "github.com/shanemcd/latis/gen/go/latis/v1"
)

// ShutdownFunc is called when a shutdown is requested via the Control RPC.
type ShutdownFunc func(graceful bool, timeout time.Duration, reason string)

// Server implements latisv1.ControlServiceServer.
type Server struct {
	latisv1.UnimplementedControlServiceServer

	state    *State
	shutdown ShutdownFunc
}

// NewServer creates a new ControlService server.
func NewServer(state *State, shutdownFn ShutdownFunc) *Server {
	return &Server{
		state:    state,
		shutdown: shutdownFn,
	}
}

// Ping implements health check with latency measurement.
func (s *Server) Ping(ctx context.Context, req *latisv1.PingRequest) (*latisv1.PingResponse, error) {
	slog.Debug("ping received", "timestamp", req.Timestamp)
	return &latisv1.PingResponse{
		PingTimestamp: req.Timestamp,
		PongTimestamp: time.Now().UnixNano(),
	}, nil
}

// GetStatus returns the current status of the node.
func (s *Server) GetStatus(ctx context.Context, req *latisv1.GetStatusRequest) (*latisv1.GetStatusResponse, error) {
	state := s.state.GetState()
	slog.Debug("status requested", "state", state.String())
	return &latisv1.GetStatusResponse{
		Identity:      s.state.GetIdentity(),
		State:         state,
		UptimeSeconds: s.state.GetUptime(),
		ActiveTasks:   s.state.GetActiveTasks(),
		Metadata:      s.state.GetMetadata(),
	}, nil
}

// Shutdown requests graceful termination of the node.
func (s *Server) Shutdown(ctx context.Context, req *latisv1.ShutdownRequest) (*latisv1.ShutdownResponse, error) {
	currentState := s.state.GetState()
	slog.Info("shutdown RPC received", "graceful", req.Graceful, "timeout", req.TimeoutSeconds, "reason", req.Reason, "current_state", currentState.String())

	if currentState == latisv1.NodeState_NODE_STATE_DRAINING ||
		currentState == latisv1.NodeState_NODE_STATE_STOPPED {
		slog.Warn("shutdown rejected", "reason", "already shutting down", "state", currentState.String())
		return &latisv1.ShutdownResponse{
			Accepted:        false,
			RejectionReason: "node is already shutting down",
		}, nil
	}

	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	go s.shutdown(req.Graceful, timeout, req.Reason)

	return &latisv1.ShutdownResponse{
		Accepted: true,
	}, nil
}
