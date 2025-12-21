# A2A Protocol Alignment

## Decision

Latis will align with the [Agent2Agent (A2A) protocol](https://a2a-protocol.org/) rather than maintain a separate wire protocol. A2A provides the agent communication semantics; Latis provides the control plane, transport, and orchestration layer.

## Rationale

- **Industry adoption**: A2A is backed by Google, Microsoft, Linux Foundation, 50+ companies
- **Interoperability**: Units become compatible with the broader agent ecosystem
- **Focus**: Latis can focus on its unique value (orchestration, provisioning, mTLS) rather than protocol semantics
- **Complementary to MCP**: A2A handles agent-to-agent; MCP handles agent-to-tool

## Architecture

```
                         ┌─────────────────────────────────┐
                         │     External A2A Agents         │
                         └───────────────┬─────────────────┘
                                         │ A2A Protocol
                                         ▼
┌────────────────────────────────────────────────────────────────┐
│                           cmdr                                  │
│                                                                 │
│   ┌─────────────┐    ┌─────────────┐    ┌─────────────────┐    │
│   │  A2A Agent  │◄──►│ Orchestrator│◄──►│  Human Interface│    │
│   │  Interface  │    │             │    │   (CLI/TUI)     │    │
│   └─────────────┘    └──────┬──────┘    └─────────────────┘    │
│                             │                                   │
└─────────────────────────────┼───────────────────────────────────┘
                              │ A2A over QUIC/mTLS
                              ▼
              ┌───────────────┴───────────────┐
              │               │               │
         ┌────┴────┐    ┌─────┴────┐    ┌────┴────┐
         │  unit   │    │   unit   │    │  unit   │
         │ (A2A)   │    │  (A2A)   │    │ (A2A)   │
         └─────────┘    └──────────┘    └─────────┘
```

## What A2A Provides

- **Message**: User/agent messages with content parts (text, data, file)
- **Task**: Stateful operation with lifecycle (submitted → working → completed)
- **AgentCard**: Self-describing manifest for discovery
- **Artifacts**: Structured outputs from agent work
- **Streaming**: Real-time status and artifact updates via events

## What Latis Adds

| Layer | Latis Contribution |
|-------|-------------------|
| **Transport** | QUIC for performance, connection migration |
| **Security** | mTLS with built-in CA, SPIFFE-compatible identities |
| **Orchestration** | Multi-agent coordination via cmdr |
| **Provisioning** | Unit lifecycle (spawn, stop, destroy) |
| **Discovery** | Private agent registry (vs public AgentCard) |

## Concept Mapping

Current Latis → A2A:

| Latis (current) | A2A Equivalent |
|-----------------|----------------|
| `ConnectRequest` | `MessageSendParams` |
| `PromptSend` | `Message` (role=user) |
| `ResponseChunk` | `TaskArtifactUpdateEvent` |
| `ResponseComplete` | `TaskStatusUpdateEvent` (state=completed) |
| `Ping/Pong` | Extension or transport-level keepalive |
| `StateGet` | `tasks/get` |
| Unit identity | AgentCard + mTLS |

## Implementation Path

### Phase 1: Adopt a2a-go types
- Import `github.com/a2aproject/a2a-go/a2a` for core types
- Units implement `AgentExecutor` interface
- Keep QUIC/mTLS transport layer

### Phase 2: A2A-compatible wire format
- Support A2A gRPC transport (already in a2a-go)
- Map to our QUIC transport
- Units expose AgentCard

### Phase 3: Cmdr as A2A agent
- Cmdr exposes AgentCard for external discovery
- External agents can request cmdr to coordinate tasks
- Cmdr orchestrates units, returns aggregated results

### Phase 4: Agent-to-agent via cmdr
- Units can request cmdr to talk to other units
- Cmdr mediates inter-unit A2A communication
- Policy/authorization at cmdr level

## Open Questions

1. **QUIC transport for A2A**: A2A supports gRPC but not explicitly QUIC. Do we contribute QUIC transport to a2a-go, or wrap it?

2. **Ping/Pong**: A2A doesn't have keepalive. Handle at transport layer (QUIC) or add as extension?

3. **Private vs public AgentCard**: Units should be discoverable by cmdr but not necessarily public. How to handle?

4. **Backward compatibility**: Migrate existing proto or maintain both during transition?

## References

- [A2A Protocol Spec](https://a2a-protocol.org/latest/)
- [a2a-go SDK](https://github.com/a2aproject/a2a-go)
- [Google A2A Announcement](https://developers.googleblog.com/en/a2a-a-new-era-of-agent-interoperability/)
- [Linux Foundation A2A Project](https://www.linuxfoundation.org/press/linux-foundation-launches-the-agent2agent-protocol-project-to-enable-secure-intelligent-communication-between-ai-agents)
