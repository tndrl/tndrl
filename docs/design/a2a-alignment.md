# A2A Protocol Alignment

## Decision

Latis aligns with the [Agent2Agent (A2A) protocol](https://a2a-protocol.org/) for agent communication. A2A provides the agent communication semantics; Latis provides the control plane, transport, and infrastructure layer.

## Rationale

- **Industry adoption**: A2A is backed by Google, Microsoft, Linux Foundation, 50+ companies
- **Interoperability**: Latis nodes become compatible with the broader agent ecosystem
- **Focus**: Latis focuses on its unique value (orchestration, mTLS, QUIC transport) rather than protocol semantics
- **Complementary to MCP**: A2A handles agent-to-agent; MCP handles agent-to-tool

## Architecture

```
                         ┌─────────────────────────────────┐
                         │     External A2A Agents         │
                         └───────────────┬─────────────────┘
                                         │ A2A Protocol
                                         ▼
┌────────────────────────────────────────────────────────────────┐
│                           latis                                 │
│                                                                 │
│   ┌─────────────┐    ┌─────────────┐    ┌─────────────────┐    │
│   │  A2A Agent  │◄──►│   Control   │◄──►│  Human Interface│    │
│   │  Interface  │    │   Plane     │    │   (CLI)         │    │
│   └─────────────┘    └──────┬──────┘    └─────────────────┘    │
│                             │                                   │
└─────────────────────────────┼───────────────────────────────────┘
                              │ A2A over QUIC/mTLS
                              ▼
              ┌───────────────┴───────────────┐
              │               │               │
         ┌────┴────┐    ┌─────┴────┐    ┌────┴────┐
         │  latis  │    │  latis   │    │ latis   │
         │  node   │    │  node    │    │ node    │
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
| **Control Plane** | Node lifecycle (ping, status, shutdown) |
| **Configuration** | Unified CLI/env/file configuration |
| **LLM Integration** | Pluggable providers (echo, ollama) |

## Implementation Status

### Completed

- **a2a-go integration**: Using `github.com/a2aproject/a2a-go` for A2A types and gRPC services
- **AgentCard exposure**: Nodes expose capabilities via `GetAgentCard` RPC
- **Message handling**: `SendMessage` and `SendStreamingMessage` implemented
- **QUIC transport**: A2A runs over multiplexed QUIC streams
- **Separation from Control**: A2A and Control protocols run on separate stream types

### Open Questions

1. **QUIC transport for broader A2A**: A2A supports gRPC but not explicitly QUIC. Could contribute QUIC transport upstream.

2. **Private vs public AgentCard**: Nodes should be discoverable by peers but not necessarily public. May need access control.

3. **Task persistence**: Current implementation is stateless. May need task storage for long-running operations.

## References

- [A2A Protocol Spec](https://a2a-protocol.org/latest/)
- [a2a-go SDK](https://github.com/a2aproject/a2a-go)
- [Google A2A Announcement](https://developers.googleblog.com/en/a2a-a-new-era-of-agent-interoperability/)
- [Linux Foundation A2A Project](https://www.linuxfoundation.org/press/linux-foundation-launches-the-agent2agent-protocol-project-to-enable-secure-intelligent-communication-between-ai-agents)
