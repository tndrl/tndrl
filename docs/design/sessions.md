# Sessions

Design document for latis session management — orchestrating isolated agent environments.

## Motivation

The core use case: "turn loose" an AI agent to accomplish a task in a sandboxed environment where it can iterate autonomously, without risking unintended changes to the host machine. The agent works until it reaches a point where it needs human input, then waits.

This requires:
- **Isolation** — Agent runs in a container/VM, can't touch the host
- **Autonomy** — Agent works independently until it genuinely needs input
- **Interruptibility** — Human can check in, provide input, detach
- **Persistence** — Session state survives disconnects

## Concepts

### Session

A session is a unit of work with:
- A provisioned environment (container, VM, pod)
- A running latis node inside that environment
- Conversation history (multi-turn context)
- A task description
- Lifecycle state

### Driver

A driver provisions and manages execution environments. Different drivers support different platforms (podman, kubernetes, qemu, etc.). Each driver has platform-specific configuration options.

### Session States

```
┌─────────┐
│ creating│ ── driver provisioning environment
└────┬────┘
     │
     ▼
┌─────────┐
│ starting│ ── latis node booting, connecting
└────┬────┘
     │
     ▼
┌─────────┐     ┌─────────┐
│ working │◄───►│ waiting │ ── agent blocked on human input
└────┬────┘     └─────────┘
     │
     ▼
┌─────────┐
│ complete│ ── task finished (success or failure)
└────┬────┘
     │
     ▼
┌─────────┐
│ stopped │ ── environment torn down
└─────────┘
```

**State transitions:**
- `creating` → `starting`: Environment provisioned
- `starting` → `working`: Node ready, begins task
- `working` → `waiting`: Agent needs human input
- `waiting` → `working`: Human provides input
- `working` → `complete`: Task finished
- Any state → `stopped`: Session destroyed

## CLI Interface

### Create a session

```bash
# Minimal
latis session create --task "Fix the failing tests"

# With driver options
latis session create \
    --driver podman \
    --image ghcr.io/shanemcd/latis-workspace:latest \
    --mount ./myrepo:/workspace \
    --task "Fix the failing tests in this repo"

# From a config file
latis session create -c session.yaml
```

### List sessions

```bash
$ latis session list
ID       DRIVER   STATUS    TASK                        AGE
s-7f3a   podman   working   Fix the failing tests       2m
s-9b2c   k8s      waiting   Refactor auth module        1h
s-1d4e   podman   complete  Add unit tests              3h
```

### Attach to a session

```bash
$ latis session attach s-7f3a

# Interactive conversation with the agent
Agent: I fixed 2 of 3 failing tests. TestDatabaseMigration requires
a decision: the test assumes PostgreSQL 14 but your CI uses 15.
Should I update the test or the CI config?

You: Update the CI config

Agent: On it...

# Ctrl-D to detach, agent continues working
```

### Other commands

```bash
# View session logs
latis session logs s-7f3a

# Get session status/details
latis session status s-7f3a

# Delete a session (tears down environment)
latis session delete s-7f3a

# Delete all completed sessions
latis session prune
```

## Driver Interface

```go
// Driver provisions and manages execution environments.
type Driver interface {
    // Name returns the driver identifier (e.g., "podman", "kubernetes").
    Name() string

    // Create provisions a new environment and starts a latis node.
    // Returns the session ID and endpoint to connect to the node.
    Create(ctx context.Context, opts CreateOptions) (*Environment, error)

    // Destroy tears down the environment.
    Destroy(ctx context.Context, id string) error

    // Status returns the current state of the environment.
    Status(ctx context.Context, id string) (EnvironmentStatus, error)

    // Logs returns a reader for environment logs.
    Logs(ctx context.Context, id string, opts LogOptions) (io.ReadCloser, error)

    // Exec runs a command in the environment (for debugging).
    Exec(ctx context.Context, id string, cmd []string) error
}

type CreateOptions struct {
    // Task is the initial task description for the agent.
    Task string

    // Image is the container/VM image to use.
    Image string

    // Mounts are host paths to mount into the environment.
    Mounts []Mount

    // DriverConfig is driver-specific configuration (raw YAML/JSON).
    DriverConfig map[string]any
}

type Environment struct {
    // ID is the unique session identifier.
    ID string

    // Endpoint is the address to connect to the latis node.
    Endpoint string

    // Driver is the driver that created this environment.
    Driver string
}

type EnvironmentStatus struct {
    State     string    // creating, starting, running, stopped, error
    StartedAt time.Time
    Error     string    // if State == "error"
}
```

