package quic

import (
	"context"
	"crypto/tls"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/shanemcd/latis/pkg/pki"
)

func setupTestTLS(t *testing.T) (serverConfig, clientConfig *tls.Config) {
	t.Helper()

	ca, err := pki.GenerateCA()
	if err != nil {
		t.Fatalf("generate CA: %v", err)
	}

	serverCert, err := pki.GenerateCert(ca, pki.NodeIdentity("test-server"), true, false)
	if err != nil {
		t.Fatalf("generate server cert: %v", err)
	}

	clientCert, err := pki.GenerateCert(ca, pki.NodeIdentity("test-client"), false, true)
	if err != nil {
		t.Fatalf("generate client cert: %v", err)
	}

	serverConfig, err = pki.ServerTLSConfig(serverCert, ca)
	if err != nil {
		t.Fatalf("server TLS config: %v", err)
	}

	clientConfig, err = pki.ClientTLSConfig(clientCert, ca, "localhost")
	if err != nil {
		t.Fatalf("client TLS config: %v", err)
	}

	return serverConfig, clientConfig
}

func TestMuxListener_StreamRouting(t *testing.T) {
	serverTLS, clientTLS := setupTestTLS(t)

	// Start multiplexed listener
	listener, err := ListenMux("127.0.0.1:0", serverTLS, nil)
	if err != nil {
		t.Fatalf("ListenMux: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()

	// Track received messages by stream type
	var mu sync.Mutex
	received := make(map[StreamType][]string)

	// Handle control streams
	go func() {
		controlListener := listener.ControlListener()
		for {
			conn, err := controlListener.Accept()
			if err != nil {
				return
			}
			go func(c interface{ io.ReadWriteCloser }) {
				defer c.Close()
				buf := make([]byte, 1024)
				n, err := c.Read(buf)
				if err != nil {
					return
				}
				mu.Lock()
				received[StreamTypeControl] = append(received[StreamTypeControl], string(buf[:n]))
				mu.Unlock()
				c.Write([]byte("control-ack"))
			}(conn)
		}
	}()

	// Handle A2A streams
	go func() {
		a2aListener := listener.A2AListener()
		for {
			conn, err := a2aListener.Accept()
			if err != nil {
				return
			}
			go func(c interface{ io.ReadWriteCloser }) {
				defer c.Close()
				buf := make([]byte, 1024)
				n, err := c.Read(buf)
				if err != nil {
					return
				}
				mu.Lock()
				received[StreamTypeA2A] = append(received[StreamTypeA2A], string(buf[:n]))
				mu.Unlock()
				c.Write([]byte("a2a-ack"))
			}(conn)
		}
	}()

	// Give listeners time to start
	time.Sleep(10 * time.Millisecond)

	// Create client
	dialer := NewMuxDialer(clientTLS, nil)
	defer dialer.Close()

	ctx := context.Background()

	// Send control message
	controlConn, err := dialer.DialControl(ctx, addr)
	if err != nil {
		t.Fatalf("DialControl: %v", err)
	}
	controlConn.Write([]byte("ping"))
	buf := make([]byte, 1024)
	n, _ := controlConn.Read(buf)
	if string(buf[:n]) != "control-ack" {
		t.Errorf("expected control-ack, got %q", buf[:n])
	}
	controlConn.Close()

	// Send A2A message
	a2aConn, err := dialer.DialA2A(ctx, addr)
	if err != nil {
		t.Fatalf("DialA2A: %v", err)
	}
	a2aConn.Write([]byte("hello-agent"))
	n, _ = a2aConn.Read(buf)
	if string(buf[:n]) != "a2a-ack" {
		t.Errorf("expected a2a-ack, got %q", buf[:n])
	}
	a2aConn.Close()

	// Verify routing
	time.Sleep(10 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()

	if len(received[StreamTypeControl]) != 1 || received[StreamTypeControl][0] != "ping" {
		t.Errorf("control stream: expected [ping], got %v", received[StreamTypeControl])
	}
	if len(received[StreamTypeA2A]) != 1 || received[StreamTypeA2A][0] != "hello-agent" {
		t.Errorf("a2a stream: expected [hello-agent], got %v", received[StreamTypeA2A])
	}
}

func TestMuxDialer_ConnectionReuse(t *testing.T) {
	serverTLS, clientTLS := setupTestTLS(t)

	listener, err := ListenMux("127.0.0.1:0", serverTLS, nil)
	if err != nil {
		t.Fatalf("ListenMux: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()

	// Accept and echo on both stream types
	go func() {
		for {
			conn, err := listener.ControlListener().Accept()
			if err != nil {
				return
			}
			go io.Copy(conn, conn)
		}
	}()
	go func() {
		for {
			conn, err := listener.A2AListener().Accept()
			if err != nil {
				return
			}
			go io.Copy(conn, conn)
		}
	}()

	time.Sleep(10 * time.Millisecond)

	dialer := NewMuxDialer(clientTLS, nil)
	defer dialer.Close()

	ctx := context.Background()

	// Open multiple streams - should reuse same QUIC connection
	conn1, err := dialer.DialControl(ctx, addr)
	if err != nil {
		t.Fatalf("DialControl 1: %v", err)
	}

	conn2, err := dialer.DialA2A(ctx, addr)
	if err != nil {
		t.Fatalf("DialA2A: %v", err)
	}

	conn3, err := dialer.DialControl(ctx, addr)
	if err != nil {
		t.Fatalf("DialControl 2: %v", err)
	}

	// Verify all work independently
	testEcho := func(conn interface{ io.ReadWriter }, msg string) {
		conn.Write([]byte(msg))
		buf := make([]byte, len(msg))
		io.ReadFull(conn.(io.Reader), buf)
		if string(buf) != msg {
			t.Errorf("expected %q, got %q", msg, buf)
		}
	}

	testEcho(conn1, "control-1")
	testEcho(conn2, "a2a-msg")
	testEcho(conn3, "control-2")

	// Check that only one connection exists in the dialer
	dialer.mu.Lock()
	numConns := len(dialer.conns)
	dialer.mu.Unlock()

	if numConns != 1 {
		t.Errorf("expected 1 connection, got %d", numConns)
	}

	conn1.(io.Closer).Close()
	conn2.(io.Closer).Close()
	conn3.(io.Closer).Close()
}

func TestStreamType_String(t *testing.T) {
	tests := []struct {
		st   StreamType
		want string
	}{
		{StreamTypeControl, "control"},
		{StreamTypeA2A, "a2a"},
		{StreamType(0xFF), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.st.String(); got != tt.want {
			t.Errorf("StreamType(%d).String() = %q, want %q", tt.st, got, tt.want)
		}
	}
}
