# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Finta is an AI Agent framework inspired by ClaudeCode's design philosophy. It provides a modular, extensible foundation for building AI agents that can execute tools, interact with LLMs, and handle complex multi-turn conversations.

**Current Status**: Phase 2 complete (parallel tool execution & streaming support) + Reasoning support
**Go Version**: 1.24.5
**Primary LLM Integration**: OpenAI API (with Extended Thinking / Reasoning)

## Build and Development Commands

### Build
```bash
# Build the CLI binary
go build -o finta cmd/finta/main.go
```

### Run
```bash
# Basic usage (requires OPENAI_API_KEY environment variable)
export OPENAI_API_KEY="your-key"
./finta chat "Your task here"

# With verbose logging
./finta chat --verbose "Your task"

# Custom model and parameters
./finta chat "task" --model gpt-4o --temperature 0.5 --max-turns 20

# Disable colored output (for logs)
./finta chat --no-color "task" > log.txt
```

### Available CLI Flags
- `--api-key` - OpenAI API key (or env: OPENAI_API_KEY)
- `--api-base-url` - Custom API endpoint (or env: OPENAI_API_BASE_URL)
- `--model` - Model to use (default: gpt-4-turbo)
- `--temperature` - Temperature parameter (default: 0.7)
- `--max-turns` - Max conversation turns (default: 10)
- `--verbose` - Enable debug logging
- `--no-color` - Disable colored output
- `--streaming` - Enable streaming output (default: false)
- `--parallel` - Enable parallel tool execution (default: true)

## Architecture

### Core Design Pattern: Interface-Driven Modularity

The framework follows a strict interface-based architecture where all major components (Agent, LLM Client, Tools, Logger) are defined by interfaces in `internal/*/` packages. This allows for:
- Easy testing via mocks
- Multiple implementations (e.g., different LLM providers)
- Clear separation of concerns

### Reasoning Support (Extended Thinking)

**NEW**: The framework now supports LLM reasoning/thinking process:
- **Message.Reason**: Stores the LLM's internal reasoning
- **Message.Content**: Stores the final answer
- **Complete History**: Reasoning is preserved and sent back to LLM in subsequent turns
- **Visual Separation**: Reasoning (💭 yellow) vs Response (💬 green)
- **Streaming**: Reasoning is streamed in real-time

See `REASONING_SUPPORT.md` for detailed documentation.

### Critical Data Flow: Agent Run Loop

The heart of the system is in `internal/agent/base.go`:

```
User Input → BaseAgent.Run() → Loop (up to MaxTurns):
  1. Call LLM with conversation history + tool definitions
  2. If StopReasonStop → Return final response
  3. If StopReasonToolCalls → Execute tools → Add results to history → Continue
  4. If StopReasonLength → Return truncated response
```

Key insight: The agent maintains a growing message slice that includes system prompt, user messages, assistant responses, and tool results. Each tool call result becomes a new message with `Role: RoleTool`.

### Tool System Architecture

Tools are registered in a thread-safe `Registry` (`internal/tool/registry.go`):
- **Registration**: Tools must implement `Tool` interface (Name, Description, Parameters, Execute)
- **Execution**: BaseAgent fetches tools from registry and executes them with JSON parameters
- **LLM Integration**: Registry converts tools to `ToolDefinition` format for OpenAI API

Tool execution flow in `base.go`:
1. LLM returns tool calls with function name + JSON arguments
2. Agent looks up tool in registry
3. Tool executes with context + parameters
4. Result (with timing) is logged and added to conversation

### Logger Integration Pattern

**Critical**: All agent execution MUST receive a `Logger` in the `Input` struct. The logger is NOT a field of BaseAgent - it's injected per-run to allow different logging configurations per execution.

Execution context (`internal/agent/context.go`) wraps the logger and tracks:
- Current turn number
- Total tool calls made
- Session start time

This pattern enables rich logging without polluting the agent's core logic.

### Package Import Rules

**IMPORTANT**: The codebase uses `internal/` instead of `pkg/` (changed in commit f62c5f3).

Import paths must be:
```go
import (
    "finta/internal/agent"
    "finta/internal/llm"
    "finta/internal/logger"
    "finta/internal/tool"
)
```

NOT `finta/pkg/*` - that directory no longer exists.

## Key Architectural Decisions

### 1. Why Logger is in Input, not BaseAgent
Logger is passed per-execution to allow different logging levels/outputs for different runs of the same agent. This enables testing with different verbosity without creating new agent instances.

