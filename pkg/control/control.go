package control

import (
	"context"
	"log/slog"
	"time"

	tndrlv1 "github.com/shanemcd/tndrl/gen/go/tndrl/v1"
)

// ShutdownFunc is called when a shutdown is requested via the Control RPC.
type ShutdownFunc func(graceful bool, timeout time.Duration, reason string)

// Server implements tndrlv1.ControlServiceServer.
type Server struct {
	tndrlv1.UnimplementedControlServiceServer

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
func (s *Server) Ping(ctx context.Context, req *tndrlv1.PingRequest) (*tndrlv1.PingResponse, error) {
	slog.Debug("ping received", "timestamp", req.Timestamp)
	return &tndrlv1.PingResponse{
		PingTimestamp: req.Timestamp,
		PongTimestamp: time.Now().UnixNano(),
	}, nil
}

// GetStatus returns the current status of the node.
func (s *Server) GetStatus(ctx context.Context, req *tndrlv1.GetStatusRequest) (*tndrlv1.GetStatusResponse, error) {
	state := s.state.GetState()
	slog.Debug("status requested", "state", state.String())
	return &tndrlv1.GetStatusResponse{
		Identity:      s.state.GetIdentity(),
		State:         state,
		UptimeSeconds: s.state.GetUptime(),
		ActiveTasks:   s.state.GetActiveTasks(),
		Metadata:      s.state.GetMetadata(),
	}, nil
}

// Shutdown requests graceful termination of the node.
func (s *Server) Shutdown(ctx context.Context, req *tndrlv1.ShutdownRequest) (*tndrlv1.ShutdownResponse, error) {
	currentState := s.state.GetState()
	slog.Info("shutdown RPC received", "graceful", req.Graceful, "timeout", req.TimeoutSeconds, "reason", req.Reason, "current_state", currentState.String())

	if currentState == tndrlv1.NodeState_NODE_STATE_DRAINING ||
		currentState == tndrlv1.NodeState_NODE_STATE_STOPPED {
		slog.Warn("shutdown rejected", "reason", "already shutting down", "state", currentState.String())
		return &tndrlv1.ShutdownResponse{
			Accepted:        false,
			RejectionReason: "node is already shutting down",
		}, nil
	}

	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	go s.shutdown(req.Graceful, timeout, req.Reason)

	return &tndrlv1.ShutdownResponse{
		Accepted: true,
	}, nil
}
