# protocol

The Latis wire protocol. Defines how cmdr, connectors, and units communicate.

## Design Decisions

### Protobuf schemas, not full gRPC

We use **protobuf** for message definitions (type safety, code generation, language agnostic) but do NOT require gRPC's HTTP/2 framing. This allows the protocol to work over any byte stream.

Rationale:
- gRPC requires HTTP/2, which is awkward over SSH stdin/stdout or container exec
- Protobuf gives us schema-driven development without the transport constraint
- Connectors that support it (WebSocket, TCP, QUIC) can optionally upgrade to full gRPC

### Wire format

Messages are **length-prefixed protobuf**:

```
┌──────────────┬─────────────────────────┐
│ length (4B)  │ protobuf message        │
│ big-endian   │ (variable length)       │
└──────────────┴─────────────────────────┘
```

This works over any byte stream: stdin/stdout, SSH channels, Unix sockets, TCP, etc.

### Multiplexed and async

The protocol is fully multiplexed:
- Every message has an `id` for request/response correlation
- Either side can send at any time
- Control messages (cancel, state queries) can be injected mid-stream
- Responses reference the originating request ID

Example flow:
```
→ {id:"1", type:PROMPT_SEND, content:"analyze this"}
→ {id:"2", type:STATE_GET}                           ← sent while prompt in progress
← {id:"1", type:RESPONSE_CHUNK, content:"Looking..."}
← {id:"2", type:STATE_UPDATE, state:{busy:true}}     ← response to state query
← {id:"1", type:RESPONSE_CHUNK, content:"at the..."}
→ {id:"3", type:PROMPT_CANCEL, target:"1"}           ← cancel mid-stream
← {id:"1", type:RESPONSE_CANCELLED}
```

## Message Types

### Controller → Unit

| Type | Description |
|------|-------------|
| `SESSION_CREATE` | Create a new agent session |
| `SESSION_RESUME` | Reconnect to existing session |
| `SESSION_DESTROY` | Terminate session |
| `PROMPT_SEND` | Send input (expects streaming response) |
| `PROMPT_CANCEL` | Cancel in-progress operation |
| `STATE_GET` | Query agent state |
| `STATE_SUBSCRIBE` | Subscribe to state changes |

### Unit → Controller

| Type | Description |
|------|-------------|
| `ACK` | Acknowledgment (with optional data) |
| `RESPONSE_CHUNK` | Streaming output fragment |
| `RESPONSE_COMPLETE` | Operation finished |
| `RESPONSE_CANCELLED` | Operation was cancelled |
| `STATE_UPDATE` | State change notification |
| `ERROR` | Error occurred |

## Transport Flexibility

The protocol layer is transport-agnostic. How messages get from A to B is the connector's job:

| Connector | Wire format |
|-----------|-------------|
| SSH/stdio | Length-prefixed protobuf over stdin/stdout |
| Local process | Same, over pipes |
| Container exec | Same, over exec stdin/stdout |
| WebSocket | Length-prefixed protobuf over WS frames (binary) |
| TCP/QUIC | Length-prefixed protobuf, or full gRPC |

## Future: gRPC upgrade path

For transports that natively support HTTP/2 or HTTP/3 (QUIC), we can offer full gRPC as an option. The protobuf message definitions remain the same — only the framing changes.

```
Option A: length-prefixed protobuf (works everywhere)
Option B: gRPC over HTTP/2 or HTTP/3 (native transports only)
```

Both use the same `.proto` schemas.
