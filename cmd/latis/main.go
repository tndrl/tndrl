// latis is the CLI and control plane for managing distributed AI agents.
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	latisv1 "github.com/shanemcd/latis/gen/go/latis/v1"
	"github.com/shanemcd/latis/pkg/pki"
	quictransport "github.com/shanemcd/latis/pkg/transport/quic"
)

var (
	addr     = flag.String("addr", "localhost:4433", "unit address to connect to")
	status   = flag.Bool("status", false, "get unit status")
	shutdown = flag.Bool("shutdown", false, "shutdown the unit")
	prompt   = flag.String("prompt", "", "prompt to send via A2A (not yet implemented)")
	pkiDir   = flag.String("pki-dir", "", "PKI directory (default: ~/.latis/pki)")
	caCert   = flag.String("ca-cert", "", "CA certificate path (overrides pki-dir)")
	cert     = flag.String("cert", "", "cmdr certificate path (overrides pki-dir)")
	key      = flag.String("key", "", "cmdr private key path (overrides pki-dir)")
	initPKI  = flag.Bool("init-pki", false, "initialize PKI (generate cmdr cert if CA exists)")
)

func main() {
	flag.Parse()

	log.Printf("latis connecting to %s", *addr)

	tlsConfig, err := setupTLS()
	if err != nil {
		log.Fatalf("failed to setup TLS: %v", err)
	}

	// Create multiplexed QUIC dialer
	muxDialer := quictransport.NewMuxDialer(tlsConfig, nil)
	defer muxDialer.Close()

	// Create Control gRPC connection
	controlConn, err := grpc.NewClient(
		*addr,
		grpc.WithContextDialer(muxDialer.ControlDialer()),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to create control connection: %v", err)
	}
	defer controlConn.Close()

	// Create Control client
	controlClient := latisv1.NewControlServiceClient(controlConn)

	ctx := context.Background()

	// Handle commands
	switch {
	case *status:
		doGetStatus(ctx, controlClient)
	case *shutdown:
		doShutdown(ctx, controlClient)
	case *prompt != "":
		// TODO: Implement A2A prompt sending
		log.Println("A2A prompt sending not yet implemented")
		log.Println("For now, use --status or --shutdown, or default ping")
		os.Exit(1)
	default:
		doPing(ctx, controlClient)
	}
}

func doPing(ctx context.Context, client latisv1.ControlServiceClient) {
	log.Println("sending ping via Control stream")

	pingTime := time.Now()
	resp, err := client.Ping(ctx, &latisv1.PingRequest{
		Timestamp: pingTime.UnixNano(),
	})
	if err != nil {
		log.Fatalf("ping failed: %v", err)
	}

	rtt := time.Since(pingTime)
	serverLatency := time.Duration(resp.PongTimestamp - resp.PingTimestamp)

	log.Printf("pong received: rtt=%v, server_latency=%v", rtt, serverLatency)
}

func doGetStatus(ctx context.Context, client latisv1.ControlServiceClient) {
	log.Println("getting unit status via Control stream")

	resp, err := client.GetStatus(ctx, &latisv1.GetStatusRequest{})
	if err != nil {
		log.Fatalf("get status failed: %v", err)
	}

	fmt.Printf("Unit Status:\n")
	fmt.Printf("  Identity:     %s\n", resp.Identity)
	fmt.Printf("  State:        %s\n", resp.State.String())
	fmt.Printf("  Uptime:       %ds\n", resp.UptimeSeconds)
	fmt.Printf("  Active Tasks: %d\n", resp.ActiveTasks)
	if len(resp.Metadata) > 0 {
		fmt.Printf("  Metadata:\n")
		for k, v := range resp.Metadata {
			fmt.Printf("    %s: %s\n", k, v)
		}
	}
}

func doShutdown(ctx context.Context, client latisv1.ControlServiceClient) {
	log.Println("requesting shutdown via Control stream")

	resp, err := client.Shutdown(ctx, &latisv1.ShutdownRequest{
		Graceful:       true,
		TimeoutSeconds: 30,
		Reason:         "requested by cmdr",
	})
	if err != nil {
		log.Fatalf("shutdown request failed: %v", err)
	}

	if resp.Accepted {
		log.Println("shutdown accepted")
	} else {
		log.Printf("shutdown rejected: %s", resp.RejectionReason)
		os.Exit(1)
	}
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
	certPath := *cert
	keyPath := *key

	if caCertPath == "" {
		caCertPath = filepath.Join(dir, "ca.crt")
	}
	if certPath == "" {
		certPath = filepath.Join(dir, "cmdr.crt")
	}
	if keyPath == "" {
		keyPath = filepath.Join(dir, "cmdr.key")
	}

	// Handle PKI initialization
	if *initPKI {
		if err := initializePKI(dir, caCertPath, certPath, keyPath); err != nil {
			return nil, fmt.Errorf("initialize PKI: %w", err)
		}
	}

	// Load CA (only need cert for verification, not key)
	caKeyPath := filepath.Join(dir, "ca.key")
	ca, err := pki.LoadCA(caCertPath, caKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load CA (run latis-unit --init-pki first): %w", err)
	}

	// Load cmdr certificate
	cmdrCert, err := pki.LoadCert(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("load cmdr cert (try --init-pki to generate): %w", err)
	}

	// Create mTLS client config
	// Extract hostname from address for TLS ServerName verification
	host, _, err := net.SplitHostPort(*addr)
	if err != nil {
		host = *addr // fallback if no port
	}
	return pki.ClientTLSConfig(cmdrCert, ca, host)
}

func initializePKI(dir, caCertPath, certPath, keyPath string) error {
	// Check if CA exists (we need CA to sign cmdr cert)
	caKeyPath := filepath.Join(dir, "ca.key")
	if !pki.CertExists(caCertPath, caKeyPath) {
		return fmt.Errorf("CA not found at %s - run latis-unit --init-pki first to create CA", caCertPath)
	}

	// Check if cmdr cert already exists
	if pki.CertExists(certPath, keyPath) {
		log.Println("cmdr certificate already exists")
		return nil
	}

	// Load CA to sign cmdr cert
	ca, err := pki.LoadCA(caCertPath, caKeyPath)
	if err != nil {
		return fmt.Errorf("load CA: %w", err)
	}

	// Generate cmdr certificate
	log.Println("generating cmdr certificate")
	identity := pki.CmdrIdentity()
	cmdrCert, err := pki.GenerateCert(ca, identity, false, true) // client only
	if err != nil {
		return fmt.Errorf("generate cmdr cert: %w", err)
	}

	if err := cmdrCert.Save(certPath, keyPath); err != nil {
		return fmt.Errorf("save cmdr cert: %w", err)
	}
	log.Printf("cmdr certificate saved to %s", certPath)

	return nil
}
