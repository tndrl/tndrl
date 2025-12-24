package main

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"

	"github.com/a2aproject/a2a-go/a2aclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	latisv1 "github.com/shanemcd/latis/gen/go/latis/v1"
	"github.com/shanemcd/latis/pkg/pki"
	quictransport "github.com/shanemcd/latis/pkg/transport/quic"
)

// PeerConnection holds connections to a peer.
type PeerConnection struct {
	addr       string
	muxDialer  *quictransport.MuxDialer
	controlConn *grpc.ClientConn
	a2aConn    *grpc.ClientConn
}

// ConnectToPeer establishes a connection to a peer.
func ConnectToPeer(cli *CLI, peerAddr string) (*PeerConnection, error) {
	tlsConfig, err := setupClientTLS(cli, peerAddr)
	if err != nil {
		return nil, fmt.Errorf("setup TLS: %w", err)
	}

	muxDialer := quictransport.NewMuxDialer(tlsConfig, nil)

	// Create Control gRPC connection
	controlConn, err := grpc.NewClient(
		peerAddr,
		grpc.WithContextDialer(muxDialer.ControlDialer()),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		muxDialer.Close()
		return nil, fmt.Errorf("create control connection: %w", err)
	}

	return &PeerConnection{
		addr:        peerAddr,
		muxDialer:   muxDialer,
		controlConn: controlConn,
	}, nil
}

// ControlClient returns a Control service client.
func (pc *PeerConnection) ControlClient() latisv1.ControlServiceClient {
	return latisv1.NewControlServiceClient(pc.controlConn)
}

// A2ATransport returns an A2A transport for sending messages.
func (pc *PeerConnection) A2ATransport() (a2aclient.Transport, error) {
	if pc.a2aConn == nil {
		a2aConn, err := grpc.NewClient(
			pc.addr,
			grpc.WithContextDialer(pc.muxDialer.A2ADialer()),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return nil, fmt.Errorf("create A2A connection: %w", err)
		}
		pc.a2aConn = a2aConn
	}
	return a2aclient.NewGRPCTransport(pc.a2aConn), nil
}

// Close closes all connections.
func (pc *PeerConnection) Close() {
	if pc.a2aConn != nil {
		pc.a2aConn.Close()
	}
	if pc.controlConn != nil {
		pc.controlConn.Close()
	}
	if pc.muxDialer != nil {
		pc.muxDialer.Close()
	}
}

func setupClientTLS(cli *CLI, peerAddr string) (*tls.Config, error) {
	// Handle PKI initialization for client
	if cli.PKI.Init {
		if err := initializeClientPKI(cli); err != nil {
			return nil, fmt.Errorf("initialize PKI: %w", err)
		}
	}

	// Load CA
	ca, err := pki.LoadCA(cli.PKI.CACert, cli.PKI.CAKey)
	if err != nil {
		return nil, fmt.Errorf("load CA (run 'latis serve --pki-init' first): %w", err)
	}

	// Load certificate
	cert, err := pki.LoadCert(cli.PKI.Cert, cli.PKI.Key)
	if err != nil {
		return nil, fmt.Errorf("load cert (try --pki-init to generate): %w", err)
	}

	// Extract hostname for TLS ServerName verification
	host, _, err := net.SplitHostPort(peerAddr)
	if err != nil {
		host = peerAddr
	}

	return pki.ClientTLSConfig(cert, ca, host)
}

func initializeClientPKI(cli *CLI) error {
	// Check if CA exists (we need CA to sign client cert)
	if !pki.CertExists(cli.PKI.CACert, cli.PKI.CAKey) {
		return fmt.Errorf("CA not found at %s - run 'latis serve --pki-init' first to create CA", cli.PKI.CACert)
	}

	// Check if cert already exists
	if pki.CertExists(cli.PKI.Cert, cli.PKI.Key) {
		slog.Debug("certificate already exists", "cert", cli.PKI.Cert)
		return nil
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(cli.PKI.Cert), 0700); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Load CA to sign cert
	ca, err := pki.LoadCA(cli.PKI.CACert, cli.PKI.CAKey)
	if err != nil {
		return fmt.Errorf("load CA: %w", err)
	}

	// Generate certificate (client + server for peer-to-peer)
	// Use a unique node identity based on hostname
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "client"
	}
	slog.Info("generating certificate", "hostname", hostname)
	identity := pki.NodeIdentity(hostname)
	cert, err := pki.GenerateCert(ca, identity, true, true)
	if err != nil {
		return fmt.Errorf("generate cert: %w", err)
	}

	if err := cert.Save(cli.PKI.Cert, cli.PKI.Key); err != nil {
		return fmt.Errorf("save cert: %w", err)
	}
	slog.Info("certificate saved", "path", cli.PKI.Cert)

	return nil
}
