package pki

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// ServerTLSConfig creates a TLS config for a server that requires client certificates.
func ServerTLSConfig(cert *Cert, ca *CA) (*tls.Config, error) {
	tlsCert, err := cert.TLSCertificate()
	if err != nil {
		return nil, fmt.Errorf("create TLS certificate: %w", err)
	}

	caPool := x509.NewCertPool()
	caPool.AddCert(ca.Cert)

	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
		NextProtos:   []string{"latis"},
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// ClientTLSConfig creates a TLS config for a client with its own certificate.
// serverName should match the server certificate's DNS name (e.g., "localhost").
func ClientTLSConfig(cert *Cert, ca *CA, serverName string) (*tls.Config, error) {
	tlsCert, err := cert.TLSCertificate()
	if err != nil {
		return nil, fmt.Errorf("create TLS certificate: %w", err)
	}

	caPool := x509.NewCertPool()
	caPool.AddCert(ca.Cert)

	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		RootCAs:      caPool,
		ServerName:   serverName,
		NextProtos:   []string{"latis"},
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// LoadCACert loads just the CA certificate (without private key) for verification.
// This is useful for clients that only need to verify server certs.
func LoadCACert(certPath string) (*x509.CertPool, error) {
	certPEM, err := readFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(certPEM) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return pool, nil
}

// readFile is a helper for reading files.
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
