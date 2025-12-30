# Tndrl

A control plane for distributed AI agents built on the [A2A protocol](https://a2a-protocol.org/).

Tndrl provides a unified interface for orchestrating AI agents running across multiple machines, containers, and environments. It's transport-agnostic, agent-agnostic, and designed to scale from a single local agent to coordinated fleets.

## Quickstart

```bash
# Terminal 1: Start a node as a daemon
tndrl serve -c examples/echo.yaml

# Terminal 2: Interact with the node (using named peer from config)
tndrl ping -c examples/echo.yaml local
tndrl status -c examples/echo.yaml local
tndrl prompt -c examples/echo.yaml local "Hello, what can you do?"
tndrl discover -c examples/echo.yaml local
```

Or with CLI flags only (no config file):

```bash
tndrl serve --pki-init --llm-provider=echo
tndrl ping localhost:4433
```

## Architecture

Tndrl uses a **peer-to-peer** model where any node can both serve requests and connect to other nodes.

```
┌─────────────────────────────────────────────────────────────┐
│                         tndrl node                          │
│                                                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────┐ │
│  │   A2A       │    │   Control   │    │      LLM        │ │
│  │   Server    │    │   Server    │    │    Provider     │ │
│  └─────────────┘    └─────────────┘    └─────────────────┘ │
│         │                  │                    │          │
│         └──────────────────┼────────────────────┘          │
│                            │                               │
│                    ┌───────┴───────┐                       │
│                    │  QUIC/mTLS    │                       │
│                    └───────────────┘                       │
└─────────────────────────────────────────────────────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │   Other Nodes   │
                    │   (peers)       │
                    └─────────────────┘
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `tndrl serve` | Run as daemon, listen for connections |
| `tndrl ping <peer>` | Ping a peer node |
| `tndrl status <peer>` | Get peer status |
| `tndrl prompt <peer> <message>` | Send prompt via A2A |
| `tndrl discover <peer>` | Fetch peer's AgentCard (capabilities) |
| `tndrl shutdown <peer>` | Request peer shutdown |

All commands support `--help` for detailed options.

## Configuration

Tndrl uses a unified configuration system where CLI flags, environment variables, and config files all derive from the same schema.

**Precedence**: CLI flags > environment variables > config file > defaults

### Example Config File

```yaml
version: v1

server:
  addr: "[::]:4433"

agent:
  name: my-agent
  description: "AI assistant for code review"
  streaming: true
  skills:
    - id: review
      name: Code Review
      description: "Review code for issues and improvements"
      tags: [code, review]

llm:
  provider: ollama
  model: llama3.2
  url: http://localhost:11434/v1

pki:
  dir: ~/.tndrl/pki
  init: true

peers:
  - name: backend
    addr: backend.local:4433
```

### Using Config Files

```bash
# Use a config file
tndrl serve -c config.yaml

# Override config with CLI flags
tndrl serve -c config.yaml --llm-model=mistral

# Use environment variables
TNDRL_LLM_PROVIDER=ollama tndrl serve
```

## LLM Providers

Tndrl requires an LLM provider to be configured:

| Provider | Description |
|----------|-------------|
| `echo` | Echoes input (for testing) |
| `ollama` | Connects to Ollama via OpenAI-compatible API |
| `mcphost` | Full agentic loop with MCP tool support via [mcphost](https://github.com/mark3labs/mcphost) |

```bash
# For testing
tndrl serve --pki-init --llm-provider=echo

# With Ollama
tndrl serve --pki-init --llm-provider=ollama --llm-model=llama3.2

# With custom URL
tndrl serve --pki-init --llm-provider=ollama --llm-model=llama3.2 --llm-url=http://ollama:11434/v1

# With MCP tools (see examples/mcphost.yaml)
tndrl serve -c examples/mcphost.yaml
```

## Security

All connections use **mTLS** (mutual TLS) — both sides present and verify certificates.

Key features:
- **Built-in CA** — Tndrl generates and manages its own certificate authority
- **BYO CA** — Bring your own root certificate for enterprise deployments
- **SPIFFE-compatible** — Certificate identities use SPIFFE URI format
- **TLS 1.3** — Modern encryption via QUIC

```bash
# Initialize PKI (creates CA + node certificate)
tndrl serve --pki-init

# Certificates are stored in ~/.tndrl/pki/
```

## Peer Discovery

Nodes can discover each other's capabilities via the A2A AgentCard:

```bash
$ tndrl discover backend.local:4433

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

## Named Peers

Define peers in your config file to use names instead of addresses:

```yaml
# examples/echo.yaml
peers:
  - name: local
    addr: localhost:4433
```

Then use peer names in commands:

```bash
tndrl prompt -c examples/echo.yaml local "Hello!"
tndrl status -c examples/echo.yaml local
```

## Design Principles

- **A2A protocol alignment** — agent communication follows the [A2A spec](https://a2a-protocol.org/)
- **Transport agnostic** — QUIC today, more transports tomorrow
- **Peer-to-peer** — any node can both serve and connect
- **Single binary** — one `tndrl` binary for all roles
- **Config-driven** — unified CLI/env/file configuration

## Documentation

- [Configuration Reference](./docs/configuration.md)
- [CLI Reference](./docs/cli.md)
- [PKI & Security](./pkg/pki/README.md)
- [Project Status](./docs/PROJECT.md)

### Design Documents

- [A2A Alignment](./docs/design/a2a-alignment.md)
- [Protocol](./docs/design/protocol.md)

## License

[Apache-2.0](LICENSE)
