package quic

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/quic-go/quic-go"
)

// MuxDialer manages multiplexed QUIC connections to remote endpoints.
// It maintains a single QUIC connection per address and creates typed streams on demand.
type MuxDialer struct {
	tlsConfig  *tls.Config
	quicConfig *quic.Config

	mu    sync.Mutex
	conns map[string]*MuxConn // addr -> connection
}

// NewMuxDialer creates a new multiplexed dialer.
func NewMuxDialer(tlsConfig *tls.Config, quicConfig *quic.Config) *MuxDialer {
	return &MuxDialer{
		tlsConfig:  tlsConfig,
		quicConfig: quicConfig,
		conns:      make(map[string]*MuxConn),
	}
}

// Dial opens a stream of the given type to the address.
// If a connection already exists, it reuses it; otherwise it creates a new one.
func (d *MuxDialer) Dial(ctx context.Context, addr string, streamType StreamType) (net.Conn, error) {
	muxConn, err := d.getOrCreateConn(ctx, addr)
	if err != nil {
		return nil, err
	}

	stream, err := muxConn.OpenStream(ctx, streamType)
	if err != nil {
		// Connection may be dead, remove it and retry once
		d.removeConn(addr)
		muxConn, err = d.getOrCreateConn(ctx, addr)
		if err != nil {
			return nil, err
		}
		stream, err = muxConn.OpenStream(ctx, streamType)
		if err != nil {
			return nil, fmt.Errorf("open stream after reconnect: %w", err)
		}
	}

	return stream, nil
}

// DialControl opens a control stream to the address.
func (d *MuxDialer) DialControl(ctx context.Context, addr string) (net.Conn, error) {
	return d.Dial(ctx, addr, StreamTypeControl)
}

// DialA2A opens an A2A stream to the address.
func (d *MuxDialer) DialA2A(ctx context.Context, addr string) (net.Conn, error) {
	return d.Dial(ctx, addr, StreamTypeA2A)
}

// getOrCreateConn returns an existing connection or creates a new one.
func (d *MuxDialer) getOrCreateConn(ctx context.Context, addr string) (*MuxConn, error) {
	d.mu.Lock()
	if muxConn, ok := d.conns[addr]; ok {
		d.mu.Unlock()
		slog.Debug("reusing connection", "addr", addr)
		return muxConn, nil
	}
	d.mu.Unlock()

	// Create new connection
	slog.Debug("establishing connection", "addr", addr)
	qconn, err := quic.DialAddr(ctx, addr, d.tlsConfig, d.quicConfig)
	if err != nil {
		slog.Debug("connection failed", "addr", addr, "err", err)
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}

	muxConn := NewMuxConn(qconn)

	d.mu.Lock()
	// Check again in case another goroutine created it
	if existing, ok := d.conns[addr]; ok {
		d.mu.Unlock()
		muxConn.Close() // Close the one we just created
		return existing, nil
	}
	d.conns[addr] = muxConn
	d.mu.Unlock()

	slog.Debug("connection established", "addr", addr)

	// Monitor connection for closure
	go func() {
		<-muxConn.Context().Done()
		slog.Debug("connection closed", "addr", addr)
		d.removeConn(addr)
	}()

	return muxConn, nil
}

// removeConn removes a connection from the pool.
func (d *MuxDialer) removeConn(addr string) {
	d.mu.Lock()
	delete(d.conns, addr)
	d.mu.Unlock()
}

// Close closes all connections.
func (d *MuxDialer) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	var lastErr error
	for addr, conn := range d.conns {
		if err := conn.Close(); err != nil {
			lastErr = err
		}
		delete(d.conns, addr)
	}
	return lastErr
}

// ControlDialer returns a function compatible with grpc.WithContextDialer for control streams.
func (d *MuxDialer) ControlDialer() func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, addr string) (net.Conn, error) {
		return d.DialControl(ctx, addr)
	}
}

// A2ADialer returns a function compatible with grpc.WithContextDialer for A2A streams.
func (d *MuxDialer) A2ADialer() func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, addr string) (net.Conn, error) {
		return d.DialA2A(ctx, addr)
	}
}
