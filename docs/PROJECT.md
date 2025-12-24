# Project Tracking

Current focus and progress for Latis development.

## Current Objective

**Add tool execution framework**

LLM integration complete (Ollama). Next: enable agents to execute tools and maintain conversation context.

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

### Unified Binary with LLM Providers (PR #19 - merged)
- [x] Merged `latis` (cmdr) and `latis-unit` into single `latis` binary
- [x] Kong CLI framework with subcommands (serve, ping, status, prompt, discover, shutdown)
- [x] Config loading with reflection-based merge (CLI > config file > defaults)
- [x] Versioned config schema (v1)
- [x] Peer-to-peer architecture (any node can serve and connect)
- [x] Named peer configuration (`latis prompt local "Hello"`)
- [x] AgentCard discovery via `latis discover`
- [x] Pluggable LLM providers (echo, ollama) - requires explicit `--llm-provider`
- [x] Example configs: `examples/echo.yaml`, `examples/ollama.yaml`
- [x] Deleted `cmd/latis-unit/` (merged into `cmd/latis/`)
- [x] Updated README.md, CLAUDE.md with new architecture
- [x] Created docs/configuration.md and docs/cli.md

### Documentation Audit & Unit→Node Rename (PR #21 - merged)
- [x] Rewrote `docs/protobuf.md` for current `control.proto` and A2A integration
- [x] Rewrote `pkg/pki/README.md` with correct CLI flags and cert filenames
- [x] Rewrote `docs/design/protocol.md` to describe gRPC over QUIC stack
- [x] Renamed `docs/design/connector.md` → `transport.md`
- [x] Renamed `docs/design/unit.md` → `server.md`
- [x] Renamed `docs/design/cmdr.md` → `cli.md`
- [x] Updated `docs/design/a2a-alignment.md` and `policy.md` terminology
- [x] Fixed `docs/configuration.md` env vars and added missing PKI vars
- [x] Deleted unused `pkg/protocol` and `pkg/provisioner` stubs
- [x] Deleted `docs/design/execution-model.md` (contained outdated examples)
- [x] **Breaking**: Renamed `UnitState` → `NodeState` in proto
- [x] **Breaking**: Changed SPIFFE URIs from `spiffe://latis/unit/*` to `spiffe://latis/node/*`
- [x] Renamed `UnitIdentity` → `NodeIdentity` (kept deprecated alias)
- [x] Fixed undefined `pki.CmdrIdentity` bug in tests

### Structured Logging (PR #22)
- [x] Created `cmd/latis/logging.go` with `setupLogger()` for slog configuration
- [x] Added `--log-level` flag (debug, info, warn, error) and `-v`/`--verbose` shortcut
- [x] Migrated all `log.Printf/Println` to structured `slog` calls in cmd/latis/
- [x] Added logging to `pkg/control/` (RPC handlers, state transitions)
- [x] Added logging to `pkg/a2aexec/` (message execution, errors, cancellation)
- [x] Added logging to `pkg/llm/ollama.go` (API requests, errors)
- [x] Added logging to `pkg/transport/quic/` (connection lifecycle, stream routing)
- [x] Updated `docs/cli.md` with `--log-level` flag
- [x] Updated `docs/configuration.md` with `logLevel` config option

## Next Steps

1. **Tool execution framework**
   - Define tool interface for agents
   - Tool calling via LLM function calling
   - Conversation context/history

2. **Dynamic peer discovery**
   - DNS SRV records
   - Multicast/broadcast discovery

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         latis node                          │
│                                                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────┐ │
│  │   A2A       │    │   Control   │    │      LLM        │ │
│  │   Server    │    │   Server    │    │    Provider     │ │
│  └─────────────┘    └─────────────┘    └─────────────────┘ │
│         │                  │                    │          │
│         └──────────────────┼────────────────────┘          │
│                            │                               │
│                    ┌───────┴───────┐                       │
│                    │  QUIC/mTLS    │                       │
│                    └───────────────┘                       │
└─────────────────────────────────────────────────────────────┘

QUIC Connection (peer ↔ peer)
│
├── Stream (type=0x01): Control
│   └── gRPC ControlService
│       ├── Ping — health check, latency measurement
│       ├── GetStatus — node state, active tasks
│       └── Shutdown — graceful termination
│
└── Stream (type=0x02): A2A
    └── gRPC a2a.v1.A2AService
        ├── GetAgentCard — discover capabilities
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
