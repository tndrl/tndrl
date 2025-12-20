# unit

The agent endpoint daemon for Latis.

## Responsibilities

- **Protocol handling**: Parse incoming messages, emit responses
- **Agent wrapping**: Interface with underlying AI agents (Claude, GPT, local models, custom code)
- **Session state**: Maintain conversation context and agent state
- **Streaming**: Push response chunks back to cmdr in real-time
- **Connection management**: Accept connections from cmdr OR dial out to cmdr

## Operation Modes

A unit can run as:

- **Daemon**: Long-running process accepting connections
- **On-demand**: Spawned by provisioner, runs for session duration
- **Embedded**: Library mode, integrated into other applications

## Connection Direction

Units can connect to cmdr in two ways:

```
┌────────────┐                    ┌────────────┐
│    cmdr    │ ──── dial-out ───→ │    unit    │   cmdr initiates (unit listens)
└────────────┘                    └────────────┘

┌────────────┐                    ┌────────────┐
│    cmdr    │ ←─── dial-in ────  │    unit    │   unit initiates (cmdr listens)
└────────────┘                    └────────────┘
```

**When cmdr dials out (unit listens):**
- Unit exposes an endpoint (TCP, WebSocket, Unix socket)
- cmdr connects to it
- Works well for: local processes, accessible servers, VMs with known addresses

**When unit dials out (cmdr listens):**
- cmdr exposes a listener endpoint
- Unit connects back to cmdr
- Works well for: NAT traversal, firewalled environments, ephemeral cloud instances

**Constraint:** At least one connection method must be configured. A unit can support both (listen AND dial), and can have multiple listeners.

## Protocol Messages Handled

```
← session.create     → ack + session_id
← session.resume     → ack + state
← session.destroy    → ack
← prompt.send        → response.chunk* + response.complete
← prompt.cancel      → ack
← state.get          → state.update
← state.subscribe    → state.update*
```

## Agent Adapters

Units don't implement AI directly — they wrap agents:

```
unit
 └── agent adapter
      └── actual agent (claude, gpt, llama, custom, etc.)
```

The adapter interface is minimal: receive prompt, stream response, report state.

## Design Notes

Units are designed to be lightweight. They should be easy to deploy anywhere — a remote server, a container, a Raspberry Pi. The heavy lifting happens in the wrapped agent; the unit just handles protocol and plumbing.
