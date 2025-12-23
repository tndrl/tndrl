// latis-unit is the agent endpoint daemon that runs on remote machines.
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	latisv1 "github.com/shanemcd/latis/gen/go/latis/v1"
	"github.com/shanemcd/latis/pkg/a2aexec"
	"github.com/shanemcd/latis/pkg/control"
	"github.com/shanemcd/latis/pkg/pki"
	quictransport "github.com/shanemcd/latis/pkg/transport/quic"
)

var (
	addr    = flag.String("addr", "localhost:4433", "address to listen on")
	pkiDir  = flag.String("pki-dir", "", "PKI directory (default: ~/.latis/pki)")
	caCert  = flag.String("ca-cert", "", "CA certificate path (overrides pki-dir)")
	caKey   = flag.String("ca-key", "", "CA private key path (for init-pki with BYO CA)")
	cert    = flag.String("cert", "", "unit certificate path (overrides pki-dir)")
	key     = flag.String("key", "", "unit private key path (overrides pki-dir)")
	initPKI = flag.Bool("init-pki", false, "initialize PKI (generate CA + unit cert if missing)")
	unitID  = flag.String("unit-id", "", "unit ID for certificate identity (default: auto-generated)")
)

// Unit encapsulates the unit daemon's runtime components.
type Unit struct {
	listener      *quictransport.MuxListener
	controlServer *grpc.Server
	a2aServer     *grpc.Server
	state         *control.State

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewUnit creates a new unit with the given listener and identity.
func NewUnit(listener *quictransport.MuxListener, identity string) *Unit {
	ctx, cancel := context.WithCancel(context.Background())

	u := &Unit{
		listener: listener,
		state:    control.NewState(identity),
		ctx:      ctx,
		cancel:   cancel,
	}

	// Create control server
	u.controlServer = grpc.NewServer()
	controlSvc := control.NewServer(u.state, u.triggerShutdown)
	latisv1.RegisterControlServiceServer(u.controlServer, controlSvc)

	// Create A2A server
	u.a2aServer = grpc.NewServer()
	executor := a2aexec.NewExecutor()
	a2aexec.RegisterWithGRPC(u.a2aServer, &a2aexec.ServerConfig{
		Executor:  executor,
		AgentCard: a2aexec.DefaultAgentCard("latis-unit", "Latis Unit Agent", listener.Addr().String()),
	})

	return u
}

// Run starts both servers and blocks until shutdown.
func (u *Unit) Run() error {
	u.state.SetReady()
	log.Printf("unit ready, listening on %s (QUIC)", u.listener.Addr())
	log.Printf("  control stream: 0x%02x", quictransport.StreamTypeControl)
	log.Printf("  a2a stream:     0x%02x", quictransport.StreamTypeA2A)

	errChan := make(chan error, 2)

	// Start control server
	u.wg.Add(1)
	go func() {
		defer u.wg.Done()
		if err := u.controlServer.Serve(u.listener.ControlListener()); err != nil {
			select {
			case errChan <- fmt.Errorf("control server: %w", err):
			default:
			}
		}
	}()

	// Start A2A server
	u.wg.Add(1)
	go func() {
		defer u.wg.Done()
		if err := u.a2aServer.Serve(u.listener.A2AListener()); err != nil {
			select {
			case errChan <- fmt.Errorf("a2a server: %w", err):
			default:
			}
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-u.ctx.Done():
		log.Println("shutdown signal received")
	case err := <-errChan:
		return err
	}

	return nil
}

// triggerShutdown initiates graceful shutdown.
func (u *Unit) triggerShutdown(graceful bool, timeout time.Duration, reason string) {
	log.Printf("shutdown requested: graceful=%v, timeout=%v, reason=%q", graceful, timeout, reason)

	u.state.SetDraining()

	if graceful {
		// Create timeout context if specified
		if timeout > 0 {
			go func() {
				time.Sleep(timeout)
				log.Println("graceful shutdown timeout exceeded, forcing stop")
				u.controlServer.Stop()
				u.a2aServer.Stop()
			}()
		}
		u.controlServer.GracefulStop()
		u.a2aServer.GracefulStop()
	} else {
		u.controlServer.Stop()
		u.a2aServer.Stop()
	}

	u.state.SetStopped()
	u.cancel()
}

// Shutdown gracefully shuts down the unit.
func (u *Unit) Shutdown() {
	u.triggerShutdown(true, 30*time.Second, "signal")
	u.wg.Wait()
	u.listener.Close()
}

func main() {
	flag.Parse()

	log.Printf("latis-unit starting on %s", *addr)

	tlsConfig, err := setupTLS()
	if err != nil {
		log.Fatalf("failed to setup TLS: %v", err)
	}

	// Create multiplexed QUIC listener
	listener, err := quictransport.ListenMux(*addr, tlsConfig, nil)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Extract identity from certificate or use generated ID
	identity := extractIdentity()

	// Create and run unit
	unit := NewUnit(listener, identity)

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("received shutdown signal")
		unit.Shutdown()
	}()

	if err := unit.Run(); err != nil {
		log.Fatalf("unit error: %v", err)
	}

	log.Println("unit stopped")
}

func extractIdentity() string {
	id := *unitID
	if id == "" {
		id = uuid.New().String()[:8]
	}
	return pki.UnitIdentity(id)
}

func setupTLS() (*tls.Config, error) {
	dir := *pkiDir
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		dir = filepath.Join(home, ".latis", "pki")
	}

	// Determine paths
	caCertPath := *caCert
	caKeyPath := *caKey
	certPath := *cert
	keyPath := *key

	if caCertPath == "" {
		caCertPath = filepath.Join(dir, "ca.crt")
	}
	if caKeyPath == "" {
		caKeyPath = filepath.Join(dir, "ca.key")
	}
	if certPath == "" {
		certPath = filepath.Join(dir, "unit.crt")
	}
	if keyPath == "" {
		keyPath = filepath.Join(dir, "unit.key")
	}

	// Handle PKI initialization
	if *initPKI {
		if err := initializePKI(dir, caCertPath, caKeyPath, certPath, keyPath); err != nil {
			return nil, fmt.Errorf("initialize PKI: %w", err)
		}
	}

	// Load CA
	ca, err := pki.LoadCA(caCertPath, caKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load CA (try --init-pki to generate): %w", err)
	}

	// Load unit certificate
	unitCert, err := pki.LoadCert(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("load unit cert (try --init-pki to generate): %w", err)
	}

	// Create mTLS server config
	return pki.ServerTLSConfig(unitCert, ca)
}

