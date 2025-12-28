# Configuration Reference

Tndrl uses a unified configuration system where CLI flags, environment variables, and config files all derive from the same schema.

## Precedence

**CLI flags > environment variables > config file > defaults**

When the same option is specified in multiple places, higher-precedence sources override lower ones.

## Config File

Config files use YAML format. Specify with `-c` or `--config`:

```bash
tndrl serve -c config.yaml
tndrl serve --config=/etc/tndrl/config.yaml
```

### Example Config

```yaml
version: v1

logLevel: info  # debug, info, warn, error

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
      examples:
        - "Review this function for bugs"
        - "Check this code for security issues"

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
  - name: frontend
    addr: frontend.local:4433
```

## Configuration Sections

### version

```yaml
version: v1
```

Config file version. Currently only `v1` is supported.

### logLevel

```yaml
logLevel: info
```

Log verbosity level. Valid values: `debug`, `info`, `warn`, `error`. Default: `info`.

- **debug**: Detailed tracing (connection lifecycle, stream routing, internal state)
- **info**: Normal operations (startup, ready, shutdown, requests served)
- **warn**: Non-critical issues (shutdown timeouts, unknown stream types)
- **error**: Failures (connection errors, request failures)

### server

Server configuration for daemon mode (`tndrl serve`).

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `addr` | string | `[::]:4433` | Listen address (host:port) |

```yaml
server:
  addr: "[::]:4433"
```

### agent

Agent identity and capabilities, exposed via A2A AgentCard.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `name` | string | `tndrl-agent` | Agent name |
| `description` | string | `""` | Agent description |
| `streaming` | bool | `true` | Whether agent supports streaming |
| `skills` | array | `[]` | List of agent skills |

#### Skills

Each skill has:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Unique skill identifier |
| `name` | string | yes | Human-readable name |
| `description` | string | no | What the skill does |
| `tags` | array | no | Categorization tags |
| `examples` | array | no | Example prompts |

```yaml
agent:
  name: code-assistant
  description: "AI assistant for software development"
  streaming: true
  skills:
    - id: review
      name: Code Review
      description: "Review code for issues"
      tags: [code, review]
      examples:
        - "Review this function"
```

### llm

LLM provider configuration. **Required** - you must specify a provider.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `provider` | string | **yes** | Provider type (echo, ollama, mcphost) |
| `model` | string | for ollama/mcphost | Model name |
| `url` | string | no | Provider API URL (defaults to localhost:11434/v1 for ollama) |
| `systemPrompt` | string | no | System prompt for the LLM |
| `maxSteps` | int | no | Maximum tool call iterations (0=unlimited) |
| `mcpConfigFile` | string | no | Path to external mcphost config file |
| `mcpServers` | map | no | MCP server configurations (ignored if mcpConfigFile is set) |

#### Providers

| Provider | Description |
|----------|-------------|
| `echo` | Echoes input back (for testing) |
| `ollama` | Connects to Ollama via OpenAI-compatible API |
| `mcphost` | Full MCP tool support via mcphost SDK |

```yaml
# For testing
llm:
  provider: echo

# For production with Ollama (no tools)
llm:
  provider: ollama
  model: llama3.2
  url: http://localhost:11434/v1

# For production with MCP tools (embedded config)
llm:
  provider: mcphost
  model: ollama:llama3.2
  systemPrompt: "You are a helpful assistant with tool access."
  maxSteps: 10
  mcpServers:
    filesystem:
      type: builtin
      name: fs
      options:
        allowed_directories: ["/tmp"]

# For production with MCP tools (external config file)
# This allows sharing the same config with mcphost CLI
llm:
  provider: mcphost
  model: ollama:llama3.2
  mcpConfigFile: ~/.mcphost.yaml
```

#### MCP Server Configuration

When using the `mcphost` provider, you can configure MCP servers to provide tools:

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Server type: `local`, `remote`, or `builtin` |
| `command` | []string | Command to run (for `local` type) |
| `environment` | map | Environment variables (for `local` type) |
| `url` | string | Server URL (for `remote` type) |
| `headers` | []string | HTTP headers (for `remote` type) |
| `name` | string | Builtin server name (for `builtin` type) |
| `options` | map | Server options (for `builtin` type) |

**Builtin servers available:**
- `fs` - Filesystem access with configurable allowed directories
- `bash` - Shell command execution
- `todo` - Task management
- `http` - HTTP fetch operations

```yaml
llm:
  provider: mcphost
  model: ollama:llama3.2
  mcpServers:
    # Builtin server (in-process, fast)
    filesystem:
      type: builtin
      name: fs
      options:
        allowed_directories: ["/tmp", "/home/user"]

    # Local server (stdio)
    sqlite:
      type: local
      command: ["uvx", "mcp-server-sqlite", "--db-path", "/tmp/db.sqlite"]
      environment:
        DEBUG: "true"

    # Remote server (HTTP)
    api:
      type: remote
      url: "https://api.example.com/mcp"
      headers:
        - "Authorization: Bearer ${API_TOKEN}"
```

### pki

PKI (certificate) configuration.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `dir` | string | `~/.tndrl/pki` | PKI directory path |
| `init` | bool | `false` | Auto-initialize PKI if missing |

```yaml
pki:
  dir: ~/.tndrl/pki
  init: true
```

### peers

Named peers for convenience. Can use peer names instead of addresses in commands.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Peer name (used in commands) |
| `addr` | string | Peer address (host:port) |

```yaml
peers:
  - name: backend
    addr: backend.local:4433
  - name: frontend
    addr: frontend.local:4433
```

Usage:
```bash
# Use name from config
tndrl ping backend

# Or use address directly
tndrl ping backend.local:4433
```

## Environment Variables

All config options can be set via environment variables with `TNDRL_` prefix:

| Config Path | Environment Variable |
|-------------|---------------------|
| `logLevel` | `TNDRL_LOG_LEVEL` |
| `server.addr` | `TNDRL_ADDR` |
| `agent.name` | `TNDRL_AGENT_NAME` |
| `agent.description` | `TNDRL_AGENT_DESCRIPTION` |
| `agent.streaming` | `TNDRL_AGENT_STREAMING` |
| `llm.provider` | `TNDRL_LLM_PROVIDER` |
| `llm.model` | `TNDRL_LLM_MODEL` |
| `llm.url` | `TNDRL_LLM_URL` |
| `pki.dir` | `TNDRL_PKI_DIR` |
| `pki.caCert` | `TNDRL_CA_CERT` |
| `pki.caKey` | `TNDRL_CA_KEY` |
| `pki.cert` | `TNDRL_CERT` |
| `pki.key` | `TNDRL_KEY` |
| `pki.init` | `TNDRL_INIT_PKI` |

```bash
TNDRL_LLM_PROVIDER=ollama TNDRL_LLM_MODEL=llama3.2 tndrl serve
```

## CLI Flags

CLI flags use `--section-field` format:

```bash
tndrl serve --server-addr=:4433 --llm-provider=ollama --llm-model=llama3.2
```

See [CLI Reference](./cli.md) for all available flags.

## Combining Sources

All three sources can be combined:

```bash
# Base config from file
# Override LLM model from environment
# Override address from CLI
TNDRL_LLM_MODEL=mistral tndrl serve -c config.yaml --server-addr=:8443
```
