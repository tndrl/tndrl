package pki

import (
	"crypto/tls"
	"net"
	"testing"
	"time"
)

func TestServerTLSConfig(t *testing.T) {
	ca, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA() error = %v", err)
	}

	serverCert, err := GenerateCert(ca, "spiffe://latis/node/test", true, false)
	if err != nil {
		t.Fatalf("GenerateCert() error = %v", err)
	}

	config, err := ServerTLSConfig(serverCert, ca)
	if err != nil {
		t.Fatalf("ServerTLSConfig() error = %v", err)
	}

	if config.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Errorf("ClientAuth = %v, want RequireAndVerifyClientCert", config.ClientAuth)
	}
	if config.ClientCAs == nil {
		t.Error("ClientCAs is nil")
	}
	if len(config.Certificates) == 0 {
		t.Error("No certificates in config")
	}
	if config.MinVersion != tls.VersionTLS13 {
		t.Errorf("MinVersion = %v, want TLS 1.3", config.MinVersion)
	}
}

func TestClientTLSConfig(t *testing.T) {
	ca, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA() error = %v", err)
	}

	clientCert, err := GenerateCert(ca, "spiffe://latis/node/client", false, true)
	if err != nil {
		t.Fatalf("GenerateCert() error = %v", err)
	}

	config, err := ClientTLSConfig(clientCert, ca, "localhost")
	if err != nil {
		t.Fatalf("ClientTLSConfig() error = %v", err)
	}

	if config.RootCAs == nil {
		t.Error("RootCAs is nil")
	}
	if config.ServerName != "localhost" {
		t.Errorf("ServerName = %q, want %q", config.ServerName, "localhost")
	}
	if len(config.Certificates) == 0 {
		t.Error("No certificates in config")
	}
	if config.MinVersion != tls.VersionTLS13 {
		t.Errorf("MinVersion = %v, want TLS 1.3", config.MinVersion)
	}
}

func TestMTLSHandshake(t *testing.T) {
	// Generate CA and certificates
	ca, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA() error = %v", err)
	}

	serverCert, err := GenerateCert(ca, "spiffe://latis/node/test", true, false)
	if err != nil {
		t.Fatalf("GenerateCert() for server error = %v", err)
	}

	clientCert, err := GenerateCert(ca, "spiffe://latis/node/client", false, true)
	if err != nil {
		t.Fatalf("GenerateCert() for client error = %v", err)
	}

	// Create TLS configs
	serverConfig, err := ServerTLSConfig(serverCert, ca)
	if err != nil {
		t.Fatalf("ServerTLSConfig() error = %v", err)
	}

	clientConfig, err := ClientTLSConfig(clientCert, ca, "localhost")
	if err != nil {
		t.Fatalf("ClientTLSConfig() error = %v", err)
	}

	// Create a listener
	listener, err := tls.Listen("tcp", "127.0.0.1:0", serverConfig)
	if err != nil {
		t.Fatalf("tls.Listen() error = %v", err)
	}
	defer listener.Close()

	// Channel to communicate results
	serverDone := make(chan error, 1)
	clientDone := make(chan error, 1)

	// Server goroutine
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer conn.Close()

		// Perform handshake
		tlsConn := conn.(*tls.Conn)
		if err := tlsConn.Handshake(); err != nil {
			serverDone <- err
			return
		}

		// Read a message
		buf := make([]byte, 100)
		n, err := conn.Read(buf)
		if err != nil {
			serverDone <- err
			return
		}

		// Echo it back
		if _, err := conn.Write(buf[:n]); err != nil {
			serverDone <- err
			return
		}

		serverDone <- nil
	}()

	// Client goroutine
	go func() {
		conn, err := tls.Dial("tcp", listener.Addr().String(), clientConfig)
		if err != nil {
			clientDone <- err
			return
		}
		defer conn.Close()

		// Send a message
		msg := []byte("hello mTLS")
		if _, err := conn.Write(msg); err != nil {
			clientDone <- err
			return
		}

		// Read the echo
		buf := make([]byte, 100)
		n, err := conn.Read(buf)
		if err != nil {
			clientDone <- err
			return
		}

		if string(buf[:n]) != string(msg) {
			clientDone <- &net.OpError{Op: "read", Err: &testError{"message mismatch"}}
			return
		}

		clientDone <- nil
	}()

	// Wait for both to complete
	if err := <-serverDone; err != nil {
		t.Fatalf("Server error: %v", err)
	}
	if err := <-clientDone; err != nil {
		t.Fatalf("Client error: %v", err)
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestMTLSHandshake_WrongCA(t *testing.T) {
	// Generate two different CAs
	serverCA, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA() for server error = %v", err)
	}

	clientCA, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA() for client error = %v", err)
	}

	// Server uses serverCA
	serverCert, err := GenerateCert(serverCA, "spiffe://latis/unit/test", true, false)
	if err != nil {
		t.Fatalf("GenerateCert() for server error = %v", err)
	}

	// Client uses clientCA (different!)
	clientCert, err := GenerateCert(clientCA, "spiffe://latis/node/client", false, true)
	if err != nil {
		t.Fatalf("GenerateCert() for client error = %v", err)
	}

	// Server config expects clients signed by serverCA
	serverConfig, err := ServerTLSConfig(serverCert, serverCA)
	if err != nil {
		t.Fatalf("ServerTLSConfig() error = %v", err)
	}

	// Client config trusts clientCA (wrong CA for server)
	clientConfig, err := ClientTLSConfig(clientCert, clientCA, "localhost")
	if err != nil {
		t.Fatalf("ClientTLSConfig() error = %v", err)
	}

	// Create a listener
	listener, err := tls.Listen("tcp", "127.0.0.1:0", serverConfig)
	if err != nil {
		t.Fatalf("tls.Listen() error = %v", err)
	}
	defer listener.Close()

	// Server goroutine - accept and attempt handshake
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		// Try handshake - it will fail due to client cert not being trusted
		tlsConn := conn.(*tls.Conn)
		tlsConn.Handshake()
	}()

	// Use a dialer with timeout to prevent hanging
	dialer := &net.Dialer{
		Timeout: 2 * time.Second,
	}
	conn, err := tls.DialWithDialer(dialer, "tcp", listener.Addr().String(), clientConfig)
	if err == nil {
		conn.Close()
		t.Error("Expected handshake to fail with mismatched CAs")
	}
	// Error is expected - test passes

	// Wait for server goroutine to clean up
	<-serverDone
}
