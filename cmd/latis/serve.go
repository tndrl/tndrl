package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/google/uuid"
	"google.golang.org/grpc"

	latisv1 "github.com/shanemcd/latis/gen/go/latis/v1"
	"github.com/shanemcd/latis/pkg/a2aexec"
	"github.com/shanemcd/latis/pkg/control"
	"github.com/shanemcd/latis/pkg/llm"
	"github.com/shanemcd/latis/pkg/pki"
	quictransport "github.com/shanemcd/latis/pkg/transport/quic"
)

// ServeCmd runs latis as a daemon, listening for incoming connections.
type ServeCmd struct{}

// Run executes the serve command.
func (c *ServeCmd) Run(cli *CLI) error {
	slog.Info("starting", "addr", cli.Server.Addr)

	tlsConfig, err := setupServerTLS(cli)
	if err != nil {
		return fmt.Errorf("setup TLS: %w", err)
	}

	// Create multiplexed QUIC listener
	listener, err := quictransport.ListenMux(cli.Server.Addr, tlsConfig, nil)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	// Create LLM provider
	provider, err := cli.CreateLLMProvider()
	if err != nil {
		return fmt.Errorf("create LLM provider: %w", err)
	}
	slog.Info("llm provider configured", "provider", provider.Name())

	// Create and run server
	srv := newServer(serverConfig{
		listener:    listener,
		identity:    cli.Identity(),
		llmProvider: provider,
		agentCard:   cli.AgentCard(listener.Addr().String()),
		streaming:   cli.IsStreaming(),
	})

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("shutdown signal received")
		srv.shutdown()
	}()

	if err := srv.run(); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	slog.Info("stopped")
	return nil
}

// server encapsulates the daemon's runtime components.
type server struct {
	listener      *quictransport.MuxListener
	controlServer *grpc.Server
	a2aServer     *grpc.Server
	state         *control.State

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type serverConfig struct {
	listener    *quictransport.MuxListener
	identity    string
	llmProvider llm.Provider
	agentCard   *a2a.AgentCard
	streaming   bool
}

func newServer(cfg serverConfig) *server {
	ctx, cancel := context.WithCancel(context.Background())

	s := &server{
		listener: cfg.listener,
		state:    control.NewState(cfg.identity),
		ctx:      ctx,
		cancel:   cancel,
	}

	// Create control server
	s.controlServer = grpc.NewServer()
	controlSvc := control.NewServer(s.state, s.triggerShutdown)
	latisv1.RegisterControlServiceServer(s.controlServer, controlSvc)

	// Create A2A server with LLM provider
	s.a2aServer = grpc.NewServer()
	executor := &a2aexec.Executor{
		Provider:  cfg.llmProvider,
		Streaming: cfg.streaming,
	}

	a2aexec.RegisterWithGRPC(s.a2aServer, &a2aexec.ServerConfig{
		Executor:  executor,
		AgentCard: cfg.agentCard,
	})

	return s
}

func (s *server) run() error {
	s.state.SetReady()
	slog.Info("ready", "addr", s.listener.Addr(), "transport", "QUIC")
	slog.Debug("stream type registered", "type", "control", "id", fmt.Sprintf("0x%02x", quictransport.StreamTypeControl))
	slog.Debug("stream type registered", "type", "a2a", "id", fmt.Sprintf("0x%02x", quictransport.StreamTypeA2A))

	errChan := make(chan error, 2)

	// Start control server
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.controlServer.Serve(s.listener.ControlListener()); err != nil {
			select {
			case <-s.ctx.Done():
			case errChan <- fmt.Errorf("control server: %w", err):
			}
		}
	}()

	// Start A2A server
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.a2aServer.Serve(s.listener.A2AListener()); err != nil {
			select {
			case <-s.ctx.Done():
			case errChan <- fmt.Errorf("a2a server: %w", err):
			}
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-s.ctx.Done():
		slog.Debug("context cancelled")
	case err := <-errChan:
		return err
	}

	return nil
}

func (s *server) stopServers(graceful bool) {
	s.cancel()
	s.listener.Close()
	if graceful {
		s.controlServer.GracefulStop()
		s.a2aServer.GracefulStop()
	} else {
		s.controlServer.Stop()
		s.a2aServer.Stop()
	}
}

func (s *server) triggerShutdown(graceful bool, timeout time.Duration, reason string) {
	slog.Info("shutdown requested", "graceful", graceful, "timeout", timeout, "reason", reason)

	s.state.SetDraining()

	if graceful && timeout > 0 {
		go func() {
			time.Sleep(timeout)
			slog.Warn("graceful shutdown timeout exceeded, forcing stop")
			s.controlServer.Stop()
			s.a2aServer.Stop()
		}()
	}

	s.stopServers(graceful)
	s.state.SetStopped()
}

func (s *server) shutdown() {
	s.triggerShutdown(true, 30*time.Second, "signal")
	s.wg.Wait()
}

func setupServerTLS(cli *CLI) (*tls.Config, error) {
	// Handle PKI initialization
	if cli.PKI.Init {
		if err := initializeServerPKI(cli); err != nil {
			return nil, fmt.Errorf("initialize PKI: %w", err)
		}
	}

	// Load CA
	ca, err := pki.LoadCA(cli.PKI.CACert, cli.PKI.CAKey)
	if err != nil {
		return nil, fmt.Errorf("load CA (try --pki-init to generate): %w", err)
	}

	// Load certificate
	cert, err := pki.LoadCert(cli.PKI.Cert, cli.PKI.Key)
	if err != nil {
		return nil, fmt.Errorf("load cert (try --pki-init to generate): %w", err)
	}

	// Create mTLS server config
	return pki.ServerTLSConfig(cert, ca)
}

func initializeServerPKI(cli *CLI) error {
	// Ensure directory exists
	if err := os.MkdirAll(cli.PKI.Dir, 0700); err != nil {
		return fmt.Errorf("create PKI directory: %w", err)
	}

	// Check if CA exists
	var ca *pki.CA
	var err error

	if pki.CAExists(cli.PKI.Dir) {
		slog.Debug("loading existing CA", "dir", cli.PKI.Dir)
		ca, err = pki.LoadCA(cli.PKI.CACert, cli.PKI.CAKey)
		if err != nil {
			return fmt.Errorf("load existing CA: %w", err)
		}
	} else {
		slog.Info("generating new CA")
		ca, err = pki.GenerateCA()
		if err != nil {
			return fmt.Errorf("generate CA: %w", err)
		}
		if err := ca.Save(cli.PKI.Dir); err != nil {
			return fmt.Errorf("save CA: %w", err)
		}
		slog.Info("CA saved", "dir", cli.PKI.Dir)
	}

	// Check if cert exists
	if pki.CertExists(cli.PKI.Cert, cli.PKI.Key) {
		slog.Debug("certificate already exists", "cert", cli.PKI.Cert)
		return nil
	}

	// Generate certificate
	name := cli.Agent.Name
	if name == "" {
		name = uuid.New().String()[:8]
	}

	slog.Info("generating certificate", "name", name)
	identity := pki.UnitIdentity(name)
	cert, err := pki.GenerateCert(ca, identity, true, true) // server + client
	if err != nil {
		return fmt.Errorf("generate cert: %w", err)
	}

	if err := cert.Save(cli.PKI.Cert, cli.PKI.Key); err != nil {
		return fmt.Errorf("save cert: %w", err)
	}
	slog.Info("certificate saved", "path", cli.PKI.Cert)

	return nil
}
