# CLAUDE.md

Guidance for Claude Code when working in this repository.

## What is Latis?

A control plane for distributed AI agents. Transport-agnostic, agent-agnostic, pluggable everything.

## Architecture

```
cmdr → connector → unit → [agent]
```

| Component | Purpose | Details |
|-----------|---------|---------|
| **cmdr** | Human interface, orchestration, provisioning | [cmdr/README.md](./cmdr/README.md) |
| **connector** | Transport layer (SSH, WebSocket, etc.) | [connector/README.md](./connector/README.md) |
| **unit** | Agent endpoint daemon | [unit/README.md](./unit/README.md) |
| **protocol** | Wire format, message types | [protocol/README.md](./protocol/README.md) |

## Key Design Decisions

- **Go** for the core (cmdr, unit, connectors) — single binary, easy cross-compilation, good concurrency
- **gRPC with protobuf** — type-safe, bidirectional streaming, code generation
- **QUIC transport** — modern, multiplexed, encrypted by default
- **buf for protobuf tooling** — linting, code generation, no exceptions
- **Bidirectional connections** — cmdr can dial out, units can dial in
- **Pluggable provisioners** — process, container, VM, cloud (design phase)
- **Polyglot plugins** — connectors and agent adapters can be any language

## Code Structure

```
latis/
├── cmd/
│   ├── latis/           # cmdr CLI
│   └── latis-unit/      # unit daemon
├── gen/go/latis/v1/     # generated protobuf/gRPC code
├── pkg/
│   ├── transport/quic/  # QUIC transport for gRPC
│   ├── protocol/        # (placeholder)
│   ├── connector/       # (placeholder)
│   └── ...
├── proto/latis/v1/      # protobuf definitions
├── buf.yaml             # buf configuration
└── buf.gen.yaml         # code generation config
```

## Quickstart

```bash
# Terminal 1: Start unit
go run ./cmd/latis-unit/

# Terminal 2: Send a prompt
go run ./cmd/latis/ -prompt "Hello, World!"

# Or just ping
go run ./cmd/latis/
```

## Design Documents

Open design areas and future considerations:

- [Execution Model](./docs/design/execution-model.md) — tools, autonomy, yield points
- [Policy](./docs/design/policy.md) — authorization, OPA integration (deferred)

## Status

Core loop working: cmdr ↔ unit over gRPC/QUIC with bidirectional streaming.

## When Working Here

1. Read the component README for context before making changes
2. Capture design decisions in documentation as they're made
3. Keep docs minimal — prefer pointers over duplication
