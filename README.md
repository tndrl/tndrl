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
┌─────────────────────────────────────────────────────────┐
│                        CLI / API                        │
├─────────────────────────────────────────────────────────┤
│                   Orchestration Layer                   │
│          (coordination, routing, session mgmt)          │
├─────────────────────────────────────────────────────────┤
│                      Core Protocol                      │
│             (the contract everything speaks)            │
├─────────────────────────────────────────────────────────┤
│                    Transport Layer                      │
│         SSH │ WebSocket │ HTTP │ Local │ Container      │
├─────────────────────────────────────────────────────────┤
│                     Agent Adapters                      │
│              (any agent that speaks the protocol)       │
└─────────────────────────────────────────────────────────┘
```

## Design Principles

- **Transport agnostic**: SSH today, WebSocket tomorrow, carrier pigeon if you write the plugin
- **Agent agnostic**: No opinions on what runs at the endpoints
- **Protocol-first**: A well-defined contract that any language can implement
- **Pluggable everything**: Transports, agents, authentication, storage

## Core Protocol

The protocol defines how the control plane communicates with agents:

```
Messages (Controller → Agent):
  session.create      Create a new agent session
  session.resume      Reconnect to existing session
  session.destroy     Terminate session
  prompt.send         Send input (streaming response)
  prompt.cancel       Cancel in-progress operation
  state.get           Query agent state
  state.subscribe     Subscribe to state changes

Messages (Agent → Controller):
  response.chunk      Streaming output
  response.complete   Operation finished
  state.update        State change notification
  error               Error occurred
```

## Transports

Transports are pluggable. Initial targets:

- **SSH**: Execute commands on remote hosts
- **Local**: Direct process communication
- **Container**: Podman/Docker exec
- **WebSocket**: Persistent bidirectional connections
- **HTTP**: Request/response for simpler integrations

## Status

Early design phase. Everything is subject to change.

## Name

Latis: from "lattice" — a structure of interconnected points. Agents connected across a distributed mesh.

Or if you prefer acronyms: **L**inked **A**gent **T**ransport & **I**nterconnection **S**ystem.

## License

TBD
