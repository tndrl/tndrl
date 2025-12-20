# connector

Transport abstraction layer for Latis. Connectors are spawned by cmdr and act as bidirectional proxies to units.

## Process Model

Connectors run as child processes of cmdr:

```
┌────────────┐
│    cmdr    │
└─────┬──────┘
      │ spawns
      ↓
┌────────────┐  stdin   ┌────────────┐
│ connector  │ ←──────→ │    cmdr    │  (protocol messages)
└─────┬──────┘  stdout  └────────────┘
      │
      │ transport (ssh, ws, etc.)
      ↓
┌────────────┐
│    unit    │
└────────────┘
```

1. cmdr spawns connector executable with target address as argument
2. Connector establishes transport to the unit
3. cmdr writes protocol messages to connector's stdin
4. Connector forwards messages to unit over the transport
5. Connector reads responses from unit, writes to stdout
6. cmdr reads from connector's stdout

## Responsibilities

- **Transport bytes**: Move protocol messages between cmdr and units
- **Connection lifecycle**: Establish, maintain, and teardown connections to units
- **Bidirectional proxy**: Read from stdin → send to unit; receive from unit → write to stdout
- **Pluggable**: Any transport mechanism can be a connector

## Executable Interface

Connectors are executables (any language) that follow this contract:

```
# Invocation
latis-connector-<type> <address> [options]

# Example
latis-connector-ssh user@host:22
latis-connector-ws wss://example.com/latis
latis-connector-local /path/to/unit

# I/O
stdin  ← protocol messages from cmdr (newline-delimited JSON or similar)
stdout → protocol messages to cmdr
stderr → logging/diagnostics (not protocol)
```

The connector doesn't understand the protocol messages — it just moves them. Serialization and deserialization happen at the protocol layer in cmdr and unit.

## Discovery

cmdr finds connectors via:

1. **Built-in**: Common connectors compiled into cmdr
2. **PATH**: Executables named `latis-connector-*`
3. **Plugin directory**: `~/.latis/connectors/`

## Planned Connectors

- **ssh**: Shell into remote hosts, run unit, communicate via stdin/stdout
- **local**: Spawn local unit process, communicate via stdin/stdout
- **container**: Exec into containers (podman, docker)
- **websocket**: Persistent bidirectional connections
- **http**: Request/response for stateless interactions

## Design Notes

- Connectors are intentionally dumb — they know how to establish a channel and push bytes through it
- Authentication and encryption are connector concerns (SSH handles auth, WebSocket can use TLS)
- Protocol semantics are NOT connector concerns — cmdr and unit handle that
- Connectors can be written in any language since they're separate executables