## Drivers

### Podman

Runs agents in podman containers. Good for local development and single-machine deployments.

```yaml
driver: podman
podman:
  image: ghcr.io/shanemcd/latis-workspace:latest
  mounts:
    - ./repo:/workspace:Z
  ports:
    - "4433"  # latis node port, auto-assigned host port
  flags:
    - "--userns=keep-id"
    - "--security-opt=label=disable"
  env:
    ANTHROPIC_VERTEX_PROJECT_ID: my-project
```

**Implementation:** Shells out to `podman run`, `podman stop`, `podman logs`, etc.

### Kubernetes

Runs agents as pods in a Kubernetes cluster. Good for teams and production deployments.

```yaml
driver: kubernetes
kubernetes:
  namespace: latis-sessions
  serviceAccount: latis-agent
  podSpec:
    containers:
      - name: agent
        image: ghcr.io/shanemcd/latis-workspace:latest
        resources:
          requests:
            memory: "2Gi"
            cpu: "1"
          limits:
            memory: "4Gi"
            cpu: "2"
        volumeMounts:
          - name: workspace
            mountPath: /workspace
    volumes:
      - name: workspace
        persistentVolumeClaim:
          claimName: my-repo-pvc
```

**Implementation:** Uses client-go to create/delete pods, services for connectivity.

### QEMU

Runs agents in virtual machines. Maximum isolation, good for untrusted workloads.

```yaml
driver: qemu
qemu:
  image: /var/lib/latis/images/fedora-dev.qcow2
  memory: 4G
  cpus: 2
  flags:
    - "-enable-kvm"
  ssh:
    user: latis
    keyFile: ~/.ssh/latis_ed25519
```

**Implementation:** Manages QEMU processes, SSH for connectivity.

### Local

No isolation — runs directly on host. Useful for development and testing.

```yaml
driver: local
local:
  workdir: /tmp/latis-workspace
```

**Implementation:** Just starts `latis serve` as a subprocess.

### SSH

Connects to an existing remote machine. No provisioning, just connection.

```yaml
driver: ssh
ssh:
  host: dev-server.example.com
  user: latis
  keyFile: ~/.ssh/id_ed25519
  # Assumes latis is already installed on remote
```

## Session Storage

Sessions need persistent storage for:
- Session metadata (ID, driver, task, state, created_at)
- Conversation history
- Connection info

**Local storage:** `~/.latis/sessions/`

```
~/.latis/sessions/
├── s-7f3a/
│   ├── meta.json       # session metadata
│   ├── history.json    # conversation history
│   └── driver.json     # driver-specific state
├── s-9b2c/
│   └── ...
```

**Future:** Optional remote storage (PostgreSQL, SQLite, etc.) for shared access.

## Conversation History

Each session maintains conversation history for multi-turn interactions:

```go
type Message struct {
    Role      string    `json:"role"`      // "user", "assistant", "system"
    Content   string    `json:"content"`
    Timestamp time.Time `json:"timestamp"`
}

type History struct {
    Messages []Message `json:"messages"`
}
```

When attaching to a session, the CLI shows recent history for context.

The latis node inside the environment needs access to this history to maintain context across attach/detach cycles. Options:
1. **Push model:** CLI sends history with each message
2. **Pull model:** Node fetches history from shared storage
3. **Local model:** Node maintains its own history, synced on attach

## Waiting for Input

When an agent needs human input, it signals the "waiting" state. This requires a protocol addition:

**Option A: Explicit RPC**

Add a `RequestInput` RPC to the A2A or Control service:

```protobuf
message RequestInputRequest {
    string prompt = 1;        // What the agent needs
    repeated string options = 2;  // Optional: suggested choices
}

message RequestInputResponse {
    string input = 1;
}
```

