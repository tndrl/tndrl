# QUIC Transport

Adapters that allow gRPC to run over QUIC.

## Why QUIC?

- **Multiplexed streams** — no head-of-line blocking
- **Built-in TLS 1.3** — encrypted by default
- **Connection migration** — survives network changes
- **Faster handshake** — 0-RTT in many cases

## How It Works

gRPC expects `net.Listener` (server) and `net.Conn` (client). This package wraps quic-go to provide these interfaces:

```
┌─────────────────────────────────────────────────────┐
│                      gRPC                           │
│            (expects net.Listener/net.Conn)          │
├─────────────────────────────────────────────────────┤
│                   This Package                      │
│  ┌─────────────────┐    ┌─────────────────────────┐ │
│  │ Listener        │    │ Conn                    │ │
│  │ wraps           │    │ wraps quic.Stream       │ │
│  │ quic.Listener   │    │ as net.Conn             │ │
│  └─────────────────┘    └─────────────────────────┘ │
├─────────────────────────────────────────────────────┤
│                    quic-go                          │
└─────────────────────────────────────────────────────┘
```

## Usage

### Server

```go
import quictransport "github.com/shanemcd/latis/pkg/transport/quic"

listener, err := quictransport.Listen(addr, tlsConfig, nil)
if err != nil {
    log.Fatal(err)
}

grpcServer := grpc.NewServer()
grpcServer.Serve(listener)
```

### Client

```go
import quictransport "github.com/shanemcd/latis/pkg/transport/quic"

dialer := quictransport.NewDialer(tlsConfig, nil)

conn, err := grpc.NewClient(
    addr,
    grpc.WithContextDialer(dialer),
    grpc.WithTransportCredentials(insecure.NewCredentials()),
)
```

## Files

- `listener.go` — `Listener` wraps `quic.Listener` as `net.Listener`
- `conn.go` — `Conn` wraps `quic.Stream` as `net.Conn`
- `dialer.go` — `Dial` and `NewDialer` for client connections