func initializePKI(dir, caCertPath, caKeyPath, certPath, keyPath string) error {
	// Ensure directory exists
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create PKI directory: %w", err)
	}

	// Check if CA exists
	var ca *pki.CA
	var err error

	if pki.CAExists(dir) || (*caCert != "" && *caKey != "") {
		log.Println("loading existing CA")
		ca, err = pki.LoadCA(caCertPath, caKeyPath)
		if err != nil {
			return fmt.Errorf("load existing CA: %w", err)
		}
	} else {
		log.Println("generating new CA")
		ca, err = pki.GenerateCA()
		if err != nil {
			return fmt.Errorf("generate CA: %w", err)
		}
		if err := ca.Save(dir); err != nil {
			return fmt.Errorf("save CA: %w", err)
		}
		log.Printf("CA saved to %s", dir)
	}

	// Check if unit cert exists
	if pki.CertExists(certPath, keyPath) {
		log.Println("unit certificate already exists")
		return nil
	}

	// Generate unit ID if not provided
	id := *unitID
	if id == "" {
		id = uuid.New().String()[:8]
	}

	// Generate unit certificate
	log.Printf("generating unit certificate (id=%s)", id)
	identity := pki.UnitIdentity(id)
	unitCert, err := pki.GenerateCert(ca, identity, true, false) // server only
	if err != nil {
		return fmt.Errorf("generate unit cert: %w", err)
	}

	if err := unitCert.Save(certPath, keyPath); err != nil {
		return fmt.Errorf("save unit cert: %w", err)
	}
	log.Printf("unit certificate saved to %s", certPath)

	return nil
}
