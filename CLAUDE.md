# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Finta is an AI Agent framework inspired by ClaudeCode's design philosophy. It provides a modular, extensible foundation for building AI agents that can execute tools, interact with LLMs, and handle complex multi-turn conversations.

**Current Status**: Phase 4 complete (MCP integration) - specialized agents + reasoning + extensible tool system
**Go Version**: 1.24.5
**Primary LLM Integration**: OpenAI API (with Extended Thinking / Reasoning)
**Tool System**: Built-in tools + MCP (Model Context Protocol) server integration

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

# Use specialized agent
./finta chat --agent-type explore "Find all Go files in internal/"
./finta chat --agent-type plan "Plan how to add a new tool"
./finta chat --agent-type execute "Create test file"

# With MCP servers (requires config file)
./finta chat --config configs/finta.yaml "List files in project directory"
```

### Available CLI Flags
- `--api-key` - OpenAI API key (or env: OPENAI_API_KEY)
- `--api-base-url` - Custom API endpoint (or env: OPENAI_API_BASE_URL)
- `--model` - Model to use (default: gpt-4-turbo)
- `--agent-type` - Agent type (general, explore, plan, execute; default: general)
- `--temperature` - Temperature parameter (default: 0.7)
- `--max-turns` - Max conversation turns (default: 10)
- `--verbose` - Enable debug logging
- `--no-color` - Disable colored output
- `--streaming` - Enable streaming output (default: false)
- `--parallel` - Enable parallel tool execution (default: true)
- `--config` - Path to config file with MCP servers (default: auto-detect from ./finta.yaml, ./configs/finta.yaml, ~/.config/finta/finta.yaml, or /etc/finta/finta.yaml)

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

### Specialized Agents & Factory Pattern (Phase 3)

**NEW**: The framework supports specialized agent types with different capabilities:

```
Factory Pattern:
  DefaultFactory.CreateAgent(agentType) → {Explore, Plan, Execute, General}Agent

Agent Types:
  • General  - All tools, temp=0.7, turns=20 (default)
  • Explore  - Read-only tools, temp=0.3, turns=15
  • Plan     - Read+glob only, temp=0.5, turns=10
  • Execute  - All tools, temp=0.5, turns=20

Task Tool:
  Parent Agent → Task("explore", "task") → Explore Sub-Agent
                                          ↓
                                    Shared Logger
                                    Depth tracking (max 3 levels)
                                    Result formatting
```

**Key Features**:
- **Factory Pattern**: `DefaultFactory` in `internal/agent/types.go` creates specialized agents
- **Task Tool**: Enables hierarchical agent composition (parent spawns sub-agents)
- **Nesting Control**: Maximum 3-level depth prevents infinite recursion
- **Logger Propagation**: Sub-agents share parent's logger via context
- **Tool Filtering**: Each agent type has access to specific tool subsets

**Usage**:
```bash
# Direct usage
./finta chat --agent-type explore "Find all Go files"
./finta chat --agent-type plan "Plan how to add X feature"

# From within agent (Task tool)
task(agent_type="explore", task="Explore codebase", description="Code exploration")
```

See `PHASE3_SUMMARY.md` for detailed documentation.

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
- **Best Practices** (Optional): Tools can implement `ToolWithBestPractices` to provide usage guidelines

Tool execution flow in `base.go`:
1. LLM returns tool calls with function name + JSON arguments
2. Agent looks up tool in registry
3. Tool executes with context + parameters
4. Result (with timing) is logged and added to conversation

#### Tool Best Practices System

**NEW**: Tools can optionally provide best practices that are automatically included in agent system prompts.

**How it works**:
1. Tools implement `ToolWithBestPractices` interface with a `BestPractices()` method
2. When creating agents, the factory calls `registry.GetToolBestPractices()`
3. Best practices are appended to the agent's system prompt
4. LLM learns optimal tool usage patterns

**Benefits**:
- Self-contained: Tools define their own usage guidelines
- Automatic: Best practices update whenever tools update
- Configurable: Can be enabled/disabled per factory with `SetIncludeBestPractices()`
- Educational: Improves LLM tool usage without prompt engineering

**Example**:
```go
func (t *ReadTool) BestPractices() string {
    return `**Read Tool Best Practices**:
1. Use line ranges for large files
2. Read multiple related files together
3. Read files before modifying them`
}
```

See `docs/TOOL_BEST_PRACTICES.md` for complete documentation.

### Built-in Tools

The framework includes several built-in tools:

- **read** - Read one or more files (max 8) with optional line range support
- **bash** - Execute bash commands with timeout support (handles empty output safely)
- **write** - Write/create files with automatic directory creation
- **glob** - Find files matching glob patterns (supports `**` recursive matching)
- **grep** - Search file contents with regex support
- **task** - Spawn sub-agents for hierarchical task delegation
- **TodoWrite** - Task progress tracking and management

#### Glob Tool (Enhanced)

**NEW**: The glob tool now supports `**` recursive directory matching.

**Supported Patterns**:
- `*.go` - All Go files in current directory (standard glob)
- `**/*.go` - All Go files recursively in all subdirectories
- `internal/**/*.go` - All Go files under internal/ recursively
- `src/**/test/*.ts` - Test files in any subdirectory of src/
- `**` - All files recursively

**Examples**:
```json
// Find all Go files recursively
{"pattern": "**/*.go"}

