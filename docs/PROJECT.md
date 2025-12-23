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

### Cmdr Multiplexed Transport (PR #12 - merged)
- [x] Refactored `cmd/latis/main.go` to use `MuxDialer`
- [x] Control gRPC client via `muxDialer.ControlDialer()`
- [x] `--status` flag for GetStatus RPC
- [x] `--shutdown` flag for Shutdown RPC
- [x] Default ping via ControlService.Ping
- [x] Removed old LatisService bidirectional stream dependency
- [x] Integration tests for multiplexed Control stream (Ping, GetStatus, Shutdown)
- [x] Connection reuse tests for MuxDialer

### Legacy Cleanup (PR #14 - merged)
- [x] Deleted `pkg/transport/quic/dialer.go` (replaced by mux_dialer)
- [x] Deleted `pkg/transport/quic/listener.go` (replaced by mux_listener)
- [x] Deleted `pkg/transport/quic/conn.go` (replaced by stream_conn)
- [x] Updated `pkg/transport/quic/README.md` for multiplexed API
- [x] Deleted `pkg/dialer/` (unused interface)
- [x] Deleted `pkg/listener/` (unused interface)
- [x] Deleted `pkg/connector/` (unused abstraction)
- [x] Removed legacy LatisService integration tests

### A2A End-to-End (PR #15 - merged)
- [x] A2A executor already wired in unit (from PR #10)
- [x] Added `--prompt` and `--stream` flags to cmdr
- [x] A2A client via `muxDialer.A2ADialer()` and `a2aclient.NewGRPCTransport`
- [x] `doPrompt()` sends message via A2A SendMessage
- [x] `doStreamingPrompt()` uses A2A SendStreamingMessage
- [x] Integration tests: TestA2ASendMessage, TestA2AGetAgentCard, TestA2AMultipleMessages
- [x] TestBothStreamsWork verifies Control and A2A work independently
- [x] Fixed shutdown hang with `stopServers()` helper (close listener before GracefulStop)
- [x] Added cleanup ordering regression tests
- [x] Added IP SANs (127.0.0.1, ::1) to generated certificates
- [x] Unit listens on `[::]:4433` by default for dual-stack IPv4/IPv6 support

### Legacy Proto Cleanup (PR #18 - merged)
- [x] Deleted `proto/latis/v1/latis.proto` (LatisService no longer used)
- [x] Deleted generated `latis.pb.go` and `latis_grpc.pb.go`

## Next Steps

1. **Add real agent execution**
   - Replace echo handler with actual LLM integration
   - Tool execution framework

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