### 2. Tool Registry is Shared
The `Registry` is passed to BaseAgent at construction and shared across all runs. Tools are stateless and thread-safe. This means you can register tools once and reuse the agent multiple times.

### 3. Message History Management
The agent maintains the FULL conversation history (including tool results) in memory. There's no automatic summarization yet (planned for Phase 6). For long conversations, this can consume significant memory and context window.

### 4. Error Handling in Tool Execution
Tool execution failures don't terminate the agent run. Failed tools return a `Result{Success: false, Error: "..."}` which is passed back to the LLM. The LLM can then retry, use a different tool, or acknowledge the failure.

### 5. JSON Parameter Formatting (Logging)
The logger uses adaptive JSON formatting:
- Short params (< 80 chars): Single line
- Long params (≥ 80 chars): Pretty-printed with indentation

This is implemented in `logger.formatJSON()` and significantly improves log readability.

## Critical Files to Understand

### `internal/agent/base.go` - The Agent Loop
Contains the core run loop that orchestrates LLM calls and tool execution. Understanding this file is essential for any agent behavior modifications.

Key methods:
- `Run()` - Main execution loop
- `executeToolsWithLogging()` - Tool execution with comprehensive logging

### `internal/llm/openai/client.go` - LLM Integration
Handles conversion between internal message types and OpenAI API format. Critical for understanding:
- How tool definitions are sent to OpenAI
- How tool calls are parsed from responses
- Message format conversions

### `internal/tool/registry.go` - Tool Management
Thread-safe tool registry. Implements:
- `Register()` - Add tools (errors if duplicate name)
- `Get()` - Retrieve tool by name
- `GetToolDefinitions()` - Convert to LLM-compatible format

### `internal/logger/logger.go` - Structured Logging
Provides the comprehensive logging system with:
- ANSI color codes for terminal output
- Structured sections for tool calls, results, and responses
- Session start/end banners with statistics

## Extending the Framework

### Adding a New Tool

1. Create a new file in `internal/tool/builtin/`
2. Implement the `Tool` interface:
```go
type YourTool struct{}

func (t *YourTool) Name() string { return "your_tool" }
func (t *YourTool) Description() string { return "What it does" }
func (t *YourTool) Parameters() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "param_name": map[string]any{
                "type": "string",
                "description": "Parameter description",
            },
        },
        "required": []string{"param_name"},
    }
}
func (t *YourTool) Execute(ctx context.Context, params json.RawMessage) (*tool.Result, error) {
    // Implementation
}
```
3. Register it in `cmd/finta/main.go`:
```go
registry.Register(builtin.NewYourTool())
```

### Adding a New LLM Provider

1. Create directory `internal/llm/yourprovider/`
2. Implement `llm.Client` interface
3. Handle message conversion and tool call parsing
4. Update `cmd/finta/main.go` to support selection

## Future Phases (See plans.md)

- **Phase 2**: Advanced tool system (parallel execution, more built-in tools)
- **Phase 3**: Specialized agents (Explore, Plan, Execute)
- **Phase 4**: MCP integration
- **Phase 5**: Hook/plugin system
- **Phase 6**: Session management and persistence
- **Phase 7**: Configuration system
- **Phase 8**: Documentation and examples

## Testing Strategy

Currently no automated tests. When adding tests:
- Mock the `llm.Client` interface for agent tests
- Use real tools but mock external dependencies (filesystem, network)
- Test error handling paths (tool not found, execution failures)
- Verify logger output by providing a custom `io.Writer`

## Important Gotchas

1. **Nil Logger Panic**: Always pass a Logger in Input, or the agent will panic when trying to log. The CLI handles this, but any direct BaseAgent usage must provide one.

2. **Tool Name Conflicts**: Registry.Register() returns an error if a tool with the same name is already registered. Names must be unique.

3. **Context Cancellation**: Tool execution respects context cancellation, but the agent loop itself doesn't check context between LLM calls. Long-running agents may not respond to cancellation immediately.

4. **Import Path Changes**: The project was recently reorganized from `pkg/` to `internal/`. Some documentation may still reference old paths.

5. **StopReason Handling**: The agent only continues looping on `StopReasonToolCalls`. Any other stop reason (including unknown ones) will end the loop. This is intentional but can be surprising if the LLM returns unexpected stop reasons.
