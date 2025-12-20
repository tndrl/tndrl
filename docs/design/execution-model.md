# Execution Model

How units execute work and interact with their environment.

## Tools

Units don't just relay messages — they act. Tools are how agents affect the world.

```
┌────────────┐     structured      ┌────────────┐
│   agent    │ ←── request/resp ──→│   tools    │
└────────────┘                     └────────────┘
                                    ├── k8s/OCP API
                                    ├── file system
                                    ├── shell
                                    ├── HTTP
                                    └── ...
```

Tools provide:
- Structured input (agent calls tool with parameters)
- Structured output (tool returns result)
- A way to teach agents capabilities via context + tool definitions

This aligns with MCP (Model Context Protocol) server concepts.

## Execution Modes

Units can operate in different modes:

| Mode | Behavior |
|------|----------|
| **Reactive** | Single-shot: prompt → response → done |
| **Agentic** | Loop: prompt → (think → act → observe)* → response |
| **Continuous** | Runs until explicitly stopped |

## Control Flow Challenge

**The open question:** When does an agent stop and wait for input vs. continue acting autonomously?

### Possible Yield Points

| Trigger | Behavior |
|---------|----------|
| Task complete | Agent decides it's done |
| Needs clarification | Agent is uncertain, asks |
| Resource limit | Token budget, time limit, tool call limit |
| Explicit checkpoint | Pre-defined pause points |
| Error/exception | Something went wrong |
| Human interrupt | User sends cancel/pause |

### Who Decides "Done"?

- **Agent decides**: Risky — may loop forever or stop prematurely
- **Human decides**: Safe but slow — defeats autonomy
- **Hybrid**: Agent proposes, human approves (or timeout auto-approves)

### Possible Configuration

```yaml
unit: ocp-agent
execution:
  mode: agentic
  yield_on:
    - needs_input
    - task_complete
    - tool_limit: 10
    - token_limit: 50000
  auto_continue: false  # require human approval to resume
```

## Real-World Example: CRC + OpenShift

```
cmdr
  └── provisioner: libvirt (via CRC)
        └── unit running in CRC VM
              └── tools: OCP API (pods, deployments, etc.)
```

A unit inside CRC (OpenShift Local) can:
- Manipulate Kubernetes/OpenShift resources
- Access the container runtime
- Operate with real capabilities, not just chat

This raises the question of how units interact with their environment — they need tools, not just prompts.

## Status

Open design area. Needs experimentation to find the right model.
