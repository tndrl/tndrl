# CLI Reference

Tndrl is a single binary with subcommands for both daemon and client operations.

## Global Flags

These flags apply to all commands:

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Path to config file |
| `--log-level` | | Log level (debug, info, warn, error). Default: info |
| `--verbose` | `-v` | Verbose output (same as --log-level=debug) |
| `--help` | `-h` | Show help |

## Commands

### serve

Run as a daemon, listening for connections from other nodes.

```bash
tndrl serve [flags]
```

#### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--server-addr` | `[::]:4433` | Listen address |
| `--agent-name` | `tndrl-agent` | Agent name |
| `--agent-description` | | Agent description |
| `--agent-streaming` | `true` | Enable streaming responses |
| `--llm-provider` | **required** | LLM provider (echo, ollama) |
| `--llm-model` | | Model name (required for ollama) |
| `--llm-url` | | Provider API URL |
| `--pki-dir` | `~/.tndrl/pki` | PKI directory |
| `--pki-ca-cert` | `<pki-dir>/ca.crt` | CA certificate path |
| `--pki-ca-key` | `<pki-dir>/ca.key` | CA private key path |
| `--pki-cert` | `<pki-dir>/tndrl.crt` | Node certificate path |
| `--pki-key` | `<pki-dir>/tndrl.key` | Node private key path |
| `--pki-init` | `false` | Initialize PKI if missing |

#### Examples

```bash
# For testing (echo provider)
tndrl serve --pki-init --llm-provider=echo

# With Ollama
tndrl serve --pki-init --llm-provider=ollama --llm-model=llama3.2

# With config file
tndrl serve -c config.yaml

# Override config file options
tndrl serve -c config.yaml --llm-model=mistral
```

### ping

Ping a peer node to check connectivity.

```bash
tndrl ping <peer>
```

#### Arguments

| Argument | Description |
|----------|-------------|
| `peer` | Peer address (host:port) or name from config |

#### Examples

```bash
tndrl ping localhost:4433
tndrl ping backend  # uses name from config peers section
```

### status

Get status information from a peer node.

```bash
tndrl status <peer>
```

#### Arguments

| Argument | Description |
|----------|-------------|
| `peer` | Peer address or name |

#### Output

```
Node Status:
  Identity: spiffe://tndrl/node/abc123
  State: READY
  Uptime: 120s
```

#### Examples

```bash
tndrl status localhost:4433
tndrl status backend
```

### prompt

Send a prompt to a peer via the A2A protocol.

```bash
tndrl prompt [flags] <peer> <message>
```

#### Arguments

| Argument | Description |
|----------|-------------|
| `peer` | Peer address or name |
| `message` | Message to send |

#### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--stream` | `-s` | Use streaming response |

#### Examples

```bash
# Non-streaming
tndrl prompt localhost:4433 "Hello, what can you do?"

# Streaming
tndrl prompt -s localhost:4433 "Tell me a story"

# Using peer name
tndrl prompt backend "Review this code"
```

### discover

Fetch a peer's AgentCard to discover its capabilities.

```bash
tndrl discover <peer>
```

#### Arguments

| Argument | Description |
|----------|-------------|
| `peer` | Peer address or name |

#### Output

```
Agent: backend-agent
Description: Backend processing agent
Transport: grpc
Streaming: true

Skills:
  - code-review: Review code for issues and improvements
    Tags: [code, review]
  - summarize: Summarize documents
    Tags: [text, summarization]
```

#### Examples

```bash
tndrl discover localhost:4433
tndrl discover backend
```

### shutdown

Request a peer to shut down.

```bash
tndrl shutdown [flags] <peer>
```

#### Arguments

| Argument | Description |
|----------|-------------|
| `peer` | Peer address or name |

#### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--force` | `false` | Force immediate shutdown (not graceful) |
| `--timeout` | `30` | Graceful shutdown timeout in seconds |
| `--reason` | `requested by peer` | Reason for shutdown |

#### Examples

```bash
# Graceful shutdown
tndrl shutdown localhost:4433

# Force immediate shutdown
tndrl shutdown --force localhost:4433

# With custom timeout and reason
tndrl shutdown --timeout=60 --reason="maintenance" backend
```

## PKI Configuration

All client commands (ping, status, prompt, discover, shutdown) require valid certificates to connect to peers.

### First-time Setup

When running a client command for the first time, initialize PKI:

```bash
# Run any command with --pki-init
tndrl ping localhost:4433 --pki-init
```

Or use a config file with `pki.init: true`.

### Certificate Location

By default, certificates are stored in `~/.tndrl/pki/`:

```
~/.tndrl/pki/
├── ca.crt          # CA certificate
├── node.crt        # Node certificate
└── node.key        # Node private key
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (connection failed, request rejected, etc.) |

## Examples

### Complete Workflow

```bash
# Terminal 1: Start a node
tndrl serve --pki-init --llm-provider=ollama --llm-model=llama3.2

# Terminal 2: Interact with the node
tndrl ping localhost:4433
tndrl status localhost:4433
tndrl discover localhost:4433
tndrl prompt localhost:4433 "Hello!"
tndrl prompt -s localhost:4433 "Tell me about yourself"
tndrl shutdown localhost:4433
```

### Using Config Files

```bash
# Create config file
cat > config.yaml << 'EOF'
version: v1
server:
  addr: "[::]:4433"
agent:
  name: my-agent
  description: "My custom agent"
llm:
  provider: ollama
  model: llama3.2
pki:
  init: true
peers:
  - name: main
    addr: localhost:4433
EOF

# Use config file
tndrl serve -c config.yaml

# Client commands can also use config (for PKI and peers)
tndrl ping main -c config.yaml
```
