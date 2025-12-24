# Transport

Network transport layer for Latis node communication.

## Current Implementation

Latis uses QUIC with mTLS for all node-to-node communication. This is implemented in `pkg/transport/quic/`.

```
┌────────────┐                    ┌────────────┐
│   latis    │                    │   latis    │
│   (client) │                    │   (server) │
└─────┬──────┘                    └─────┬──────┘
      │                                 │
      │ ←── QUIC + mTLS connection ───► │
      │                                 │
      │    ┌── Control stream ──────►   │
      │    └── A2A stream ──────────►   │
      │                                 │
```

## Key Components

### MuxDialer (Client)

Manages outbound connections with connection pooling.

```go
import quictransport "github.com/shanemcd/latis/pkg/transport/quic"

// Create dialer with TLS config
muxDialer := quictransport.NewMuxDialer(tlsConfig, nil)
defer muxDialer.Close()

// Get Control stream dialer
controlDialer := muxDialer.ControlDialer()

// Get A2A stream dialer
a2aDialer := muxDialer.A2ADialer()

// Both reuse the same underlying QUIC connection
```

### MuxListener (Server)

Accepts inbound connections and routes streams by type.

```go
import quictransport "github.com/shanemcd/latis/pkg/transport/quic"

// Start listener
listener, err := quictransport.ListenMux(addr, tlsConfig, nil)
defer listener.Close()

// Get type-specific listeners for gRPC servers
controlListener := listener.ControlListener()
a2aListener := listener.A2AListener()

// Route to separate gRPC servers
go controlServer.Serve(controlListener)
go a2aServer.Serve(a2aListener)
```

## Files

| File | Purpose |
|------|---------|
| `stream_type.go` | StreamType constants (Control=0x01, A2A=0x02) |
| `stream_conn.go` | Wraps QUIC stream as net.Conn |
| `mux.go` | MuxConn for typed stream open/accept |
| `mux_listener.go` | Server-side stream routing |
| `mux_dialer.go` | Client-side connection pooling |

## Design Decisions

### Why QUIC?

- **Multiplexed streams** — no head-of-line blocking
- **Built-in TLS 1.3** — encrypted by default
- **Connection migration** — survives network changes
- **Faster handshake** — 0-RTT in many cases

### Why direct connections?

The current design uses direct QUIC connections between nodes. This is simple and works well for:

- Local development (both nodes on same machine)
- Direct network access (nodes can reach each other)
- Controlled environments (VPN, overlay network)

### Connection Pooling

`MuxDialer` maintains a connection pool to avoid repeated handshakes:

```go
// Multiple calls reuse the same connection
controlConn1 := muxDialer.ControlDialer()(ctx, addr)
controlConn2 := muxDialer.ControlDialer()(ctx, addr)  // Same QUIC conn
a2aConn := muxDialer.A2ADialer()(ctx, addr)           // Still same conn
```

## Future Considerations

### NAT Traversal

For nodes behind NAT, potential approaches:

- **Relay nodes** — route through publicly accessible nodes
- **STUN/TURN** — standard NAT traversal protocols
- **Dial-back** — server initiates connection when client can't

### Alternative Transports

The gRPC layer is transport-agnostic. Future transports could include:

- **WebSocket** — for browser clients or firewall traversal
- **SSH** — for environments with existing SSH access
- **Container exec** — for containerized agents

These would implement the same stream multiplexing pattern.
