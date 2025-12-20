# Latis

A control plane for distributed AI agents.

Latis provides a unified interface for orchestrating AI agents running across multiple machines, containers, and environments. It's transport-agnostic, agent-agnostic, and designed to scale from a single local agent to coordinated fleets.

## Vision

```
latis connect prod-server
latis session new --transport ssh://gpu-box
latis prompt "analyze the deployment logs"
latis coordinate task-123 --agents prod-1,prod-2,dev-local
```

## Architecture

```
┌──────────────────┐
│      cmdr        │  ← CLI + orchestration brain
└────────┬─────────┘
         │
    ┌────┴────┐
    │connector│  ← transport plugins (ssh, ws, local, etc.)
    └────┬────┘
         │
┌────────┴─────────┐
│       unit       │  ← agent endpoint (runs on remote/local)
└──────────────────┘
```

### Components

- **[cmdr](./cmdr/)** — The control plane. CLI interface, orchestration, session management, agent coordination. This is what users interact with.

- **[connector](./connector/)** — Transport abstraction layer. Pluggable modules that know how to move bytes between cmdr and units. SSH, WebSocket, local process, container exec — each is a connector plugin.

- **[unit](./unit/)** — The endpoint daemon. Runs wherever agents live. Receives protocol messages, executes work (wrapping any underlying AI agent), streams responses back. Lightweight and embeddable.

- **[protocol](./protocol/)** — The wire protocol. Protobuf schemas for type safety, length-prefixed framing for transport flexibility. Fully multiplexed and async.

## Design Principles

- **Transport agnostic**: SSH today, WebSocket tomorrow, carrier pigeon if you write the plugin
- **Agent agnostic**: No opinions on what runs at the endpoints
- **Protocol-first**: A well-defined contract that any language can implement
- **Pluggable everything**: Transports, agents, authentication, storage

## Core Protocol

See **[protocol/](./protocol/)** for full details.

Key decisions:
- **Protobuf schemas** for type safety and code generation
- **Length-prefixed framing** (not HTTP/2) so it works over any byte stream
- **Fully multiplexed** — messages have IDs, either side can send anytime, control messages interleave with data
- **gRPC upgrade path** for transports that support it (WebSocket, TCP, QUIC)

## Transports

Transports are pluggable. See [pkg/transport/](./pkg/transport/) for implementations.

Currently implemented:
- **[QUIC](./pkg/transport/quic/)**: gRPC over QUIC — multiplexed streams, built-in TLS 1.3, connection migration

Planned:
- **SSH**: Execute commands on remote hosts
- **Local**: Direct process communication
- **Container**: Podman/Docker exec
- **WebSocket**: Persistent bidirectional connections

## Quickstart

```bash
# Terminal 1: Start unit
go run ./cmd/latis-unit/

# Terminal 2: Send a ping
go run ./cmd/latis/

# Or send a prompt
go run ./cmd/latis/ -prompt "Hello, World!"
```

## Status

Core loop working: cmdr ↔ unit over gRPC/QUIC with bidirectional streaming.

## Name

Latis: from "lattice" — a structure of interconnected points. Agents connected across a distributed mesh.

Or if you prefer acronyms: **L**inked **A**gent **T**ransport & **I**nterconnection **S**ystem.

## License

TBD
