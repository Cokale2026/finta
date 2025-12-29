# Finta

[![Go Version](https://img.shields.io/badge/Go-1.24.5-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

English | [中文](README_zh.md)

A flexible AI Agent framework inspired by Claude Code's design philosophy. Finta provides a modular, extensible foundation for building AI agents that can execute tools, interact with LLMs, and handle complex multi-turn conversations.

## Features

- **Interface-Driven Architecture** - Clean separation of concerns with pluggable components
- **Specialized Agents** - Four agent types optimized for different tasks (General, Explore, Plan, Execute)
- **Hierarchical Agent Composition** - Spawn sub-agents for complex task delegation
- **Built-in Tools** - Read, Write, Bash, Glob, Grep, TodoWrite, and Task tools
- **MCP Integration** - Extend capabilities with Model Context Protocol servers
- **Hook System** - User confirmation before executing potentially dangerous operations
- **Parallel Tool Execution** - Smart dependency analysis for concurrent tool calls
- **Streaming Output** - Real-time response streaming with markdown rendering
- **Reasoning Support** - Extended thinking/reasoning process visualization

## Installation

```bash
# Clone the repository
git clone https://github.com/cokale/finta.git
cd finta

# Build the binary
go build -o finta cmd/finta/main.go
```

## Quick Start

```bash
# Set your OpenAI API key
export OPENAI_API_KEY="your-api-key"

# Start interactive chat
./finta chat

# With custom model
./finta chat --model gpt-4o

# Use specialized agent
./finta chat --agent-type explore
```

## Usage

### CLI Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--api-key` | OpenAI API key | `$OPENAI_API_KEY` |
| `--api-base-url` | Custom API endpoint | `$OPENAI_API_BASE_URL` |
| `--model` | Model to use | `gpt-4-turbo` |
| `--agent-type` | Agent type (general, explore, plan, execute) | `general` |
| `--temperature` | Temperature parameter | `0.7` |
| `--max-turns` | Max conversation turns | `10` |
| `--verbose` | Enable debug logging | `false` |
| `--streaming` | Enable streaming output | `false` |
| `--parallel` | Enable parallel tool execution | `true` |
| `--config` | Path to config file | auto-detect |

### Agent Types

| Type | Description | Tools | Temperature |
|------|-------------|-------|-------------|
| **General** | All-purpose agent | All tools | 0.7 |
| **Explore** | Code exploration and search | Read-only tools | 0.3 |
| **Plan** | Implementation planning | Read + Glob | 0.5 |
| **Execute** | Task execution | All tools | 0.5 |

```bash
# Explore codebase
./finta chat --agent-type explore
> Find all Go files that handle HTTP requests

# Plan implementation
./finta chat --agent-type plan
> Plan how to add user authentication
```

## Built-in Tools

| Tool | Description |
|------|-------------|
| `read` | Read files with optional line ranges (up to 8 files) |
| `write` | Create or overwrite files |
| `bash` | Execute shell commands with timeout |
| `glob` | Find files matching patterns (supports `**` recursion) |
| `grep` | Search file contents with regex |
| `task` | Spawn sub-agents for task delegation |
| `TodoWrite` | Track progress on multi-step tasks |

## Configuration

Finta looks for configuration files in these locations (in order):
1. `./finta.yaml`
2. `./configs/finta.yaml`
3. `~/.config/finta/finta.yaml`
4. `/etc/finta/finta.yaml`

### Example Configuration

```yaml
# MCP Server Configuration
mcp:
  servers:
    - name: filesystem
      transport: stdio
      command: npx
      args:
        - "-y"
        - "@modelcontextprotocol/server-filesystem"
        - "/allowed/path"

    - name: github
      transport: stdio
      command: npx
      args:
        - "-y"
        - "@modelcontextprotocol/server-github"
      env:
        GITHUB_TOKEN: ${GITHUB_TOKEN}

# Hook Configuration
hooks:
  bash_confirm: true
  tool_confirm:
    - write
    - bash
```

## MCP Integration

Finta supports [Model Context Protocol](https://modelcontextprotocol.io/) servers for extending tool capabilities.

```bash
# Install MCP servers
npm install -g @modelcontextprotocol/server-filesystem
npm install -g @modelcontextprotocol/server-github

# Set environment variables
export GITHUB_TOKEN=your_token

# Run with MCP
./finta chat --config configs/finta.yaml
```

MCP tools are namespaced as `{server}_{tool}` (e.g., `filesystem_read_file`, `github_create_issue`).

## Hook System

Hooks allow user confirmation before executing potentially dangerous operations:

- **bash_confirm** - Confirm before executing shell commands
- **tool_confirm** - Confirm before specific tool executions

When a hook is triggered, you'll be prompted to allow or deny the operation.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         CLI (main.go)                        │
├─────────────────────────────────────────────────────────────┤
│                      Agent Factory                           │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐           │
│  │ General │ │ Explore │ │  Plan   │ │ Execute │           │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘           │
├─────────────────────────────────────────────────────────────┤
│                      Base Agent                              │
│         Run Loop: LLM Call → Tool Execution → Repeat        │
├──────────────────────┬──────────────────────────────────────┤
│     Tool Registry    │           LLM Client                  │
│  ┌────────────────┐  │  ┌────────────────────────────────┐  │
│  │  Built-in      │  │  │  OpenAI API                    │  │
│  │  + MCP Tools   │  │  │  (with Reasoning Support)      │  │
│  └────────────────┘  │  └────────────────────────────────┘  │
├──────────────────────┴──────────────────────────────────────┤
│                      Hook Manager                            │
│              User Confirmation & Feedback                    │
└─────────────────────────────────────────────────────────────┘
```

## Project Structure

```
finta/
├── cmd/finta/          # CLI entry point
├── internal/
│   ├── agent/          # Agent implementations and factory
│   ├── config/         # Configuration parsing
│   ├── hook/           # Hook system
│   ├── llm/            # LLM client interface and OpenAI implementation
│   ├── logger/         # Structured logging with markdown rendering
│   ├── mcp/            # MCP integration
│   └── tool/           # Tool interface, registry, and built-in tools
├── configs/            # Example configuration files
└── docs/               # Documentation
```

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
