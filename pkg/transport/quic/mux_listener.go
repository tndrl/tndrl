package quic

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/quic-go/quic-go"
)

// MuxListener accepts QUIC connections and routes streams by type.
// It provides separate net.Listener interfaces for each stream type,
// allowing different gRPC servers to handle different stream types.
type MuxListener struct {
	ql        *quic.Listener
	tlsConfig *tls.Config

	mu      sync.Mutex
	closed  bool
	conns   map[*MuxConn]struct{}
	streams map[StreamType]chan net.Conn
	errors  chan error

	ctx    context.Context
	cancel context.CancelFunc
}

// ListenMux creates a new multiplexed QUIC listener.
func ListenMux(addr string, tlsConfig *tls.Config, quicConfig *quic.Config) (*MuxListener, error) {
	ql, err := quic.ListenAddr(addr, tlsConfig, quicConfig)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	l := &MuxListener{
		ql:        ql,
		tlsConfig: tlsConfig,
		conns:     make(map[*MuxConn]struct{}),
		streams: map[StreamType]chan net.Conn{
			StreamTypeControl: make(chan net.Conn, 16),
			StreamTypeA2A:     make(chan net.Conn, 16),
		},
		errors: make(chan error, 8),
		ctx:    ctx,
		cancel: cancel,
	}

	go l.acceptLoop()

	return l, nil
}

// acceptLoop accepts QUIC connections and spawns stream handlers.
func (l *MuxListener) acceptLoop() {
	for {
		qconn, err := l.ql.Accept(l.ctx)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				select {
				case l.errors <- fmt.Errorf("accept connection: %w", err):
				default:
				}
			}
			return
		}

		muxConn := NewMuxConn(qconn)

		l.mu.Lock()
		if l.closed {
			l.mu.Unlock()
			muxConn.Close()
			return
		}
		l.conns[muxConn] = struct{}{}
		l.mu.Unlock()

		go l.handleConnection(muxConn)
	}
}

// handleConnection accepts streams from a connection and routes them by type.
func (l *MuxListener) handleConnection(muxConn *MuxConn) {
	slog.Debug("handling connection", "remote", muxConn.RemoteAddr())
	defer func() {
		slog.Debug("connection handler done", "remote", muxConn.RemoteAddr())
		l.mu.Lock()
		delete(l.conns, muxConn)
		l.mu.Unlock()
	}()

	for {
		conn, streamType, err := muxConn.AcceptStream(l.ctx)
		if err != nil {
			// Connection closed or context canceled
			if !errors.Is(err, context.Canceled) {
				slog.Debug("accept stream error", "err", err)
			}
			return
		}

		slog.Debug("stream accepted", "type", streamType, "remote", conn.RemoteAddr())

		l.mu.Lock()
		streamChan, ok := l.streams[streamType]
		l.mu.Unlock()

		if !ok {
			// Unknown stream type, close the stream
			slog.Warn("unknown stream type", "type", streamType)
			conn.Close()
			continue
		}

		select {
		case streamChan <- conn:
		case <-l.ctx.Done():
			conn.Close()
			return
		}
	}
}

// Listener returns a net.Listener for the given stream type.
// This can be passed to grpc.Server.Serve().
func (l *MuxListener) Listener(streamType StreamType) net.Listener {
	return &streamListener{
		mux:        l,
		streamType: streamType,
	}
}

// ControlListener returns a net.Listener for control streams.
func (l *MuxListener) ControlListener() net.Listener {
	return l.Listener(StreamTypeControl)
}

// A2AListener returns a net.Listener for A2A streams.
func (l *MuxListener) A2AListener() net.Listener {
	return l.Listener(StreamTypeA2A)
}

// Addr returns the listener's network address.
func (l *MuxListener) Addr() net.Addr {
	return l.ql.Addr()
}

// Close closes the listener and all connections.
func (l *MuxListener) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	l.cancel()

	// Close all stream channels
	for _, ch := range l.streams {
		close(ch)
	}

	// Close all connections
	for conn := range l.conns {
		conn.Close()
	}
	l.mu.Unlock()

	return l.ql.Close()
}

// streamListener implements net.Listener for a specific stream type.
type streamListener struct {
	mux        *MuxListener
	streamType StreamType
}

func (l *streamListener) Accept() (net.Conn, error) {
	l.mux.mu.Lock()
	streamChan := l.mux.streams[l.streamType]
	l.mux.mu.Unlock()

	select {
	case conn, ok := <-streamChan:
		if !ok {
			return nil, net.ErrClosed
		}
		return conn, nil
	case <-l.mux.ctx.Done():
		return nil, net.ErrClosed
	}
}

func (l *streamListener) Close() error {
	// Individual stream listeners don't close the whole mux
	return nil
}

func (l *streamListener) Addr() net.Addr {
	return l.mux.Addr()
}

var _ net.Listener = (*streamListener)(nil)
