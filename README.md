# Latis

A control plane for distributed AI agents built on the [A2A protocol](https://a2a-protocol.org/).

Latis provides a unified interface for orchestrating AI agents running across multiple machines, containers, and environments. It's transport-agnostic, agent-agnostic, and designed to scale from a single local agent to coordinated fleets.

## Quickstart

```bash
# Terminal 1: Start a node as a daemon
latis serve -c examples/echo.yaml

# Terminal 2: Interact with the node (using named peer from config)
latis ping -c examples/echo.yaml local
latis status -c examples/echo.yaml local
latis prompt -c examples/echo.yaml local "Hello, what can you do?"
latis discover -c examples/echo.yaml local
```

Or with CLI flags only (no config file):

```bash
latis serve --pki-init --llm-provider=echo
latis ping localhost:4433
```

## Architecture

Latis uses a **peer-to-peer** model where any node can both serve requests and connect to other nodes.

```
┌─────────────────────────────────────────────────────────────┐
│                         latis node                          │
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
| `latis serve` | Run as daemon, listen for connections |
| `latis ping <peer>` | Ping a peer node |
| `latis status <peer>` | Get peer status |
| `latis prompt <peer> <message>` | Send prompt via A2A |
| `latis discover <peer>` | Fetch peer's AgentCard (capabilities) |
| `latis shutdown <peer>` | Request peer shutdown |

All commands support `--help` for detailed options.

## Configuration

Latis uses a unified configuration system where CLI flags, environment variables, and config files all derive from the same schema.

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
  dir: ~/.latis/pki
  init: true

peers:
  - name: backend
    addr: backend.local:4433
```

### Using Config Files

```bash
# Use a config file
latis serve -c config.yaml

# Override config with CLI flags
latis serve -c config.yaml --llm-model=mistral

# Use environment variables
LATIS_LLM_PROVIDER=ollama latis serve
```

## LLM Providers

Latis requires an LLM provider to be configured:

| Provider | Description |
|----------|-------------|
| `echo` | Echoes input (for testing) |
| `ollama` | Connects to Ollama via OpenAI-compatible API |
| `mcphost` | Full agentic loop with MCP tool support via [mcphost](https://github.com/mark3labs/mcphost) |

```bash
# For testing
latis serve --pki-init --llm-provider=echo

# With Ollama
latis serve --pki-init --llm-provider=ollama --llm-model=llama3.2

# With custom URL
latis serve --pki-init --llm-provider=ollama --llm-model=llama3.2 --llm-url=http://ollama:11434/v1

# With MCP tools (see examples/mcphost.yaml)
latis serve -c examples/mcphost.yaml
```

## Security

All connections use **mTLS** (mutual TLS) — both sides present and verify certificates.

Key features:
- **Built-in CA** — Latis generates and manages its own certificate authority
- **BYO CA** — Bring your own root certificate for enterprise deployments
- **SPIFFE-compatible** — Certificate identities use SPIFFE URI format
- **TLS 1.3** — Modern encryption via QUIC

```bash
# Initialize PKI (creates CA + node certificate)
latis serve --pki-init

# Certificates are stored in ~/.latis/pki/
```

## Peer Discovery

Nodes can discover each other's capabilities via the A2A AgentCard:

```bash
$ latis discover backend.local:4433

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
latis prompt -c examples/echo.yaml local "Hello!"
latis status -c examples/echo.yaml local
```

## Design Principles

- **A2A protocol alignment** — agent communication follows the [A2A spec](https://a2a-protocol.org/)
- **Transport agnostic** — QUIC today, more transports tomorrow
- **Peer-to-peer** — any node can both serve and connect
- **Single binary** — one `latis` binary for all roles
- **Config-driven** — unified CLI/env/file configuration

## Documentation

- [Configuration Reference](./docs/configuration.md)
- [CLI Reference](./docs/cli.md)
- [PKI & Security](./pkg/pki/README.md)
- [Project Status](./docs/PROJECT.md)

### Design Documents

- [A2A Alignment](./docs/design/a2a-alignment.md)
- [Protocol](./docs/design/protocol.md)

## Name

Latis: from "lattice" — a structure of interconnected points. Agents connected across a distributed mesh.

Or if you prefer acronyms: **L**inked **A**gent **T**ransport & **I**nterconnection **S**ystem.

## License

[Apache-2.0](LICENSE)