// Find all test files under src/
{"pattern": "src/**/*_test.go"}

// Find all TypeScript files in internal/
{"pattern": "internal/**/*.ts", "path": "."}
```

**Implementation**: Uses `filepath.WalkDir` for recursive patterns, standard `filepath.Glob` for non-recursive patterns. No external dependencies.

#### Bash Tool (Enhanced)

**FIXED**: The bash tool now handles empty command output safely.

**Issue**: When bash commands produced no output (e.g., `true`, `mkdir`, successful `find` with no matches), the tool returned an empty string. This caused LLM API errors with providers that require non-empty message content.

**Solution**: Returns placeholder message when output is empty:
```
(Command executed successfully with no output)
```

This ensures the LLM always receives valid, non-empty content while maintaining clear feedback about command success.

#### Read Tool (Enhanced)

The Read tool supports advanced file reading capabilities:

**Features**:
- Read up to 8 files in a single call
- Optional line range specification (from-to)
- Automatic formatting with file headers for multi-file reads
- Line counting and metadata

**Single File Examples**:
```json
// Read entire file
{"files": [{"file_path": "config.yaml"}]}

// Read lines 10-20
{"files": [{"file_path": "main.go", "from": 10, "to": 20}]}

// Read from line 50 to end
{"files": [{"file_path": "data.txt", "from": 50}]}
```

**Multi-File Example**:
```json
{
  "files": [
    {"file_path": "src/main.go", "from": 1, "to": 50},
    {"file_path": "src/utils.go"},
    {"file_path": "README.md", "from": 1, "to": 20}
  ]
}
```

**Output Format** (multi-file):
```
=== File 1/3: src/main.go (lines 1-50, returned 50 lines) ===
[file content here]

=== File 2/3: src/utils.go ===
[file content here]

=== File 3/3: README.md (lines 1-20, returned 20 lines) ===
[file content here]
```

**Limits**:
- Maximum 8 files per call
- Line numbers are 1-based (first line = 1)
- `from` must be ≤ `to` when both are specified

#### TodoWrite Tool

The TodoWrite tool provides task tracking functionality inspired by Claude Agent SDK:

**Purpose**: Track progress on complex multi-step tasks (3+ steps)

**Usage Patterns**:
- Create todo list when starting complex work
- Update status as tasks progress: pending → in_progress → completed
- Keep exactly ONE task in_progress at a time
- Mark tasks completed IMMEDIATELY after finishing
- Clear list (empty array) when all tasks done

**Todo Item Structure**:
```json
{
  "content": "Fix bug in auth",      // Imperative form
  "status": "in_progress",           // pending | in_progress | completed
  "activeForm": "Fixing bug in auth" // Present continuous form
}
```

**Example Usage**:
```json
{
  "todos": [
    {"content": "Read config file", "status": "completed", "activeForm": "Reading config file"},
    {"content": "Parse data", "status": "in_progress", "activeForm": "Parsing data"},
    {"content": "Write output", "status": "pending", "activeForm": "Writing output"}
  ]
}
```

**Output Format**:
```
📋 Todo List: 1/3 completed
──────────────────────────────────────────────────
1. ✅ Read config file
2. 🔧 Parsing data
3. ⏳ Write output
```

**Validation Rules**:
- Only ONE task can be in_progress at a time
- Content and activeForm cannot be empty
- Status must be valid enum value
- Pass empty array `[]` to clear all todos

**Global State**: TodoWrite maintains a global todo list accessible via `builtin.GetCurrentTodos()`

### MCP Integration (Phase 4)

**NEW**: The framework now supports MCP (Model Context Protocol) servers, enabling dynamic tool extension via external processes.

#### Overview

MCP integration allows Finta to load tools from external MCP servers at runtime. This enables:
- **Dynamic tool loading**: Add new capabilities without modifying Finta code
- **Tool namespacing**: MCP tools use `server_name_tool_name` format (e.g., `filesystem_read_file`)
- **Multiple servers**: Run multiple MCP servers concurrently
- **Seamless integration**: MCP tools work alongside built-in tools

**Note**: Tool names must match OpenAI's pattern `^[a-zA-Z0-9_-]+$` (alphanumeric, underscores, and hyphens only).

#### Architecture

```
Config File (YAML)
       ↓
MCP Manager → Server 1 (filesystem) → Tools: filesystem_read_file, filesystem_write_file
            → Server 2 (github) → Tools: github_create_issue, github_search_repos
       ↓
