# Project Tracking

Current focus and progress for Tndrl development.

## Current Objective

**Session management and environment provisioning**

Enable orchestrating isolated agent environments with full lifecycle management. See [docs/design/sessions.md](./design/sessions.md) for the design.

Core use case: "turn loose" an AI agent in a sandboxed container where it can iterate autonomously until it needs human input.

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
- [x] `proto/tndrl/v1/control.proto` — ControlService (Ping, GetStatus, Shutdown)
- [x] Generated code in `gen/go/tndrl/v1/control*.go`

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
- [x] Refactored `cmd/tndrl-unit/main.go` to use `MuxListener`
- [x] Unit struct encapsulates both gRPC servers, state, and lifecycle
- [x] Signal handling for graceful shutdown (SIGINT, SIGTERM)
- [x] Shutdown RPC triggers graceful termination

### Cmdr Multiplexed Transport (PR #12 - merged)
- [x] Refactored `cmd/tndrl/main.go` to use `MuxDialer`
- [x] Control gRPC client via `muxDialer.ControlDialer()`
- [x] `--status` flag for GetStatus RPC
- [x] `--shutdown` flag for Shutdown RPC
- [x] Default ping via ControlService.Ping
- [x] Removed old TndrlService bidirectional stream dependency
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
- [x] Removed legacy TndrlService integration tests

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
- [x] Deleted `proto/tndrl/v1/tndrl.proto` (TndrlService no longer used)
- [x] Deleted generated `tndrl.pb.go` and `tndrl_grpc.pb.go`

### Unified Binary with LLM Providers (PR #19 - merged)
- [x] Merged `tndrl` (cmdr) and `tndrl-unit` into single `tndrl` binary
- [x] Kong CLI framework with subcommands (serve, ping, status, prompt, discover, shutdown)
- [x] Config loading with reflection-based merge (CLI > config file > defaults)
- [x] Versioned config schema (v1)
- [x] Peer-to-peer architecture (any node can serve and connect)
- [x] Named peer configuration (`tndrl prompt local "Hello"`)
- [x] AgentCard discovery via `tndrl discover`
- [x] Pluggable LLM providers (echo, ollama) - requires explicit `--llm-provider`
- [x] Example configs: `examples/echo.yaml`, `examples/ollama.yaml`
- [x] Deleted `cmd/tndrl-unit/` (merged into `cmd/tndrl/`)
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
- [x] **Breaking**: Changed SPIFFE URIs from `spiffe://tndrl/unit/*` to `spiffe://tndrl/node/*`
- [x] Renamed `UnitIdentity` → `NodeIdentity` (kept deprecated alias)
- [x] Fixed undefined `pki.CmdrIdentity` bug in tests

### Structured Logging (PR #23 - merged)
- [x] Created `cmd/tndrl/logging.go` with `setupLogger()` for slog configuration
- [x] Added `--log-level` flag (debug, info, warn, error) and `-v`/`--verbose` shortcut
- [x] Migrated all `log.Printf/Println` to structured `slog` calls in cmd/tndrl/
- [x] Added logging to `pkg/control/` (RPC handlers, state transitions)
- [x] Added logging to `pkg/a2aexec/` (message execution, errors, cancellation)
- [x] Added logging to `pkg/llm/ollama.go` (API requests, errors)
- [x] Added logging to `pkg/transport/quic/` (connection lifecycle, stream routing)
- [x] Updated `docs/cli.md` with `--log-level` flag
- [x] Updated `docs/configuration.md` with `logLevel` config option

### MCP Tool Integration (PR #25 - merged)
- [x] Added `github.com/mark3labs/mcphost` dependency
- [x] Created `pkg/llm/mcphost.go` with `MCPHostProvider` implementation
- [x] Extended `LLMConfig` with `mcpServers`, `systemPrompt`, `maxSteps` fields
- [x] New provider type `mcphost` with full tool calling support
- [x] Supports builtin (fs, bash, todo, http), local (stdio), and remote (HTTP) MCP servers
- [x] Added `mcpConfigFile` option for external mcphost config files
- [x] Added `examples/mcphost.yaml` configuration example
- [x] Updated `docs/configuration.md` with MCP server configuration

## Next Steps

1. **Session management** (see [design/sessions.md](./design/sessions.md))
   - Session lifecycle: create, list, attach, delete
   - Conversation history persistence
   - "Waiting for input" protocol

2. **Environment drivers**
   - Podman driver (first)
   - Local driver (for testing)
   - Kubernetes driver
   - QEMU driver (stretch)

3. **Workspace images**
   - Base image with tndrl + shell tools
   - Language-specific variants (go, python, node)

4. **Deferred: Dynamic peer discovery**
   - DNS SRV records
   - Multicast/broadcast discovery

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         tndrl node                          │
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
- **Control protocol**: Tndrl-specific for lifecycle management. Not part of A2A spec.
