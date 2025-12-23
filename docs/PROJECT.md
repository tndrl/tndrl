# Project Tracking

Current focus and progress for Latis development.

## Current Objective

**Make units A2A-compatible with multiplexed QUIC transport**

Units should serve two protocols over separate QUIC streams:
- **Control stream** (type=0x01): Lifecycle, health checks via `ControlService`
- **A2A stream** (type=0x02): Agent communication via `a2a.v1.A2AService`

## Completed

### A2A Integration (PR #4 - merged)
- [x] Added `github.com/a2aproject/a2a-go` dependency
- [x] Created `pkg/a2aexec` with `AgentExecutor` implementation
- [x] `RegisterWithGRPC()` wires A2A services to gRPC server
- [x] Updated CLAUDE.md with toolbox development instructions

### Multiplexed QUIC Transport (PR #5 - merged)
- [x] `pkg/transport/quic/stream_type.go` — StreamType constants (Control=0x01, A2A=0x02)
- [x] `pkg/transport/quic/stream_conn.go` — Wraps QUIC stream as net.Conn
- [x] `pkg/transport/quic/mux.go` — MuxConn for typed stream open/accept
- [x] `pkg/transport/quic/mux_listener.go` — Routes streams to type-specific listeners
- [x] `pkg/transport/quic/mux_dialer.go` — Connection pooling, typed stream dialers
- [x] `pkg/transport/quic/mux_test.go` — Tests for routing and connection reuse

### Control Protocol (PR #5 - merged)
- [x] `proto/latis/v1/control.proto` — ControlService (Ping, GetStatus, Shutdown)
- [x] Generated code in `gen/go/latis/v1/control*.go`

### Test Infrastructure (PR #6 - merged)
- [x] Added `go.uber.org/goleak` for goroutine leak detection
- [x] `TestMain` with goleak in all test packages (pki, quic, a2aexec, integration)
- [x] `Makefile` with test targets (`test`, `test-verbose`, `test-cover`, `test-unit`, `test-integration`)
- [x] Race detection enabled by default (`-race` flag)
- [x] CI updated with parallel jobs (lint, test, build)
- [x] Branch ruleset updated to require all three checks

### Unit Multiplexed Transport (PR #10 - merged)
- [x] `pkg/control/state.go` — Unit state tracking (STARTING, READY, BUSY, DRAINING, STOPPED)
- [x] `pkg/control/control.go` — ControlServiceServer implementation (Ping, GetStatus, Shutdown)
- [x] `pkg/control/*_test.go` — Unit tests for control package
- [x] Refactored `cmd/latis-unit/main.go` to use `MuxListener`
- [x] Unit struct encapsulates both gRPC servers, state, and lifecycle
- [x] Signal handling for graceful shutdown (SIGINT, SIGTERM)
- [x] Shutdown RPC triggers graceful termination

## Next Steps

1. **Update cmdr to use multiplexed dialer**
   - Use `MuxDialer` for connection management
   - Create separate gRPC clients for Control and A2A
   - Health checks via Control stream
   - Agent interaction via A2A stream

2. **Integration tests for multiplexed transport**
   - Test both streams work independently
   - Test connection reuse across stream types
   - Test graceful shutdown via Control stream

3. **Cleanup legacy transport code**
   - Delete `pkg/transport/quic/dialer.go` (single-stream, replaced by mux_dialer)
   - Delete `pkg/transport/quic/listener.go` (single-stream, replaced by mux_listener)
   - Delete `pkg/transport/quic/conn.go` (replaced by stream_conn)
   - Update `pkg/transport/quic/README.md` for multiplexed API

4. **Remove unused interface packages**
   - Delete `pkg/dialer/` (interface not implemented by QUIC transport)
   - Delete `pkg/listener/` (interface not implemented by QUIC transport)
   - Delete `pkg/connector/` (unused abstraction)

## Architecture

```
QUIC Connection (cmdr ↔ unit)
│
├── Stream (type=0x01): Control
│   └── gRPC ControlService
│       ├── Ping — health check, latency measurement
│       ├── GetStatus — unit state, active tasks
│       └── Shutdown — graceful termination
│
└── Stream (type=0x02): A2A
    └── gRPC a2a.v1.A2AService
        ├── SendMessage — send prompt, get response
        ├── SendStreamingMessage — streaming response
        ├── GetTask — query task status
        └── CancelTask — cancel in-progress task
```

## Design Decisions

- **Stream type byte**: First byte of each stream identifies its purpose. Simple, extensible.
- **Separate gRPC servers**: Each stream type gets its own gRPC server for isolation.
- **Connection pooling**: MuxDialer reuses QUIC connections, opens new streams as needed.
- **A2A protocol**: Use a2a-go's implementation rather than custom protocol. Interoperability with broader agent ecosystem.
- **Control protocol**: Latis-specific for lifecycle management. Not part of A2A spec.
