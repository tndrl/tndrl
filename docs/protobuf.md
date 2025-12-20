# Protobuf & buf

Latis uses [Protocol Buffers](https://protobuf.dev/) for message definitions and [buf](https://buf.build/) for tooling.

## How It All Connects

The protobuf definitions in `proto/` generate Go code that both cmdr and unit import and use directly:

```
proto/latis/v1/latis.proto
        │
        │ buf generate
        ▼
gen/go/latis/v1/
├── latis.pb.go          # Message types (ConnectRequest, Ping, etc.)
└── latis_grpc.pb.go     # Service interface (LatisServiceServer, LatisServiceClient)
        │
        │ imported by
        ▼
┌───────────────────┐              ┌───────────────────┐
│ cmd/latis/        │              │ cmd/latis-unit/   │
│                   │              │                   │
│ latisv1.New...    │   gRPC/QUIC  │ latisv1.Register  │
│ LatisServiceClient│◄────────────►│ LatisServiceServer│
└───────────────────┘              └───────────────────┘
```

### Server Side (unit)

The unit implements the generated `LatisServiceServer` interface:

```go
import latisv1 "github.com/shanemcd/latis/gen/go/latis/v1"

// Embed the unimplemented server for forward compatibility
type server struct {
    latisv1.UnimplementedLatisServiceServer
}

// Implement the Connect RPC method
func (s *server) Connect(stream latisv1.LatisService_ConnectServer) error {
    for {
        req, err := stream.Recv()  // Receive ConnectRequest
        // ...

        // Type switch on the oneof payload
        switch payload := req.Payload.(type) {
        case *latisv1.ConnectRequest_Ping:
            // Handle ping, send pong
            stream.Send(&latisv1.ConnectResponse{
                Id: req.Id,
                Payload: &latisv1.ConnectResponse_Pong{...},
            })
        case *latisv1.ConnectRequest_PromptSend:
            // Handle prompt, stream response chunks
        }
    }
}

// Register with gRPC server
grpcServer := grpc.NewServer()
latisv1.RegisterLatisServiceServer(grpcServer, &server{})
```

### Client Side (cmdr)

The cmdr uses the generated client stub:

```go
import latisv1 "github.com/shanemcd/latis/gen/go/latis/v1"

// Create client from gRPC connection
client := latisv1.NewLatisServiceClient(conn)

// Open bidirectional stream
stream, _ := client.Connect(ctx)

// Send a request
stream.Send(&latisv1.ConnectRequest{
    Id: uuid.New().String(),
    Payload: &latisv1.ConnectRequest_Ping{
        Ping: &latisv1.Ping{Timestamp: time.Now().UnixNano()},
    },
})

// Receive response
resp, _ := stream.Recv()
switch payload := resp.Payload.(type) {
case *latisv1.ConnectResponse_Pong:
    fmt.Printf("latency: %v", time.Duration(time.Now().UnixNano() - payload.Pong.PingTimestamp))
}
```

### Key Patterns

1. **Oneof for polymorphism** — `ConnectRequest.Payload` and `ConnectResponse.Payload` use `oneof` to support multiple message types over one stream
2. **Type switches** — Go code uses type switches to handle different payload types
3. **ID correlation** — Every request has an `id`, responses reference it for async correlation
4. **Embedding UnimplementedServer** — Ensures forward compatibility when new RPCs are added

## Why buf?

We use buf instead of raw protoc because:

- **Better DX** — simpler commands, better error messages
- **Built-in linting** — enforces style consistency
- **Breaking change detection** — catches API breaks before they ship
- **Remote plugins** — no need to install protoc plugins locally
- **Managed mode** — automatically sets go_package without manual annotations

## Directory Structure

```
latis/
├── proto/
│   └── latis/v1/
│       └── latis.proto      # Service and message definitions
├── gen/
│   └── go/
│       └── latis/v1/        # Generated Go code (committed)
├── buf.yaml                 # Module configuration
└── buf.gen.yaml             # Code generation configuration
```

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
| `gen/go/latis/v1/latis.pb.go` | Protobuf messages |
| `gen/go/latis/v1/latis_grpc.pb.go` | gRPC client/server interfaces |

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

## Workflow

1. Edit `proto/latis/v1/latis.proto`
2. Run `buf lint` to check for issues
3. Run `buf generate` to regenerate Go code
4. Commit both `.proto` and generated `gen/` files

## Extending the Protocol

### Adding a New Message Type

Example: Adding a `ToolCall` request that units can send to request tool execution.

**Step 1: Define the message in `proto/latis/v1/latis.proto`**

```protobuf
// Add the message definition
message ToolCall {
  string tool_name = 1;
  string arguments = 2;  // JSON-encoded arguments
}
```

**Step 2: Add to the appropriate oneof**

```protobuf
message ConnectRequest {
  string id = 1;
  oneof payload {
    PromptSend prompt_send = 10;
    // ... existing messages ...
    ToolCall tool_call = 14;  // New! Use next available field number
  }
}
```

**Step 3: Regenerate code**

```bash
buf lint      # Check for style issues
buf generate  # Regenerate Go code
```

**Step 4: Handle in application code**

In `cmd/latis-unit/main.go`:
```go
case *latisv1.ConnectRequest_ToolCall:
    log.Printf("tool call: %s(%s)", payload.ToolCall.ToolName, payload.ToolCall.Arguments)
    // Execute tool, send response...
```

**Step 5: Commit both proto and generated code**

```bash
git add proto/ gen/
git commit -m "Add ToolCall message type"
```

### Modifying Existing Messages

**Safe changes (non-breaking):**
- Adding new fields (use new field numbers)
- Adding new values to enums
- Adding new message types to oneof

**Breaking changes (avoid):**
- Removing fields
- Changing field numbers
- Renaming fields (wire format uses numbers, not names, but breaks code)
- Changing field types

Check for breaking changes before merging:
```bash
buf breaking --against '.git#branch=main'
```

### Field Number Guidelines

- `1-15`: Use for frequently-used fields (1 byte to encode)
- `16-2047`: Standard fields (2 bytes)
- Reserve field numbers from deleted fields: `reserved 5, 6;`
- Reserve names if you want to prevent reuse: `reserved "old_field_name";`

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

## CI Integration

The `buf lint` and `buf breaking` checks should be added to CI. For now, run manually before committing proto changes.
