package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2aclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	latisv1 "github.com/shanemcd/latis/gen/go/latis/v1"
	"github.com/shanemcd/latis/pkg/a2aexec"
	"github.com/shanemcd/latis/pkg/pki"
	quictransport "github.com/shanemcd/latis/pkg/transport/quic"
)

// muxTestEnv holds the multiplexed test environment
type muxTestEnv struct {
	ca            *pki.CA
	serverCert    *pki.Cert
	clientCert    *pki.Cert
	addr          string
	listener      *quictransport.MuxListener
	controlServer *grpc.Server
	a2aServer     *grpc.Server
	cleanup       func()
}

// testControlServer implements ControlServiceServer for testing
type testControlServer struct {
	latisv1.UnimplementedControlServiceServer
	identity    string
	shutdownCh  chan struct{}
	startTime   time.Time
	activeTasks int32
}

func (s *testControlServer) Ping(ctx context.Context, req *latisv1.PingRequest) (*latisv1.PingResponse, error) {
	return &latisv1.PingResponse{
		PingTimestamp: req.Timestamp,
		PongTimestamp: time.Now().UnixNano(),
	}, nil
}

func (s *testControlServer) GetStatus(ctx context.Context, req *latisv1.GetStatusRequest) (*latisv1.GetStatusResponse, error) {
	return &latisv1.GetStatusResponse{
		Identity:      s.identity,
		State:         latisv1.UnitState_UNIT_STATE_READY,
		UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
		ActiveTasks:   s.activeTasks,
		Metadata: map[string]string{
			"test": "true",
		},
	}, nil
}

func (s *testControlServer) Shutdown(ctx context.Context, req *latisv1.ShutdownRequest) (*latisv1.ShutdownResponse, error) {
	select {
	case s.shutdownCh <- struct{}{}:
	default:
	}
	return &latisv1.ShutdownResponse{
		Accepted: true,
	}, nil
}

func setupMuxTestEnv(t *testing.T) *muxTestEnv {
	t.Helper()

	// Generate PKI
	ca, err := pki.GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}

	serverCert, err := pki.GenerateCert(ca, pki.UnitIdentity("test-mux"), true, false)
	if err != nil {
		t.Fatalf("GenerateCert for server: %v", err)
	}

	clientCert, err := pki.GenerateCert(ca, pki.CmdrIdentity(), false, true)
	if err != nil {
		t.Fatalf("GenerateCert for client: %v", err)
	}

	// Create server TLS config
	serverTLS, err := pki.ServerTLSConfig(serverCert, ca)
	if err != nil {
		t.Fatalf("ServerTLSConfig: %v", err)
	}

	// Start multiplexed QUIC listener
	listener, err := quictransport.ListenMux("127.0.0.1:0", serverTLS, nil)
	if err != nil {
		t.Fatalf("ListenMux: %v", err)
	}

	// Create and start Control gRPC server
	controlServer := grpc.NewServer()
	testControl := &testControlServer{
		identity:   "spiffe://latis/unit/test-mux",
		shutdownCh: make(chan struct{}, 1),
		startTime:  time.Now(),
	}
	latisv1.RegisterControlServiceServer(controlServer, testControl)

	go func() {
		controlServer.Serve(listener.ControlListener())
	}()

	// Create and start A2A gRPC server
	a2aServer := grpc.NewServer()
	executor := a2aexec.NewExecutor()
	a2aexec.RegisterWithGRPC(a2aServer, &a2aexec.ServerConfig{
		Executor:  executor,
		AgentCard: a2aexec.DefaultAgentCard("test-unit", "Test Unit", listener.Addr().String()),
	})

	go func() {
		a2aServer.Serve(listener.A2AListener())
	}()

	return &muxTestEnv{
		ca:            ca,
		serverCert:    serverCert,
		clientCert:    clientCert,
		addr:          listener.Addr().String(),
		listener:      listener,
		controlServer: controlServer,
		a2aServer:     a2aServer,
		cleanup: func() {
			// Close listener first to stop accepting new connections
			listener.Close()
			// GracefulStop waits for handlers but with listener closed,
			// should complete quickly
			controlServer.GracefulStop()
			a2aServer.GracefulStop()
		},
	}
}

