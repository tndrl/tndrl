# CLI

The Latis command-line interface for interacting with nodes.

## Overview

Latis is a single binary with subcommands for both server and client operations:

| Command | Purpose |
|---------|---------|
| `latis serve` | Run as daemon (server mode) |
| `latis ping` | Health check a peer |
| `latis status` | Get peer status |
| `latis prompt` | Send message to peer via A2A |
| `latis discover` | Fetch peer's AgentCard |
| `latis shutdown` | Request peer shutdown |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                           latis                              │
│                                                             │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                     CLI (Kong)                          ││
│  │  serve │ ping │ status │ prompt │ discover │ shutdown   ││
│  └─────────────────────────────────────────────────────────┘│
│                            │                                │
│           ┌────────────────┼────────────────┐              │
│           ▼                ▼                ▼              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐    │
│  │   Server    │  │   Client    │  │  Config         │    │
│  │   Mode      │  │   Mode      │  │  (YAML/env/CLI) │    │
│  └─────────────┘  └─────────────┘  └─────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

## Server Mode

`latis serve` runs as a daemon. See [unit.md](./unit.md) for details.

## Client Mode

Other commands (`ping`, `status`, `prompt`, etc.) operate as clients:

1. Load configuration (config file, env vars, CLI flags)
2. Resolve peer address (by name or direct address)
3. Connect via QUIC/mTLS
4. Make RPC call(s)
5. Display result
6. Exit

### Connection Flow

```go
// Simplified client flow (cmd/latis/client.go)
func connectAndRun(cli *CLI, peer string, fn func(mux *MuxDialer) error) error {
    // 1. Initialize PKI
    tlsConfig, err := setupPKI(cli)

    // 2. Create dialer
    muxDialer := quictransport.NewMuxDialer(tlsConfig, nil)
    defer muxDialer.Close()

    // 3. Run client operation
    return fn(muxDialer)
}
```

## Configuration

The CLI uses a unified configuration system:

**Precedence**: CLI flags > environment variables > config file > defaults

```bash
# Config file
latis serve -c config.yaml

# Environment variable
LATIS_LLM_PROVIDER=echo latis serve

# CLI flag (highest priority)
latis serve -c config.yaml --llm-model=mistral
```

### Named Peers

Config files can define named peers:

```yaml
peers:
  - name: local
    addr: localhost:4433
  - name: backend
    addr: backend.example.com:4433
```

Then use by name:
```bash
latis ping local
latis prompt backend "Hello!"
```

## Implementation

| File | Purpose |
|------|---------|
| `cmd/latis/main.go` | Entry point, Kong CLI setup |
| `cmd/latis/cli.go` | CLI struct, config loading, types |
| `cmd/latis/serve.go` | Server mode implementation |
| `cmd/latis/client.go` | Shared client connection logic |
| `cmd/latis/ping.go` | Ping command |
| `cmd/latis/status.go` | Status command |
| `cmd/latis/prompt.go` | Prompt command |
| `cmd/latis/discover.go` | Discover command |
| `cmd/latis/shutdown.go` | Shutdown command |

## Design Decisions

### Single Binary

One `latis` binary for all operations:

- **Simpler deployment** — one binary to install
- **Shared code** — PKI, config, transport code used by all modes
- **Peer-to-peer** — any node can be both server and client

### Kong CLI Framework

Using [Kong](https://github.com/alecthomas/kong) for CLI parsing:

- Struct-based configuration
- Automatic flag/env binding
- Subcommand support
- Good Go integration

### Config-Driven

All configuration comes from the same schema:

- Config file (YAML)
- Environment variables (`LATIS_*`)
- CLI flags (`--*`)

See [docs/configuration.md](../configuration.md) for the full schema.

## Future Considerations

### Interactive Mode

A persistent CLI session for multiple operations:

```bash
latis shell
> ping backend
> prompt backend "Hello"
> status backend
> exit
```

### Multi-Agent Coordination

Orchestrating multiple agents from CLI:

```bash
latis coordinate --agents agent1,agent2 "Work together on this task"
```

### Provisioning

Spawning and managing agent nodes:

```bash
latis provision --type=container --image=latis:latest
latis destroy <node-id>
```

These would require implementing the `Provisioner` interface.
