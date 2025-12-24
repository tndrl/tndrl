package a2aexec

import (
	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2agrpc"
	"github.com/a2aproject/a2a-go/a2asrv"
	"google.golang.org/grpc"
)

// ServerConfig holds configuration for an A2A-compatible node server.
type ServerConfig struct {
	// Executor handles incoming A2A messages.
	Executor *Executor

	// AgentCard describes this agent's capabilities.
	AgentCard *a2a.AgentCard
}

// RegisterWithGRPC registers the A2A service with a gRPC server.
// This wires up the executor with the a2a-go request handler and gRPC transport.
func RegisterWithGRPC(server *grpc.Server, cfg *ServerConfig) {
	// Create the transport-agnostic request handler
	var opts []a2asrv.RequestHandlerOption
	if cfg.AgentCard != nil {
		opts = append(opts, a2asrv.WithExtendedAgentCard(cfg.AgentCard))
	}
	requestHandler := a2asrv.NewHandler(cfg.Executor, opts...)

	// Create the gRPC transport handler
	grpcHandler := a2agrpc.NewHandler(requestHandler)

	// Register with the gRPC server
	grpcHandler.RegisterWith(server)
}

