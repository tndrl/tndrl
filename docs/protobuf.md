# Protobuf & buf

Latis uses [Protocol Buffers](https://protobuf.dev/) for control plane message definitions and [buf](https://buf.build/) for tooling.

## Protocol Overview

Latis has two distinct communication protocols:

| Protocol | Purpose | Implementation |
|----------|---------|----------------|
| **Control** | Node lifecycle (ping, status, shutdown) | Custom protobuf in `proto/latis/v1/control.proto` |
| **A2A** | Agent communication (prompts, tasks) | [a2a-go](https://github.com/a2aproject/a2a-go) library |

The Control protocol is defined in this repository. A2A protocol definitions come from the upstream `a2a-go` library.

## Directory Structure

```
latis/
├── proto/
│   └── latis/v1/
│       └── control.proto     # Control plane service definition
├── gen/
│   └── go/
│       └── latis/v1/         # Generated Go code (committed)
│           ├── control.pb.go
│           └── control_grpc.pb.go
├── buf.yaml                  # Module configuration
└── buf.gen.yaml              # Code generation configuration
```

## Control Protocol

The control protocol handles node lifecycle operations. It runs on a dedicated QUIC stream (type=0x01), separate from A2A traffic.

### Service Definition

```protobuf
service ControlService {
  rpc Ping(PingRequest) returns (PingResponse);
  rpc GetStatus(GetStatusRequest) returns (GetStatusResponse);
  rpc Shutdown(ShutdownRequest) returns (ShutdownResponse);
}
```

### Usage in Go

**Server side (in `latis serve`):**

```go
import (
    "google.golang.org/grpc"
    latisv1 "github.com/shanemcd/latis/gen/go/latis/v1"
    "github.com/shanemcd/latis/pkg/control"
)

// Create and register the control service
grpcServer := grpc.NewServer()
controlService := control.NewService(state)
latisv1.RegisterControlServiceServer(grpcServer, controlService)
```

**Client side (ping, status, shutdown commands):**

```go
import (
    "google.golang.org/grpc"
    latisv1 "github.com/shanemcd/latis/gen/go/latis/v1"
)

// Create client from gRPC connection
conn, _ := grpc.NewClient(addr, opts...)
client := latisv1.NewControlServiceClient(conn)

// Ping
resp, _ := client.Ping(ctx, &latisv1.PingRequest{
    Timestamp: time.Now().UnixNano(),
})
latency := time.Duration(resp.PongTimestamp - resp.PingTimestamp)

// Get status
status, _ := client.GetStatus(ctx, &latisv1.GetStatusRequest{})
fmt.Printf("State: %s, Uptime: %ds\n", status.State, status.UptimeSeconds)

// Shutdown
_, _ = client.Shutdown(ctx, &latisv1.ShutdownRequest{
    Graceful:       true,
    TimeoutSeconds: 30,
    Reason:         "requested by peer",
})
```

## A2A Protocol

Agent-to-agent communication uses the [A2A protocol](https://a2a-protocol.org/) via the `a2a-go` library. This is not defined in custom protobufs — we use the upstream library directly.

```go
import (
    "github.com/a2aproject/a2a-go/a2a"
    "github.com/a2aproject/a2a-go/a2aclient"
)

// A2A types come from the library
card := &a2a.AgentCard{
    Name:        "my-agent",
    Description: "An AI agent",
    // ...
}

// A2A client for sending messages
client := a2aclient.NewGRPCTransport(conn)
```

See the [a2a-go documentation](https://github.com/a2aproject/a2a-go) for details.

## Why buf?

We use buf instead of raw protoc because:

- **Better DX** — simpler commands, better error messages
- **Built-in linting** — enforces style consistency
- **Breaking change detection** — catches API breaks before they ship
- **Remote plugins** — no need to install protoc plugins locally
- **Managed mode** — automatically sets go_package without manual annotations

## Commands

```bash
# Lint proto files
buf lint

# Generate Go code
buf generate

# Check for breaking changes against git main
buf breaking --against '.git#branch=main'

# Format proto files
buf format -w
```

## Generated Code

Running `buf generate` produces:

| File | Contents |
|------|----------|
| `gen/go/latis/v1/control.pb.go` | Protobuf message types |
| `gen/go/latis/v1/control_grpc.pb.go` | gRPC client/server interfaces |

Import in Go:

```go
import latisv1 "github.com/shanemcd/latis/gen/go/latis/v1"
```

## Configuration

### buf.yaml

```yaml
version: v2
modules:
  - path: proto
lint:
  use:
    - STANDARD      # Enforces naming, package structure, etc.
breaking:
  use:
    - FILE          # Detects breaking changes per-file
```

### buf.gen.yaml

```yaml
version: v2
managed:
  enabled: true
  override:
    - file_option: go_package_prefix
      value: github.com/shanemcd/latis/gen/go
plugins:
  - remote: buf.build/protocolbuffers/go   # Protobuf messages
    out: gen/go
    opt: paths=source_relative
  - remote: buf.build/grpc/go              # gRPC services
    out: gen/go
    opt: paths=source_relative
```

## Extending the Control Protocol

### Adding a New RPC

1. Add the RPC to `proto/latis/v1/control.proto`
2. Define request/response messages
3. Run `buf lint` and `buf generate`
4. Implement in `pkg/control/control.go`
5. Commit both `.proto` and generated files

### Field Number Guidelines

- `1-15`: Use for frequently-used fields (1 byte to encode)
- `16-2047`: Standard fields (2 bytes)
- Reserve field numbers from deleted fields: `reserved 5, 6;`

## Breaking Changes

Before merging PRs that modify protos:

```bash
buf breaking --against '.git#branch=main'
```

This catches:
- Removed fields
- Changed field numbers
- Type changes
- Removed enum values

**Safe changes (non-breaking):**
- Adding new fields (use new field numbers)
- Adding new values to enums
- Adding new RPCs

**Breaking changes (avoid):**
- Removing fields
- Changing field numbers
- Changing field types
