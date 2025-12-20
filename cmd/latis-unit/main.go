// latis-unit is the agent endpoint daemon that runs on remote machines.
package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	latisv1 "github.com/shanemcd/latis/gen/go/latis/v1"
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

func main() {
	flag.Parse()

	log.Printf("latis-unit starting on %s", *addr)

	tlsConfig, err := setupTLS()
	if err != nil {
		log.Fatalf("failed to setup TLS: %v", err)
	}

	// Create QUIC listener
	listener, err := quictransport.Listen(*addr, tlsConfig, nil)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	log.Printf("listening on %s (QUIC)", listener.Addr())

	// Create gRPC server
	grpcServer := grpc.NewServer()
	latisv1.RegisterLatisServiceServer(grpcServer, &server{})

	// Serve
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
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

// server implements LatisServiceServer
type server struct {
	latisv1.UnimplementedLatisServiceServer
}

// Connect handles the bidirectional stream
func (s *server) Connect(stream latisv1.LatisService_ConnectServer) error {
	log.Println("new connection established")

	for {
		// Receive a message from cmdr
		req, err := stream.Recv()
		if err == io.EOF {
			log.Println("connection closed by client")
			return nil
		}
		if err != nil {
			log.Printf("error receiving: %v", err)
			return err
		}

		log.Printf("received message id=%s", req.Id)

		// Handle the message based on its type
		switch payload := req.Payload.(type) {
		case *latisv1.ConnectRequest_PromptSend:
			log.Printf("prompt: %s", payload.PromptSend.Content)

			// Echo the prompt back as response chunks
			content := fmt.Sprintf("Echo: %s", payload.PromptSend.Content)

			// Send a response chunk
			if err := stream.Send(&latisv1.ConnectResponse{
				Id: req.Id,
				Payload: &latisv1.ConnectResponse_ResponseChunk{
					ResponseChunk: &latisv1.ResponseChunk{
						RequestId: req.Id,
						Content:   content,
						Sequence:  0,
					},
				},
			}); err != nil {
				return err
			}

			// Send response complete
			if err := stream.Send(&latisv1.ConnectResponse{
				Id: req.Id,
				Payload: &latisv1.ConnectResponse_ResponseComplete{
					ResponseComplete: &latisv1.ResponseComplete{
						RequestId: req.Id,
					},
				},
			}); err != nil {
				return err
			}

		case *latisv1.ConnectRequest_Ping:
			log.Printf("ping received, timestamp=%d", payload.Ping.Timestamp)

			// Send pong
			if err := stream.Send(&latisv1.ConnectResponse{
				Id: req.Id,
				Payload: &latisv1.ConnectResponse_Pong{
					Pong: &latisv1.Pong{
						PingTimestamp: payload.Ping.Timestamp,
						PongTimestamp: time.Now().UnixNano(),
					},
				},
			}); err != nil {
				return err
			}

		default:
			log.Printf("unhandled message type: %T", payload)
		}
	}
}
