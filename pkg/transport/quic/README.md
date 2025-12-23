# QUIC Transport

Multiplexed QUIC transport for gRPC with typed streams.

## Why QUIC?

- **Multiplexed streams** — no head-of-line blocking
- **Built-in TLS 1.3** — encrypted by default
- **Connection migration** — survives network changes
- **Faster handshake** — 0-RTT in many cases

## Architecture

Latis uses multiplexed QUIC streams to separate Control and A2A protocols:

```
QUIC Connection
│
├── Stream (type=0x01): Control
│   └── gRPC ControlService (Ping, GetStatus, Shutdown)
│
└── Stream (type=0x02): A2A
    └── gRPC a2a.v1.A2AService
```

Each stream type gets its own gRPC server/client, providing protocol isolation.

## Usage

### Server (MuxListener)

```go
import quictransport "github.com/shanemcd/latis/pkg/transport/quic"

// Start multiplexed listener
listener, err := quictransport.ListenMux(addr, tlsConfig, nil)
if err != nil {
    log.Fatal(err)
}
defer listener.Close()

// Serve Control protocol on Control streams
controlServer := grpc.NewServer()
latisv1.RegisterControlServiceServer(controlServer, controlHandler)
go controlServer.Serve(listener.ControlListener())

// Serve A2A protocol on A2A streams
a2aServer := grpc.NewServer()
a2a.RegisterA2AServiceServer(a2aServer, a2aHandler)
go a2aServer.Serve(listener.A2AListener())
```

### Client (MuxDialer)

```go
import quictransport "github.com/shanemcd/latis/pkg/transport/quic"

// Create multiplexed dialer (reuses QUIC connections)
muxDialer := quictransport.NewMuxDialer(tlsConfig, nil)
defer muxDialer.Close()

// Connect to Control service
controlConn, err := grpc.NewClient(
    addr,
    grpc.WithContextDialer(muxDialer.ControlDialer()),
    grpc.WithTransportCredentials(insecure.NewCredentials()),
)
controlClient := latisv1.NewControlServiceClient(controlConn)

// Connect to A2A service (reuses same QUIC connection)
a2aConn, err := grpc.NewClient(
    addr,
    grpc.WithContextDialer(muxDialer.A2ADialer()),
    grpc.WithTransportCredentials(insecure.NewCredentials()),
)
a2aClient := a2a.NewA2AServiceClient(a2aConn)
```

## Files

- `stream_type.go` — StreamType constants (Control=0x01, A2A=0x02)
- `stream_conn.go` — Wraps QUIC stream as net.Conn
- `mux.go` — MuxConn for typed stream open/accept
- `mux_listener.go` — Routes streams to type-specific listeners
- `mux_dialer.go` — Connection pooling, typed stream dialers
- `mux_test.go` — Tests for routing and connection reuse