**Option B: Task state**

Use A2A task states — agent creates a task in `input-required` state. Polling or streaming surfaces this to the CLI.

**Option C: Streaming with markers**

In streaming responses, agent emits a special marker indicating it's waiting. CLI detects this and prompts user.

## Example Workflow

```bash
# 1. Create a session
$ latis session create \
    --driver podman \
    --image ghcr.io/shanemcd/latis-workspace:latest \
    --mount ./myrepo:/workspace \
    --task "Fix the failing tests in this repo and open a PR"

Creating session s-7f3a...
Pulling image ghcr.io/shanemcd/latis-workspace:latest...
Starting container...
Waiting for latis node...
Connected.

Session s-7f3a created.
Agent is analyzing the codebase...

# 2. Check status later
$ latis session list
ID       DRIVER   STATUS    TASK                        AGE
s-7f3a   podman   waiting   Fix the failing tests...    12m

# 3. Attach to provide input
$ latis session attach s-7f3a

[Session s-7f3a - waiting for input]

Agent: I've fixed 4 of 5 failing tests. The last one, TestOAuthFlow,
requires credentials I don't have access to. Options:

1. Skip this test for now and note it in the PR
2. Mock the OAuth provider
3. You provide test credentials

What would you prefer?

You: Mock the OAuth provider

Agent: Got it. I'll create a mock OAuth server for the tests...

[Ctrl-D to detach]

# 4. Agent continues working, eventually completes
$ latis session list
ID       DRIVER   STATUS    TASK                        AGE
s-7f3a   podman   complete  Fix the failing tests...    47m

# 5. View the result
$ latis session attach s-7f3a

Agent: Done! I've opened PR #42: "Fix failing tests"
https://github.com/you/myrepo/pull/42

Summary:
- Fixed TestDatabaseMigration: updated CI to PostgreSQL 15
- Fixed TestCacheExpiry: race condition in timer
- Fixed TestOAuthFlow: added mock OAuth server
- Fixed TestRateLimiter: flaky timing assertion
- Fixed TestWebhook: missing Content-Type header

All tests passing. PR is ready for review.

# 6. Clean up
$ latis session delete s-7f3a
Session s-7f3a deleted.
```

## Workspace Images

Pre-built container images with common development tools:

| Image | Contents |
|-------|----------|
| `latis-workspace:base` | latis binary, basic shell tools |
| `latis-workspace:go` | Go toolchain, gopls, common tools |
| `latis-workspace:python` | Python, pip, common packages |
| `latis-workspace:node` | Node.js, npm, common packages |
| `latis-workspace:full` | All of the above |

Images include:
- `latis` binary configured to start on boot
- Git, curl, jq, common CLI tools
- Language-specific toolchains
- Pre-configured for non-root user

## Security Considerations

- **Container isolation:** Agents run in containers with no host access by default
- **Mount controls:** Only explicitly mounted paths are accessible
- **Network isolation:** Optional network policies to restrict agent's network access
- **Secret injection:** Secrets passed via environment, not mounted files
- **Audit logging:** All agent actions logged for review
- **Resource limits:** CPU/memory limits prevent runaway processes

## Implementation Phases

### Phase 1: Core session lifecycle
- Session create/list/delete with podman driver
- Basic conversation history (local storage)
- Attach/detach with interactive terminal

### Phase 2: Waiting for input
- Protocol for agent to signal "waiting"
- CLI detection and prompting
- Background status polling

### Phase 3: Additional drivers
- Kubernetes driver
- Local driver (for testing)
- QEMU driver (stretch)

### Phase 4: Polish
- Workspace images
- Session templates/presets
- Remote storage for team sharing

## Open Questions

1. **Multi-node sessions:** Should a session be able to orchestrate multiple latis nodes? (e.g., one for code analysis, one for testing)

2. **Session templates:** Predefined session configs for common workflows?

3. **Web UI:** Eventually expose session management via web interface?

4. **Billing/quotas:** For shared deployments, track resource usage per session?

5. **Session sharing:** Allow multiple users to attach to the same session?
