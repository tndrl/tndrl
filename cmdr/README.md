# cmdr

The control plane and CLI for Latis. This is the human interface layer.

## Responsibilities

- **Human interface**: CLI (and eventually TUI/GUI/web) for interacting with agents
- **Connector launcher**: Spawn, manage, and tear down connector processes
- **Session management**: Create, resume, destroy agent sessions
- **Routing**: Direct messages to the right units via connectors
- **Orchestration**: Coordinate multi-agent tasks
- **State tracking**: Know what's running where

## Architecture

```
Human
   ↓
┌──────────────────────────────────────┐
│               cmdr                   │
│  ┌─────────────────────────────────┐ │
│  │      connector launcher         │ │  ← spawns connector processes
│  └──────────┬──────────────────────┘ │
└─────────────┼────────────────────────┘
              │ stdin/stdout (or socket)
         ┌────┴────┐
         │connector│  ← child process
         └────┬────┘
              │
            unit
```

## Connector Management

cmdr is responsible for launching connectors. Connectors can be:

- **Separate executables**: cmdr spawns `latis-connector-ssh`, communicates via stdin/stdout
- **Built-in**: Compiled into cmdr for common transports
- **Discovered**: Found in PATH or plugin directory (`~/.latis/connectors/`)

The cmdr ↔ connector interface:
1. cmdr spawns connector with target address as argument
2. Connector establishes transport to unit
3. cmdr writes protocol messages to connector's stdin
4. Connector forwards to unit, pipes responses back to stdout
5. cmdr reads from connector's stdout

The connector is a bidirectional proxy that cmdr controls.

## Usage

```
latis connect <unit-address>
latis session new [--connector ssh|ws|local]
latis prompt "your message here"
latis agents list
latis coordinate --agents unit-1,unit-2 "work together on this"
```

## Design Notes

- cmdr is the **human-facing** layer — it translates human intent into protocol messages
- cmdr **launches and manages** connectors as child processes
- cmdr doesn't know how to transport messages — it delegates to connectors
- cmdr doesn't know how to execute agent tasks — that's the unit's job
- cmdr only knows the protocol and how to orchestrate
