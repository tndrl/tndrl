# cmdr

The control plane and CLI for Latis. This is the human interface layer.

## Responsibilities

- **Human interface**: CLI (and eventually TUI/GUI/web) for interacting with agents
- **Unit provisioning**: Create, start, stop, destroy units via pluggable provisioners
- **Connection management**: Dial out to units OR accept dial-ins from units
- **Session management**: Create, resume, destroy agent sessions
- **Routing**: Direct messages to the right units
- **Orchestration**: Coordinate multi-agent tasks
- **State tracking**: Know what's running where

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                         cmdr                            │
├─────────────────────────────────────────────────────────┤
│  Provisioners          │  Dialers        │  Listeners   │
│  (create units)        │  (cmdr → unit)  │  (unit → cmdr)│
│  ┌──────────────────┐  │  ┌───────────┐  │  ┌─────────┐ │
│  │ process          │  │  │ ssh       │  │  │ tcp     │ │
│  │ container        │  │  │ tcp       │  │  │ ws      │ │
│  │ qemu / tart      │  │  │ websocket │  │  │ quic    │ │
│  │ cloud-*          │  │  │ local     │  │  │ unix    │ │
│  └──────────────────┘  │  └───────────┘  │  └─────────┘ │
└─────────────────────────────────────────────────────────┘
                              │                   │
                              ↓                   ↓
                         ┌────────┐          ┌────────┐
                         │  unit  │          │  unit  │
                         └────────┘          └────────┘
                      (cmdr dialed)      (unit dialed in)
```

## Provisioners

Provisioners handle unit lifecycle — creating, starting, stopping, destroying units. Pluggable.

| Provisioner | Description |
|-------------|-------------|
| `process` | Spawn a local process |
| `container` | Run in podman/docker container |
| `qemu` | Start a QEMU VM |
| `tart` | Start a Tart VM (macOS) |
| `libvirt` | Manage via libvirt |
| `cloud-*` | Cloud provider APIs (AWS, GCP, etc.) |

Provisioners are decoupled from connection — a provisioned unit still needs to establish a connection via dialers or listeners.

## Dialers

Dialers establish outbound connections from cmdr to units. Used when cmdr initiates.

| Dialer | Description |
|--------|-------------|
| `ssh` | SSH into remote host, communicate via stdin/stdout |
| `tcp` | Direct TCP connection |
| `websocket` | WebSocket connection |
| `local` | Stdin/stdout to spawned process |
| `container` | Exec into container |

## Listeners

Listeners accept inbound connections from units. Used when units dial out to cmdr.

| Listener | Description |
|----------|-------------|
| `tcp` | Accept TCP connections |
| `websocket` | Accept WebSocket connections |
| `quic` | Accept QUIC connections |
| `unix` | Accept Unix socket connections |

Listeners are useful for:
- NAT traversal (unit behind firewall dials out)
- Ephemeral cloud instances (provision, then unit calls home)
- Dynamic environments where unit addresses aren't known upfront

## Connection Constraint

At least one connection method must be configured per unit:
- If no dialers configured → must have at least one listener
- A unit can support both (dial AND listen)
- Multiple listeners allowed (unit reachable multiple ways)

## Unit Configuration Examples

**Local process (cmdr spawns and dials via stdio):**
```yaml
unit: dev-local
provisioner: process
  command: latis-unit
connection:
  dial:
    via: local  # stdin/stdout
```

**Remote server (cmdr dials via SSH):**
```yaml
unit: prod-server
# no provisioner — assume unit already running
connection:
  dial:
    via: ssh
    address: user@prod.example.com
```

**Local VM (cmdr provisions and dials):**
```yaml
unit: gpu-box
provisioner: qemu
  image: /path/to/vm.qcow2
  memory: 16G
connection:
  dial:
    via: tcp
    address: localhost:9000  # VM exposes port
```

**Cloud instance (provision, unit dials back):**
```yaml
unit: ephemeral-worker
provisioner: cloud-aws
  instance_type: g4dn.xlarge
  ami: ami-xxxxx
connection:
  accept:
    via: websocket
    # unit will dial wss://cmdr.example.com/units
    # and identify itself
```

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
- cmdr **provisions** units via pluggable provisioners
- cmdr **dials** or **listens** to establish connections with units
- cmdr doesn't know how to execute agent tasks — that's the unit's job
- cmdr only knows the protocol and how to orchestrate
