# Policy

Authorization and policy evaluation for latis.

## Why Policy?

Policy governs:
- Who can connect to which units
- What tools an agent is allowed to invoke
- Resource limits (token budgets, tool call limits)
- When human approval is required
- Execution permissions

## Options Evaluated

| Solution | Language | Pros | Cons |
|----------|----------|------|------|
| **OPA** | Rego | Mature, pure Go, large ecosystem | Rego learning curve |
| **Cedar** | Cedar | Clean model, formal verification | Rust core, CGO needed for Go |
| **Casbin** | Various | Multi-language, simple models | Less expressive |
| **Custom** | YAML/JSON | Simple, no dependencies | Have to build evaluator |

## Recommendation: OPA (when needed)

**Why OPA:**
- Pure Go — embeds cleanly, no CGO/Rust dependency
- Battle-tested at scale (Kubernetes, Envoy, Terraform)
- Better tooling (playground, IDE support, `opa test`)
- Larger community, more examples

**Why "when needed":**
- Policy adds complexity before core is working
- Start with simple/hardcoded config
- Add OPA when patterns stabilize

## Example Policy (future)

```rego
package latis

default allow = false

allow {
    input.user == "admin"
}

allow {
    input.action == "prompt.send"
    input.unit.name == input.user.allowed_units[_]
}

# Tool restrictions
allow_tool {
    input.tool.name == "kubectl"
    input.unit.capabilities[_] == "k8s"
}

# Require approval for destructive actions
require_approval {
    input.tool.name == "kubectl"
    input.tool.args[0] == "delete"
}
```

## Integration Pattern

```go
import "github.com/open-policy-agent/opa/rego"

func (e *Evaluator) Allowed(ctx context.Context, input PolicyInput) (bool, error) {
    results, err := e.query.Eval(ctx, rego.EvalInput(input))
    if err != nil {
        return false, err
    }
    return results.Allowed(), nil
}
```

## Policy Questions to Answer

As latis develops, policy should answer:

1. **Connection**: Can user X connect to unit Y?
2. **Actions**: Can this session send prompts? Subscribe to state?
3. **Tools**: Can this agent invoke tool Z?
4. **Limits**: What are the resource limits for this session?
5. **Approval**: Does this action require human approval?
6. **Audit**: What should be logged?

## Status

Deferred. Build core first, add policy when authorization patterns emerge.
