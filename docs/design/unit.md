# Server Mode

How a Latis node operates as a server (daemon mode).

## Overview

When you run `latis serve`, the node operates in server mode:

- Listens for incoming connections
- Handles Control protocol requests (ping, status, shutdown)
- Handles A2A protocol requests (prompts, tasks)
- Interfaces with an LLM provider to generate responses

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      latis serve                             │
│                                                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────┐ │
│  │   A2A       │    │   Control   │    │      LLM        │ │
│  │   Server    │    │   Server    │    │    Provider     │ │
│  └─────────────┘    └─────────────┘    └─────────────────┘ │
│         │                  │                    │          │
│         └──────────────────┼────────────────────┘          │
│                            │                               │
│                    ┌───────┴───────┐                       │
│                    │  MuxListener  │                       │
│                    │  (QUIC/mTLS)  │                       │
│                    └───────────────┘                       │
└─────────────────────────────────────────────────────────────┘
```

## Responsibilities

| Component | Purpose |
|-----------|---------|
| **MuxListener** | Accept connections, route streams by type |
| **Control Server** | Handle ping, status, shutdown requests |
| **A2A Server** | Handle agent communication (prompts, tasks) |
| **LLM Provider** | Generate responses via configured LLM |

## LLM Providers

The server interfaces with AI models via pluggable LLM providers:

| Provider | Description |
|----------|-------------|
| `echo` | Returns input back (for testing) |
| `ollama` | Connects to Ollama via OpenAI-compatible API |

```bash
# Testing
latis serve --pki-init --llm-provider=echo

# Production
latis serve --pki-init --llm-provider=ollama --llm-model=llama3.2
```

Provider implementation: `pkg/llm/`

## A2A Executor

The A2A server uses an executor (`pkg/a2aexec/`) to handle agent logic:

1. Receive message via A2A protocol
2. Extract prompt content
3. Call LLM provider for response
4. Stream or return response via A2A

```go
// Simplified flow
func (e *Executor) SendMessage(ctx context.Context, req *a2a.SendMessageRequest) (*a2a.SendMessageResponse, error) {
    prompt := extractPrompt(req.Message)
    response, err := e.llmProvider.Generate(ctx, prompt)
    return buildResponse(response), nil
}
```

## State Management

The server tracks its operational state (`pkg/control/state.go`):

| State | Meaning |
|-------|---------|
| `STARTING` | Initializing, not ready for requests |
| `READY` | Accepting requests |
| `BUSY` | Processing requests (may accept more) |
| `DRAINING` | Finishing in-progress work, rejecting new requests |
| `STOPPED` | Shutdown complete |

State is exposed via `GetStatus` RPC.

## Configuration

Server behavior is controlled via config file or CLI flags:

```yaml
version: v1

server:
  addr: "[::]:4433"    # Listen address

agent:
  name: my-agent       # Agent name for AgentCard
  description: "..."   # Agent description
  streaming: true      # Enable streaming responses
  skills: [...]        # Agent capabilities

llm:
  provider: ollama
  model: llama3.2

pki:
  dir: ~/.latis/pki
  init: true
```

See [docs/configuration.md](../configuration.md) for full reference.

## Lifecycle

```
1. Parse config and flags
2. Initialize PKI (if --pki-init)
3. Create LLM provider
4. Start MuxListener
5. Start Control and A2A gRPC servers
6. Set state to READY
7. Handle requests...
8. On SIGINT/SIGTERM or Shutdown RPC:
   a. Set state to DRAINING
   b. Stop accepting new connections
   c. Wait for in-progress requests
   d. Set state to STOPPED
   e. Exit
```

## Signal Handling

The server handles graceful shutdown on:

- `SIGINT` (Ctrl+C)
- `SIGTERM` (container/system shutdown)
- `Shutdown` RPC (remote request)

In-progress requests are allowed to complete (with timeout).