Tool Registry (Built-in + MCP tools)
       ↓
Agent Execution (LLM can call any tool)
```

#### Configuration

Create a config file (e.g., `configs/finta.yaml`) with MCP server definitions:

```yaml
mcp:
  servers:
    # Filesystem access
    - name: filesystem
      transport: stdio
      command: npx
      args:
        - "-y"
        - "@modelcontextprotocol/server-filesystem"
        - "/home/user/projects"  # Allowed directory

    # GitHub integration
    - name: github
      transport: stdio
      command: npx
      args:
        - "-y"
        - "@modelcontextprotocol/server-github"
      env:
        GITHUB_TOKEN: ${GITHUB_TOKEN}  # Environment variable expansion

    # Disabled server (will be skipped)
    - name: experimental
      transport: stdio
      command: ./my-server
      disabled: true
```

**Config File Locations** (checked in order):
1. `./finta.yaml` (project directory)
2. `./configs/finta.yaml` (project directory)
3. `~/.config/finta/finta.yaml` (user config)
4. `/etc/finta/finta.yaml` (system-wide)

Or specify explicitly with `--config /path/to/config.yaml`

#### Environment Variable Interpolation

Config values support `${VAR}` and `$VAR` syntax:
- `${GITHUB_TOKEN}` - Reads from `GITHUB_TOKEN` environment variable
- `Bearer ${API_KEY}` - Interpolated to `Bearer abc123...`

#### Tool Namespacing

MCP tools are namespaced to prevent conflicts:
- Built-in tool: `read`
- MCP tool: `filesystem_read_file`, `github_create_issue`

**Naming Convention**:
- Format: `{server_name}_{tool_name}`
- Server names and tool names are automatically combined with underscore
- Must match pattern: `^[a-zA-Z0-9_-]+$` (OpenAI API requirement)
- Examples: `filesystem_read_file`, `github_create_issue`, `slack_send_message`

This allows both built-in and MCP tools with similar names to coexist.

#### Component Architecture

**Files**:
- `internal/config/` - YAML config parsing and env variable expansion
- `internal/mcp/client.go` - MCP SDK client wrapper
- `internal/mcp/server.go` - Server instance management
- `internal/mcp/adapter.go` - MCP Tool → Finta Tool adapter
- `internal/mcp/manager.go` - Multi-server coordinator
- `configs/finta.yaml` - Example configuration file

**Key Classes**:
```go
// Manager coordinates multiple MCP servers
type Manager struct {
    servers  map[string]*Server
    registry *tool.Registry
}

// Server wraps an MCP server process
type Server struct {
    config MCPServerConfig
    client *Client  // SDK client + session
}

// MCPToolAdapter implements tool.Tool interface
type MCPToolAdapter struct {
    client         *Client
    mcpTool        *mcp.Tool
    namespacedName string  // "server_tool" format
}
```

#### Usage Example

1. **Install MCP servers**:
```bash
npm install -g @modelcontextprotocol/server-filesystem
npm install -g @modelcontextprotocol/server-github
```

2. **Create config file** (`configs/finta.yaml` - see example above)

3. **Set environment variables**:
```bash
export GITHUB_TOKEN=ghp_your_token_here
```

4. **Run Finta with MCP**:
```bash
./finta chat --config configs/finta.yaml "Create a GitHub issue for bug X"
# Agent can now use github_create_issue tool
```

#### Error Handling

- **Partial Success**: If some MCP servers fail to start, Finta continues with available tools
- **Graceful Degradation**: Missing config file is not an error - Finta runs with built-in tools only
- **Clear Logging**: Failed servers are logged with specific error messages

#### Limitations

- **Transport**: Only `stdio` transport is currently supported (HTTP/SSE planned for future)
- **Resource Support**: MCP resources and prompts are not yet implemented (tools only)
- **Health Monitoring**: Basic health checks only - no auto-restart on server crashes

#### Future Enhancements

- HTTP/SSE transport support
- MCP resource and prompt support
- Health monitoring with auto-restart
- Per-agent MCP tool filtering
- Hot-reload of MCP servers

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

## Implementation Status (See plans.md and docs/phase/)

### Completed Phases

- **Phase 1**: ✅ Core agent framework with ReAct pattern
- **Phase 2**: ✅ Advanced tool system (parallel/mixed execution, glob, grep, read, write, bash, TodoWrite)
- **Phase 3**: ✅ Specialized agents (General, Explore, Plan, Execute) with Task tool for hierarchical composition
- **Phase 4**: ✅ MCP integration (stdio transport, tool namespacing, YAML config, environment variable expansion)

### Future Phases

- **Phase 5**: Hook/plugin system
- **Phase 6**: Session management and persistence
- **Phase 7**: Enhanced configuration system (per-agent MCP filtering, tool access control)
- **Phase 8**: Comprehensive testing and examples

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
