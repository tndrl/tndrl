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
- **Protobuf schemas, not full gRPC** — works over any byte stream
- **Length-prefixed framing** — simple, transport-agnostic
- **Multiplexed protocol** — message IDs, async, interleaved control
- **Bidirectional connections** — cmdr can dial out, units can dial in
- **Pluggable provisioners** — process, container, VM, cloud
- **Polyglot plugins** — connectors and agent adapters can be any language

## Design Documents

Open design areas and future considerations:

- [Execution Model](./docs/design/execution-model.md) — tools, autonomy, yield points
- [Policy](./docs/design/policy.md) — authorization, OPA integration (deferred)

## Status

Early design phase. Documentation and code stubs exist.

## When Working Here

1. Read the component README for context before making changes
2. Capture design decisions in documentation as they're made
3. Keep docs minimal — prefer pointers over duplication