func (e *muxTestEnv) connectControl(t *testing.T) (latisv1.ControlServiceClient, func()) {
	t.Helper()

	clientTLS, err := pki.ClientTLSConfig(e.clientCert, e.ca, "localhost")
	if err != nil {
		t.Fatalf("ClientTLSConfig: %v", err)
	}

	muxDialer := quictransport.NewMuxDialer(clientTLS, nil)

	conn, err := grpc.NewClient(
		e.addr,
		grpc.WithContextDialer(muxDialer.ControlDialer()),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	client := latisv1.NewControlServiceClient(conn)
	return client, func() {
		conn.Close()
		muxDialer.Close()
	}
}

func (e *muxTestEnv) connectA2A(t *testing.T) (a2aclient.Transport, func()) {
	t.Helper()

	clientTLS, err := pki.ClientTLSConfig(e.clientCert, e.ca, "localhost")
	if err != nil {
		t.Fatalf("ClientTLSConfig: %v", err)
	}

	muxDialer := quictransport.NewMuxDialer(clientTLS, nil)

	conn, err := grpc.NewClient(
		e.addr,
		grpc.WithContextDialer(muxDialer.A2ADialer()),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	transport := a2aclient.NewGRPCTransport(conn)
	return transport, func() {
		transport.Destroy()
		conn.Close()
		muxDialer.Close()
	}
}

func TestControlPing(t *testing.T) {
	env := setupMuxTestEnv(t)
	defer env.cleanup()

	client, cleanup := env.connectControl(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sendTime := time.Now().UnixNano()
	resp, err := client.Ping(ctx, &latisv1.PingRequest{
		Timestamp: sendTime,
	})
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}

	if resp.PingTimestamp != sendTime {
		t.Errorf("PingTimestamp = %d, want %d", resp.PingTimestamp, sendTime)
	}

	if resp.PongTimestamp <= sendTime {
		t.Errorf("PongTimestamp should be after PingTimestamp")
	}

	latency := time.Duration(time.Now().UnixNano() - sendTime)
	t.Logf("Control Ping RTT: %v", latency)
}

func TestControlGetStatus(t *testing.T) {
	env := setupMuxTestEnv(t)
	defer env.cleanup()

	client, cleanup := env.connectControl(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetStatus(ctx, &latisv1.GetStatusRequest{})
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	if resp.Identity != "spiffe://latis/unit/test-mux" {
		t.Errorf("Identity = %q, want %q", resp.Identity, "spiffe://latis/unit/test-mux")
	}

	if resp.State != latisv1.UnitState_UNIT_STATE_READY {
		t.Errorf("State = %v, want READY", resp.State)
	}

	if resp.Metadata["test"] != "true" {
		t.Errorf("Metadata[test] = %q, want %q", resp.Metadata["test"], "true")
	}

	t.Logf("Unit status: identity=%s, state=%v, uptime=%ds", resp.Identity, resp.State, resp.UptimeSeconds)
}

func TestControlShutdown(t *testing.T) {
	env := setupMuxTestEnv(t)
	defer env.cleanup()

	client, cleanup := env.connectControl(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Shutdown(ctx, &latisv1.ShutdownRequest{
		Graceful:       true,
		TimeoutSeconds: 10,
		Reason:         "integration test",
	})
	if err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	if !resp.Accepted {
		t.Errorf("Shutdown not accepted: %s", resp.RejectionReason)
	}

	t.Logf("Shutdown accepted")
}

func TestControlMultiplePings(t *testing.T) {
	env := setupMuxTestEnv(t)
	defer env.cleanup()

	client, cleanup := env.connectControl(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	numPings := 10
	for i := 0; i < numPings; i++ {
		sendTime := time.Now().UnixNano()
		resp, err := client.Ping(ctx, &latisv1.PingRequest{
			Timestamp: sendTime,
		})
		if err != nil {
			t.Fatalf("Ping %d: %v", i, err)
		}

		if resp.PingTimestamp != sendTime {
			t.Errorf("Ping %d: PingTimestamp mismatch", i)
		}
	}

	t.Logf("Successfully sent %d pings over Control stream", numPings)
}

func TestConnectionReuse(t *testing.T) {
	env := setupMuxTestEnv(t)
	defer env.cleanup()

	clientTLS, err := pki.ClientTLSConfig(env.clientCert, env.ca, "localhost")
	if err != nil {
		t.Fatalf("ClientTLSConfig: %v", err)
	}

	// Use a single MuxDialer for multiple connections
	muxDialer := quictransport.NewMuxDialer(clientTLS, nil)
	defer muxDialer.Close()

	// Create multiple gRPC connections - they should reuse the same QUIC connection
	var clients []latisv1.ControlServiceClient
	var conns []*grpc.ClientConn

	for i := 0; i < 3; i++ {
		conn, err := grpc.NewClient(
			env.addr,
			grpc.WithContextDialer(muxDialer.ControlDialer()),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			t.Fatalf("NewClient %d: %v", i, err)
		}
		conns = append(conns, conn)
		clients = append(clients, latisv1.NewControlServiceClient(conn))
	}

	defer func() {
		for _, conn := range conns {
			conn.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// All clients should work
	for i, client := range clients {
		_, err := client.Ping(ctx, &latisv1.PingRequest{
			Timestamp: time.Now().UnixNano(),
		})
		if err != nil {
			t.Fatalf("Client %d Ping: %v", i, err)
		}
	}

	t.Logf("Successfully used %d clients with connection reuse", len(clients))
}

func TestConnectionRequiresMTLS(t *testing.T) {
	env := setupMuxTestEnv(t)
	defer env.cleanup()

	// Try to connect without proper client cert
	wrongCA, err := pki.GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}

	wrongClientCert, err := pki.GenerateCert(wrongCA, pki.CmdrIdentity(), false, true)
	if err != nil {
		t.Fatalf("GenerateCert: %v", err)
	}

	// Use wrong CA's cert as client cert (signed by different CA)
	clientTLS, err := pki.ClientTLSConfig(wrongClientCert, wrongCA, "localhost")
	if err != nil {
		t.Fatalf("ClientTLSConfig: %v", err)
	}

	muxDialer := quictransport.NewMuxDialer(clientTLS, nil)
	defer muxDialer.Close()

	conn, err := grpc.NewClient(
		env.addr,
		grpc.WithContextDialer(muxDialer.ControlDialer()),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		// Connection creation might fail
		t.Logf("NewClient failed (expected): %v", err)
		return
	}
	defer conn.Close()

	client := latisv1.NewControlServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = client.Ping(ctx, &latisv1.PingRequest{
		Timestamp: time.Now().UnixNano(),
	})
	if err == nil {
		t.Error("Expected connection to fail with wrong CA, but it succeeded")
	} else {
		t.Logf("Connection correctly rejected: %v", err)
	}
}

// =============================================================================
// A2A Stream Integration Tests
// =============================================================================

func TestA2ASendMessage(t *testing.T) {
	env := setupMuxTestEnv(t)
	defer env.cleanup()

	transport, cleanup := env.connectA2A(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Send a message
	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: "Hello, A2A!"})
	resp, err := transport.SendMessage(ctx, &a2a.MessageSendParams{Message: msg})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	// Extract response text based on response type
	var responseText string
	switch r := resp.(type) {
	case *a2a.Task:
		t.Logf("Task ID: %s, State: %s", r.ID, r.Status.State)
		if r.Status.Message != nil {
			for _, part := range r.Status.Message.Parts {
				if text, ok := part.(a2a.TextPart); ok {
					responseText = text.Text
					break
				}
			}
		}
	case *a2a.Message:
		t.Logf("Message role: %s", r.Role)
		for _, part := range r.Parts {
			if text, ok := part.(a2a.TextPart); ok {
				responseText = text.Text
				break
			}
		}
	default:
		t.Fatalf("Unexpected response type: %T", resp)
	}

	if !strings.Contains(responseText, "Echo:") {
		t.Errorf("Expected echo response, got: %s", responseText)
	}

	t.Logf("A2A Response: %s", responseText)
}

func TestA2AGetAgentCard(t *testing.T) {
	env := setupMuxTestEnv(t)
	defer env.cleanup()

	transport, cleanup := env.connectA2A(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	card, err := transport.GetAgentCard(ctx)
	if err != nil {
		t.Fatalf("GetAgentCard: %v", err)
	}

	if card.Name != "test-unit" {
		t.Errorf("Name = %q, want %q", card.Name, "test-unit")
	}

	if card.PreferredTransport != a2a.TransportProtocolGRPC {
		t.Errorf("PreferredTransport = %v, want gRPC", card.PreferredTransport)
	}

	t.Logf("Agent: name=%s, description=%s", card.Name, card.Description)
}

func TestA2AMultipleMessages(t *testing.T) {
	env := setupMuxTestEnv(t)
	defer env.cleanup()

	transport, cleanup := env.connectA2A(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	numMessages := 5
	for i := 0; i < numMessages; i++ {
		msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{
			Text: strings.Repeat("test", i+1),
		})
		_, err := transport.SendMessage(ctx, &a2a.MessageSendParams{Message: msg})
		if err != nil {
			t.Fatalf("SendMessage %d: %v", i, err)
		}
	}

	t.Logf("Successfully sent %d A2A messages", numMessages)
}

func TestBothStreamsWork(t *testing.T) {
	env := setupMuxTestEnv(t)
	defer env.cleanup()

	// Connect to both streams
	controlClient, controlCleanup := env.connectControl(t)
	defer controlCleanup()

	a2aTransport, a2aCleanup := env.connectA2A(t)
	defer a2aCleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use Control stream
	pingResp, err := controlClient.Ping(ctx, &latisv1.PingRequest{
		Timestamp: time.Now().UnixNano(),
	})
	if err != nil {
		t.Fatalf("Control Ping: %v", err)
	}
	t.Logf("Control ping RTT: %v", time.Duration(pingResp.PongTimestamp-pingResp.PingTimestamp))

	// Use A2A stream
	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: "Hello from both streams test"})
	a2aResp, err := a2aTransport.SendMessage(ctx, &a2a.MessageSendParams{Message: msg})
	if err != nil {
		t.Fatalf("A2A SendMessage: %v", err)
	}
	t.Logf("A2A response type: %T", a2aResp)

	t.Log("Both Control and A2A streams work independently")
}

func TestCleanupDoesNotHang(t *testing.T) {
	env := setupMuxTestEnv(t)

	// Connect and do some work
	client, clientCleanup := env.connectControl(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx, &latisv1.PingRequest{
		Timestamp: time.Now().UnixNano(),
	})
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}

	// Close client connection
	clientCleanup()

	// Cleanup should complete within 5 seconds
	// If listener.Close() isn't called before GracefulStop(), this will hang
	done := make(chan struct{})
	go func() {
		env.cleanup()
		close(done)
	}()

	select {
	case <-done:
		t.Log("Cleanup completed successfully")
	case <-time.After(5 * time.Second):
		t.Fatal("Cleanup hung - listener.Close() must be called before GracefulStop()")
	}
}

func TestWrongCleanupOrderHangs(t *testing.T) {
	// This test verifies that calling GracefulStop() BEFORE listener.Close() hangs.
	// It demonstrates the bug that was in cmd/latis-unit/main.go.

	ca, _ := pki.GenerateCA()
	serverCert, _ := pki.GenerateCert(ca, pki.UnitIdentity("test"), true, false)
	serverTLS, _ := pki.ServerTLSConfig(serverCert, ca)

	listener, err := quictransport.ListenMux("127.0.0.1:0", serverTLS, nil)
	if err != nil {
		t.Fatalf("ListenMux: %v", err)
	}

	controlServer := grpc.NewServer()
	latisv1.RegisterControlServiceServer(controlServer, &testControlServer{
		identity:   "test",
		shutdownCh: make(chan struct{}, 1),
		startTime:  time.Now(),
	})

	go controlServer.Serve(listener.ControlListener())

	// Wait for server to start accepting
	time.Sleep(50 * time.Millisecond)

	// Try the WRONG cleanup order (GracefulStop before listener.Close)
	done := make(chan struct{})
	go func() {
		controlServer.GracefulStop() // This will hang!
		listener.Close()
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("Expected wrong cleanup order to hang, but it completed")
	case <-time.After(1 * time.Second):
		// Force cleanup - must close listener first to unblock GracefulStop
		listener.Close()
		<-done // Wait for the goroutine to finish
	}
}
