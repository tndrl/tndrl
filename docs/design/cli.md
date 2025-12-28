# CLI

The Tndrl command-line interface for interacting with nodes.

## Overview

Tndrl is a single binary with subcommands for both server and client operations:

| Command | Purpose |
|---------|---------|
| `tndrl serve` | Run as daemon (server mode) |
| `tndrl ping` | Health check a peer |
| `tndrl status` | Get peer status |
| `tndrl prompt` | Send message to peer via A2A |
| `tndrl discover` | Fetch peer's AgentCard |
| `tndrl shutdown` | Request peer shutdown |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                           tndrl                              │
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

`tndrl serve` runs as a daemon. See [server.md](./server.md) for details.

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
// Simplified client flow (cmd/tndrl/client.go)
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
tndrl serve -c config.yaml

# Environment variable
TNDRL_LLM_PROVIDER=echo tndrl serve

# CLI flag (highest priority)
tndrl serve -c config.yaml --llm-model=mistral
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
tndrl ping local
tndrl prompt backend "Hello!"
```

## Implementation

| File | Purpose |
|------|---------|
| `cmd/tndrl/main.go` | Entry point, Kong CLI setup |
| `cmd/tndrl/cli.go` | CLI struct, config loading, types |
| `cmd/tndrl/serve.go` | Server mode implementation |
| `cmd/tndrl/client.go` | Shared client connection logic |
| `cmd/tndrl/ping.go` | Ping command |
| `cmd/tndrl/status.go` | Status command |
| `cmd/tndrl/prompt.go` | Prompt command |
| `cmd/tndrl/discover.go` | Discover command |
| `cmd/tndrl/shutdown.go` | Shutdown command |

## Design Decisions

### Single Binary

One `tndrl` binary for all operations:

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
- Environment variables (`TNDRL_*`)
- CLI flags (`--*`)

See [docs/configuration.md](../configuration.md) for the full schema.

## Future Considerations

### Interactive Mode

A persistent CLI session for multiple operations:

```bash
tndrl shell
> ping backend
> prompt backend "Hello"
> status backend
> exit
```

### Multi-Agent Coordination

Orchestrating multiple agents from CLI:

```bash
tndrl coordinate --agents agent1,agent2 "Work together on this task"
```

### Provisioning

Spawning and managing agent nodes:

```bash
tndrl provision --type=container --image=tndrl:latest
tndrl destroy <node-id>
```

These would require implementing the `Provisioner` interface.
