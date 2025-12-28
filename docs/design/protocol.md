# Protocol

How Tndrl nodes communicate over the network.

## Protocol Stack

Tndrl uses a layered protocol stack:

```
┌─────────────────────────────────────────┐
│              Application                │
│   (A2A messages, Control messages)      │
├─────────────────────────────────────────┤
│               gRPC                      │
│   (request/response, streaming)         │
├─────────────────────────────────────────┤
│           Stream Multiplexing           │
│   (Control=0x01, A2A=0x02)              │
├─────────────────────────────────────────┤
│            QUIC + mTLS                  │
│   (encryption, connection mgmt)         │
└─────────────────────────────────────────┘
```

## Stream Types

Each QUIC connection carries multiple streams. The first byte of each stream identifies its type:

| Type | Value | Purpose |
|------|-------|---------|
| Control | `0x01` | Node lifecycle (ping, status, shutdown) |
| A2A | `0x02` | Agent communication (prompts, tasks) |

This allows:
- Protocol isolation between control and agent traffic
- Independent gRPC servers for each stream type
- Future extensibility (add new stream types)

## Connection Flow

```
Client                                    Server
  │                                         │
  │──── QUIC handshake (mTLS) ─────────────►│
  │◄─── Connection established ─────────────│
  │                                         │
  │──── Open stream, write 0x01 ───────────►│  Control stream
  │◄─── gRPC ControlService ready ──────────│
  │                                         │
  │──── Open stream, write 0x02 ───────────►│  A2A stream
  │◄─── gRPC A2AService ready ──────────────│
  │                                         │
```

## Control Protocol

Runs on stream type `0x01`. Defined in `proto/tndrl/v1/control.proto`.

| RPC | Purpose |
|-----|---------|
| `Ping` | Health check, latency measurement |
| `GetStatus` | Query node state, uptime, active tasks |
| `Shutdown` | Request graceful or immediate shutdown |

See [docs/protobuf.md](../protobuf.md) for details.

## A2A Protocol

Runs on stream type `0x02`. Uses the [A2A protocol](https://a2a-protocol.org/) via `a2a-go`.

| RPC | Purpose |
|-----|---------|
| `GetAgentCard` | Discover agent capabilities |
| `SendMessage` | Send prompt, receive response |
| `SendStreamingMessage` | Stream response chunks |
| `GetTask` | Query task status |
| `CancelTask` | Cancel in-progress task |

See the [A2A spec](https://a2a-protocol.org/latest/) for details.

## Connection Multiplexing

Multiple streams share a single QUIC connection:

```
QUIC Connection (peer ↔ peer)
│
├── Stream (type=0x01): Control
│   └── gRPC ControlService
│
└── Stream (type=0x02): A2A
    └── gRPC A2AService
```

Benefits:
- Single connection setup (one mTLS handshake)
- No head-of-line blocking between protocols
- Connection reuse across multiple requests

Implementation: `pkg/transport/quic/` (`MuxDialer`, `MuxListener`)

## Security

All connections use mTLS (mutual TLS):

- **TLS 1.3** via QUIC
- **Client authentication** — both sides present certificates
- **CA verification** — certificates must chain to trusted CA
- **SPIFFE-compatible** — identity URIs in certificate SANs

See [pkg/pki/README.md](../../pkg/pki/README.md) for certificate management.

## Design Decisions

### Why gRPC over QUIC?

- **Structured RPC** — request/response semantics, streaming, code generation
- **Multiplexing** — QUIC streams allow protocol separation without multiple connections
- **Modern transport** — connection migration, 0-RTT, no head-of-line blocking
- **mTLS built-in** — QUIC requires TLS

### Why stream type prefix?

- **Simple** — one byte overhead per stream
- **Extensible** — add new stream types without changing existing code
- **Debuggable** — easy to identify stream purpose in packet captures

### Why separate Control and A2A?

- **Isolation** — control operations don't interfere with agent traffic
- **Different concerns** — control is infrastructure, A2A is application
- **Independent evolution** — A2A follows upstream spec, control is Tndrl-specific
