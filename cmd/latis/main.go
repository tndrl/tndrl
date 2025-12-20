// latis is the CLI and control plane for managing distributed AI agents.
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	latisv1 "github.com/shanemcd/latis/gen/go/latis/v1"
	"github.com/shanemcd/latis/pkg/pki"
	quictransport "github.com/shanemcd/latis/pkg/transport/quic"
)

var (
	addr    = flag.String("addr", "localhost:4433", "unit address to connect to")
	prompt  = flag.String("prompt", "", "prompt to send (if empty, sends ping)")
	pkiDir  = flag.String("pki-dir", "", "PKI directory (default: ~/.latis/pki)")
	caCert  = flag.String("ca-cert", "", "CA certificate path (overrides pki-dir)")
	cert    = flag.String("cert", "", "cmdr certificate path (overrides pki-dir)")
	key     = flag.String("key", "", "cmdr private key path (overrides pki-dir)")
	initPKI = flag.Bool("init-pki", false, "initialize PKI (generate cmdr cert if CA exists)")
)

func main() {
	flag.Parse()

	log.Printf("latis connecting to %s", *addr)

	tlsConfig, err := setupTLS()
	if err != nil {
		log.Fatalf("failed to setup TLS: %v", err)
	}

	// Create gRPC connection over QUIC
	dialer := quictransport.NewDialer(tlsConfig, nil)

	// QUIC handles TLS at the transport layer, so gRPC uses "insecure" credentials
	// (the connection is actually secured by QUIC's TLS 1.3)
	conn, err := grpc.NewClient(
		*addr,
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Create client
	client := latisv1.NewLatisServiceClient(conn)

	// Establish bidirectional stream
	ctx := context.Background()
	stream, err := client.Connect(ctx)
	if err != nil {
		log.Fatalf("failed to establish stream: %v", err)
	}

	// Send message based on flags
	msgID := uuid.New().String()

	if *prompt != "" {
		// Send prompt
		log.Printf("sending prompt: %s", *prompt)
		if err := stream.Send(&latisv1.ConnectRequest{
			Id: msgID,
			Payload: &latisv1.ConnectRequest_PromptSend{
				PromptSend: &latisv1.PromptSend{
					Content: *prompt,
				},
			},
		}); err != nil {
			log.Fatalf("failed to send prompt: %v", err)
		}
	} else {
		// Send ping
		log.Println("sending ping")
		if err := stream.Send(&latisv1.ConnectRequest{
			Id: msgID,
			Payload: &latisv1.ConnectRequest_Ping{
				Ping: &latisv1.Ping{
					Timestamp: time.Now().UnixNano(),
				},
			},
		}); err != nil {
			log.Fatalf("failed to send ping: %v", err)
		}
	}

	// Receive responses
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			log.Println("stream closed")
			break
		}
		if err != nil {
			log.Fatalf("failed to receive: %v", err)
		}

		// Handle response based on type
		switch payload := resp.Payload.(type) {
		case *latisv1.ConnectResponse_ResponseChunk:
			fmt.Print(payload.ResponseChunk.Content)

		case *latisv1.ConnectResponse_ResponseComplete:
			fmt.Println()
			log.Printf("response complete for request %s", payload.ResponseComplete.RequestId)
			os.Exit(0)

		case *latisv1.ConnectResponse_Pong:
			latency := time.Now().UnixNano() - payload.Pong.PingTimestamp
			log.Printf("pong received, latency=%v", time.Duration(latency))
			os.Exit(0)

		case *latisv1.ConnectResponse_Error:
			log.Printf("error: %s - %s", payload.Error.Code, payload.Error.Message)
			os.Exit(1)

		default:
			log.Printf("unhandled response type: %T", payload)
		}
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
