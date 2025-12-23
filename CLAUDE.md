# CLAUDE.md

Guidance for Claude Code when working in this repository.

## What is Latis?

A control plane for distributed AI agents built on the [A2A protocol](https://a2a-protocol.org/).

Latis provides orchestration, provisioning, and secure transport (QUIC/mTLS) while using A2A for agent communication semantics. Units are A2A-compatible agents; cmdr is both an orchestrator and an A2A agent itself.

## Architecture

```
cmdr → connector → unit → [agent]
```

| Component | Purpose | Details |
|-----------|---------|---------|
| **cmdr** | Human interface, orchestration, provisioning | [docs/design/cmdr.md](./docs/design/cmdr.md) |
| **connector** | Transport layer (SSH, WebSocket, etc.) | [docs/design/connector.md](./docs/design/connector.md) |
| **unit** | Agent endpoint daemon | [docs/design/unit.md](./docs/design/unit.md) |
| **protocol** | Wire format, message types | [docs/design/protocol.md](./docs/design/protocol.md) |

## Key Design Decisions

- **A2A protocol alignment** — agent communication follows the [A2A spec](https://a2a-protocol.org/), enabling interop with the broader agent ecosystem
- **Go** for the core (cmdr, unit) — single binary, easy cross-compilation, good concurrency
- **QUIC transport** — modern, multiplexed, connection migration
- **mTLS everywhere** — mutual TLS with built-in CA, SPIFFE-compatible identities
- **Cmdr as orchestrator + agent** — coordinates units internally, exposes A2A externally
- **Pluggable provisioners** — process, container, VM, cloud (design phase)

## Code Structure

```
latis/
├── cmd/
│   ├── latis/           # cmdr CLI
│   └── latis-unit/      # unit daemon
├── gen/go/latis/v1/     # generated protobuf/gRPC code
├── pkg/
│   ├── pki/             # CA and certificate management (mTLS)
│   ├── transport/quic/  # QUIC transport for gRPC
│   ├── protocol/        # (placeholder)
│   ├── connector/       # (placeholder)
│   └── ...
├── proto/latis/v1/      # protobuf definitions
├── buf.yaml             # buf configuration
└── buf.gen.yaml         # code generation config
```

## Quickstart

```bash
# Terminal 1: Initialize PKI and start unit
go run ./cmd/latis-unit/ --init-pki

# Terminal 2: Generate cmdr cert and connect
go run ./cmd/latis/ --init-pki

# Or send a prompt
go run ./cmd/latis/ -prompt "Hello, World!"
```

PKI files are stored in `~/.latis/pki/`. See [pkg/pki/README.md](./pkg/pki/README.md) for details.

## Documentation

- [Protobuf & buf](./docs/protobuf.md) — schema definitions, code generation, workflow

### Design Documents

- [A2A Alignment](./docs/design/a2a-alignment.md) — adopting A2A protocol, implementation path
- [Execution Model](./docs/design/execution-model.md) — tools, autonomy, yield points
- [Policy](./docs/design/policy.md) — authorization, OPA integration (deferred)

## Status

See **[docs/PROJECT.md](./docs/PROJECT.md)** for current progress and next steps.

**Current**: Multiplexed QUIC transport complete. Next: delete legacy proto, add real agent execution.

## Development Environment

Prefer running commands inside `toolbox` when available. The host system may not have development tools like `go` installed.

```bash
# Enter toolbox before running go commands
toolbox run go test ./...
toolbox run go build ./cmd/...
```

## Testing

Tests use [goleak](https://github.com/uber-go/goleak) for goroutine leak detection and race detection is enabled by default.

```bash
# Run all tests with race detection (default)
make test

# Verbose output
make test-verbose

# With coverage report
make test-cover

# Unit tests only (skip integration)
make test-unit

# Integration tests only
make test-integration
```

Each test package has a `TestMain` that runs goleak verification after all tests complete. If a test leaks goroutines, it will fail.

## Git Workflow

Branch protection requires PRs for all changes to main. Always use fetch/rebase:

```bash
# Update main
git checkout main
git fetch origin main
git rebase origin/main

# Create a feature branch
git checkout -b feature/my-feature

# ... make changes, commit ...

# Push and create PR
git push -u origin feature/my-feature
gh pr create

# After PR is merged, return to main
git checkout main
git fetch origin main
git rebase origin/main

# Update current branch with latest main (without switching)
git fetch origin main
git rebase origin/main

# Move uncommitted work to a new branch after PR merge
git fetch origin main
git rebase origin/main
git checkout -b new-branch-name

# If upstream has diverged (e.g., squash merge, force push)
git checkout -b backup/my-feature  # backup first
git checkout my-feature
git fetch origin
git reset --hard origin/my-feature
```

**Never use `git pull` or `git merge`** — always fetch then rebase (or reset --hard only when upstream has diverged).

## When Working Here

1. Read the component README for context before making changes
2. Capture design decisions in documentation as they're made
3. Keep docs minimal — prefer pointers over duplication
4. **Always update [docs/PROJECT.md](./docs/PROJECT.md)** when completing work:
   - Move completed items from "In Progress" or "Next Steps" to "Completed"
   - Add new tasks discovered during implementation to "Next Steps"
   - Update "Current Objective" if focus has shifted
   - Keep the tracking document accurate — it's the source of truth for project state
