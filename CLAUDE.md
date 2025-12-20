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
| **cmdr** | Human interface, orchestration, provisioning | [docs/design/cmdr.md](./docs/design/cmdr.md) |
| **connector** | Transport layer (SSH, WebSocket, etc.) | [docs/design/connector.md](./docs/design/connector.md) |
| **unit** | Agent endpoint daemon | [docs/design/unit.md](./docs/design/unit.md) |
| **protocol** | Wire format, message types | [docs/design/protocol.md](./docs/design/protocol.md) |

## Key Design Decisions

- **Go** for the core (cmdr, unit, connectors) — single binary, easy cross-compilation, good concurrency
- **gRPC with protobuf** — type-safe, bidirectional streaming, code generation
- **QUIC transport** — modern, multiplexed, encrypted by default
- **mTLS everywhere** — mutual TLS with built-in CA, SPIFFE-compatible identities
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
│   ├── pki/             # CA and certificate management (mTLS)
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
# Terminal 1: Initialize PKI and start unit
go run ./cmd/latis-unit/ --init-pki

# Terminal 2: Generate cmdr cert and connect
go run ./cmd/latis/ --init-pki

# Or send a prompt
go run ./cmd/latis/ -prompt "Hello, World!"
```

PKI files are stored in `~/.latis/pki/`. See [pkg/pki/README.md](./pkg/pki/README.md) for details.

## Documentation

- [Protobuf & buf](./docs/protobuf.md) — schema definitions, code generation, workflow

### Design Documents

Open design areas and future considerations:

- [Execution Model](./docs/design/execution-model.md) — tools, autonomy, yield points
- [Policy](./docs/design/policy.md) — authorization, OPA integration (deferred)

## Status

Core loop working: cmdr ↔ unit over gRPC/QUIC with bidirectional streaming.

## When Working Here

1. Read the component README for context before making changes
2. Capture design decisions in documentation as they're made
3. Keep docs minimal — prefer pointers over duplication
