# Finta AI Agent 框架实现计划

## 项目概述

**目标**: 构建一个遵循 ClaudeCode 设计理念的通用 AI Agent 开发框架

**核心特性**:

- 可扩展的工具系统（支持并行/顺序执行）
- 专门化 Agent（Explore、Plan、Execute 等）
- MCP (Model Context Protocol) 集成
- Hook/Plugin 系统
- 基于 OpenAI API 的 LLM 集成
- CLI 交互界面

**技术栈**: Go 1.24.5, OpenAI API

---

## 整体架构

### 核心组件层次

```
┌─────────────────────────────────────────────┐
│            CLI Interface Layer              │
│  (命令行交互、流式输出、Markdown 渲染)        │
└─────────────────┬───────────────────────────┘
                  │
┌─────────────────▼───────────────────────────┐
│           Agent Orchestration Layer         │
│  (Agent 生命周期、子 Agent 管理、上下文)      │
└─────────────────┬───────────────────────────┘
                  │
        ┌─────────┼─────────┐
        │         │         │
┌───────▼──┐ ┌───▼────┐ ┌──▼─────┐
│  Tool    │ │  LLM   │ │  Hook  │
│  System  │ │ Client │ │ System │
└──────────┘ └────────┘ └────────┘
        │         │         │
        └─────────┼─────────┘
                  │
        ┌─────────▼──────────┐
        │   MCP Integration  │
        │  Session Manager   │
        │  Config System     │
        └────────────────────┘
```

### 数据流

```
用户输入 → CLI → Agent → LLM Client → OpenAI API
                    ↓
              Tool Registry → Tool Execution (并行/顺序)
                    ↓
              Hook System → 生命周期事件
                    ↓
              Session Manager → 持久化
```

---

## 项目目录结构

```
finta/
├── cmd/finta/main.go              # CLI 入口
├── internal/
│   ├── agent/                     # Agent 核心
│   │   ├── agent.go               # Agent 接口和基础实现
│   │   ├── context.go             # Agent 上下文管理
│   │   ├── executor.go            # 工具执行引擎
│   │   ├── runner.go              # Agent 运行循环
│   │   └── specialized/           # 专门化 Agent
│   │       ├── explore.go
│   │       ├── plan.go
│   │       └── general.go
│   ├── llm/                       # LLM 客户端
│   │   ├── client.go              # LLM 接口
│   │   ├── message.go             # 消息类型
│   │   └── openai/                # OpenAI 实现
│   │       ├── client.go
│   │       ├── streaming.go
│   │       └── tool_calling.go
│   ├── tool/                      # 工具系统
│   │   ├── tool.go                # Tool 接口
│   │   ├── registry.go            # 工具注册表
│   │   ├── executor.go            # 并行执行器
│   │   └── builtin/               # 内置工具
│   │       ├── bash.go
│   │       ├── read.go
│   │       ├── write.go
│   │       ├── edit.go
│   │       ├── glob.go
│   │       └── grep.go
│   ├── mcp/                       # MCP 集成
│   │   ├── client.go
│   │   ├── server.go
│   │   ├── transport/
│   │   │   ├── stdio.go
│   │   │   └── http.go
│   │   └── adapter.go
│   ├── hook/                      # Hook 系统
│   │   ├── hook.go
│   │   ├── registry.go
│   │   └── executor.go
│   ├── session/                   # Session 管理
│   │   ├── session.go
│   │   ├── persistence.go
│   │   └── summarizer.go
│   ├── config/                    # 配置系统
│   │   ├── config.go
│   │   └── loader.go
│   └── cli/                       # CLI 组件
│       ├── app.go
│       ├── interactive.go
│       ├── streaming.go
│       └── markdown.go
├── configs/
│   └── default.yaml
├── examples/
│   ├── simple_agent/
│   └── custom_tool/
└── docs/
    ├── architecture.md
    └── development.md
```

---

## Phase 1: 核心基础 (2-3 天)

### 目标

构建最小可用的 Agent 框架，能够通过 CLI 运行一个简单的 agent，调用基础工具，与 OpenAI API 交互。

### 实现步骤

#### 1.1 项目初始化

**文件**: `go.mod` (已存在，需更新)

```bash
# 添加依赖
go get github.com/sashabaranov/go-openai
go get gopkg.in/yaml.v3
go get github.com/spf13/cobra
go get github.com/charmbracelet/glamour
```

**更新后的 go.mod**:

```go
module finta

go 1.24.5

require (
    github.com/sashabaranov/go-openai v1.35.6
    github.com/spf13/cobra v1.8.1
    github.com/charmbracelet/glamour v0.8.0
    gopkg.in/yaml.v3 v3.0.1
)
```

#### 1.2 核心接口定义

**文件**: `pkg/llm/message.go`

定义基础消息类型：

```go
package llm

import "time"

type Role string

const (
    RoleSystem    Role = "system"
    RoleUser      Role = "user"
    RoleAssistant Role = "assistant"
    RoleTool      Role = "tool"
)

type Message struct {
    Role       Role
    Content    string
    ToolCalls  []*ToolCall
    ToolCallID string
    Name       string
    Timestamp  time.Time
}

type ToolCall struct {
    ID       string
    Type     string
    Function *FunctionCall
}

type FunctionCall struct {
    Name      string
    Arguments string
}

type StopReason string

const (
    StopReasonStop      StopReason = "stop"
    StopReasonLength    StopReason = "length"
    StopReasonToolCalls StopReason = "tool_calls"
)

type Usage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}
```

**文件**: `pkg/llm/client.go`

```go
package llm

import "context"

type Client interface {
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    ChatStream(ctx context.Context, req *ChatRequest) (StreamReader, error)
    Provider() string
    Model() string
}

type ChatRequest struct {
    Messages    []Message
    Tools       []*ToolDefinition
    Temperature float32
    MaxTokens   int
}

type ChatResponse struct {
    Message    Message
    StopReason StopReason
    Usage      Usage
}

type ToolDefinition struct {
    Type     string
    Function *FunctionDef
}

type FunctionDef struct {
    Name        string
    Description string
    Parameters  map[string]any
}

type StreamReader interface {
    Recv() (*Delta, error)
    Close() error
}

type Delta struct {
    Role      Role
    Content   string
    ToolCalls []*ToolCall
    Done      bool
}
```

**文件**: `pkg/tool/tool.go`

```go
package tool

import (
    "context"
    "encoding/json"
    "time"
)

type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]any
    Execute(ctx context.Context, params json.RawMessage) (*Result, error)
}

type Result struct {
    Success bool
    Output  string
    Error   string
    Data    map[string]any
}

type CallResult struct {
    ToolName  string
    CallID    string
    Params    json.RawMessage
    Result    *Result
    StartTime time.Time
    EndTime   time.Time
}
```

**文件**: `pkg/agent/agent.go`

```go
package agent

import (
    "context"
    "finta/internal/llm"
    "finta/internal/tool"
)

type Agent interface {
    Name() string
    Run(ctx context.Context, input *Input) (*Output, error)
}

type Input struct {
    Messages    []llm.Message
    Task        string
    MaxTurns    int
    Temperature float32
}

type Output struct {
    Messages  []llm.Message
    Result    string
    ToolCalls []*tool.CallResult
}

type Config struct {
    Model       string
    Temperature float32
    MaxTokens   int
    MaxTurns    int
}
```

#### 1.3 OpenAI Client 实现

**文件**: `pkg/llm/openai/client.go`

```go
package openai

import (
    "context"
    "finta/internal/llm"

    openai "github.com/sashabaranov/go-openai"
)

type Client struct {
    client *openai.Client
    model  string
}

func NewClient(apiKey, model string) *Client {
    return &Client{
        client: openai.NewClient(apiKey),
        model:  model,
    }
}

func (c *Client) Chat(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
    // 转换消息格式
    messages := c.convertMessages(req.Messages)

    // 转换工具定义
    tools := c.convertTools(req.Tools)

    // 调用 OpenAI API
    resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model:       c.model,
        Messages:    messages,
        Tools:       tools,
        Temperature: req.Temperature,
        MaxTokens:   req.MaxTokens,
    })
    if err != nil {
        return nil, err
    }

    // 转换响应
    return c.convertResponse(resp), nil
}

func (c *Client) Provider() string {
    return "openai"
}

func (c *Client) Model() string {
    return c.model
}

// 辅助方法：消息格式转换
func (c *Client) convertMessages(msgs []llm.Message) []openai.ChatCompletionMessage {
    result := make([]openai.ChatCompletionMessage, len(msgs))
    for i, msg := range msgs {
        ocMsg := openai.ChatCompletionMessage{
            Role:    string(msg.Role),
            Content: msg.Content,
        }

        // 转换 tool calls
        if len(msg.ToolCalls) > 0 {
            ocMsg.ToolCalls = make([]openai.ToolCall, len(msg.ToolCalls))
            for j, tc := range msg.ToolCalls {
                ocMsg.ToolCalls[j] = openai.ToolCall{
                    ID:   tc.ID,
                    Type: openai.ToolTypeFunction,
                    Function: openai.FunctionCall{
                        Name:      tc.Function.Name,
                        Arguments: tc.Function.Arguments,
                    },
                }
            }
        }

        // Tool 响应消息
        if msg.Role == llm.RoleTool {
            ocMsg.ToolCallID = msg.ToolCallID
        }

        result[i] = ocMsg
    }
    return result
}

// 辅助方法：工具定义转换
func (c *Client) convertTools(tools []*llm.ToolDefinition) []openai.Tool {
    result := make([]openai.Tool, len(tools))
    for i, t := range tools {
        result[i] = openai.Tool{
            Type: openai.ToolTypeFunction,
            Function: &openai.FunctionDefinition{
                Name:        t.Function.Name,
                Description: t.Function.Description,
                Parameters:  t.Function.Parameters,
            },
        }
    }
    return result
}

// 辅助方法：响应转换
func (c *Client) convertResponse(resp openai.ChatCompletionResponse) *llm.ChatResponse {
    choice := resp.Choices[0]
    msg := choice.Message

    result := &llm.ChatResponse{
        Message: llm.Message{
            Role:    llm.Role(msg.Role),
            Content: msg.Content,
        },
        Usage: llm.Usage{
            PromptTokens:     resp.Usage.PromptTokens,
            CompletionTokens: resp.Usage.CompletionTokens,
            TotalTokens:      resp.Usage.TotalTokens,
        },
    }

    // 转换 tool calls
    if len(msg.ToolCalls) > 0 {
        result.Message.ToolCalls = make([]*llm.ToolCall, len(msg.ToolCalls))
        for i, tc := range msg.ToolCalls {
            result.Message.ToolCalls[i] = &llm.ToolCall{
                ID:   tc.ID,
                Type: string(tc.Type),
                Function: &llm.FunctionCall{
                    Name:      tc.Function.Name,
                    Arguments: tc.Function.Arguments,
                },
            }
        }
        result.StopReason = llm.StopReasonToolCalls
    } else {
        result.StopReason = llm.StopReason(choice.FinishReason)
    }

    return result
}
```

#### 1.4 工具系统基础

**文件**: `pkg/tool/registry.go`

```go
package tool

import (
    "fmt"
    "sync"
)

type Registry struct {
    tools map[string]Tool
    mu    sync.RWMutex
}

func NewRegistry() *Registry {
    return &Registry{
        tools: make(map[string]Tool),
    }
}

func (r *Registry) Register(tool Tool) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    name := tool.Name()
    if _, exists := r.tools[name]; exists {
        return fmt.Errorf("tool %s already registered", name)
    }

    r.tools[name] = tool
    return nil
}

func (r *Registry) Get(name string) (Tool, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    tool, exists := r.tools[name]
    if !exists {
        return nil, fmt.Errorf("tool %s not found", name)
    }

    return tool, nil
}

func (r *Registry) List() []Tool {
    r.mu.RLock()
    defer r.mu.RUnlock()

    tools := make([]Tool, 0, len(r.tools))
    for _, t := range r.tools {
        tools = append(tools, t)
    }
    return tools
}

func (r *Registry) GetToolDefinitions() []*llm.ToolDefinition {
    tools := r.List()
    defs := make([]*llm.ToolDefinition, len(tools))

    for i, t := range tools {
        defs[i] = &llm.ToolDefinition{
            Type: "function",
            Function: &llm.FunctionDef{
                Name:        t.Name(),
                Description: t.Description(),
                Parameters:  t.Parameters(),
            },
        }
    }

    return defs
}
```

#### 1.5 基础工具实现

**文件**: `pkg/tool/builtin/read.go`

```go
package builtin

import (
    "context"
    "encoding/json"
    "fmt"
    "os"

    "finta/internal/tool"
)

type ReadTool struct{}

func NewReadTool() *ReadTool {
    return &ReadTool{}
}

func (t *ReadTool) Name() string {
    return "read"
}

func (t *ReadTool) Description() string {
    return "Read contents of a file"
}

func (t *ReadTool) Parameters() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "file_path": map[string]any{
                "type":        "string",
                "description": "Path to the file to read",
            },
        },
        "required": []string{"file_path"},
    }
}

func (t *ReadTool) Execute(ctx context.Context, params json.RawMessage) (*tool.Result, error) {
    var p struct {
        FilePath string `json:"file_path"`
    }

    if err := json.Unmarshal(params, &p); err != nil {
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("invalid parameters: %v", err),
        }, nil
    }

    content, err := os.ReadFile(p.FilePath)
    if err != nil {
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("failed to read file: %v", err),
        }, nil
    }

    return &tool.Result{
        Success: true,
        Output:  string(content),
    }, nil
}
```

**文件**: `pkg/tool/builtin/bash.go`

```go
package builtin

import (
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
    "time"

    "finta/internal/tool"
)

type BashTool struct{}

func NewBashTool() *BashTool {
    return &BashTool{}
}

func (t *BashTool) Name() string {
    return "bash"
}

func (t *BashTool) Description() string {
    return "Execute a bash command"
}

func (t *BashTool) Parameters() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "command": map[string]any{
                "type":        "string",
                "description": "The bash command to execute",
            },
            "timeout": map[string]any{
                "type":        "number",
                "description": "Timeout in milliseconds (default: 120000)",
            },
        },
        "required": []string{"command"},
    }
}

func (t *BashTool) Execute(ctx context.Context, params json.RawMessage) (*tool.Result, error) {
    var p struct {
        Command string `json:"command"`
        Timeout int    `json:"timeout"`
    }

    if err := json.Unmarshal(params, &p); err != nil {
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("invalid parameters: %v", err),
        }, nil
    }

    // 默认超时 2 分钟
    timeout := 120000
    if p.Timeout > 0 {
        timeout = p.Timeout
    }

    ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
    defer cancel()

    cmd := exec.CommandContext(ctx, "bash", "-c", p.Command)
    output, err := cmd.CombinedOutput()

    if err != nil {
        return &tool.Result{
            Success: false,
            Output:  string(output),
            Error:   err.Error(),
        }, nil
    }

    return &tool.Result{
        Success: true,
        Output:  string(output),
    }, nil
}
```

#### 1.6 Agent 基础实现

**文件**: `pkg/agent/base.go`

```go
package agent

import (
    "context"
    "fmt"
    "time"

    "finta/internal/llm"
    "finta/internal/tool"
)

type BaseAgent struct {
    name         string
    systemPrompt string
    llmClient    llm.Client
    toolRegistry *tool.Registry
    config       *Config
}

func NewBaseAgent(name, systemPrompt string, client llm.Client, registry *tool.Registry, cfg *Config) *BaseAgent {
    if cfg == nil {
        cfg = &Config{
            Model:       "gpt-4-turbo",
            Temperature: 0.7,
            MaxTokens:   4096,
            MaxTurns:    20,
        }
    }

    return &BaseAgent{
        name:         name,
        systemPrompt: systemPrompt,
        llmClient:    client,
        toolRegistry: registry,
        config:       cfg,
    }
}

func (a *BaseAgent) Name() string {
    return a.name
}

func (a *BaseAgent) Run(ctx context.Context, input *Input) (*Output, error) {
    // 初始化消息列表
    messages := make([]llm.Message, 0, len(input.Messages)+1)

    // 添加系统提示
    if a.systemPrompt != "" {
        messages = append(messages, llm.Message{
            Role:    llm.RoleSystem,
            Content: a.systemPrompt,
        })
    }

    // 添加历史消息
    messages = append(messages, input.Messages...)

    // 添加用户任务
    if input.Task != "" {
        messages = append(messages, llm.Message{
            Role:      llm.RoleUser,
            Content:   input.Task,
            Timestamp: time.Now(),
        })
    }

    maxTurns := input.MaxTurns
    if maxTurns == 0 {
        maxTurns = a.config.MaxTurns
    }

    allToolCalls := make([]*tool.CallResult, 0)

    // Agent 运行循环
    for turn := 0; turn < maxTurns; turn++ {
        // 调用 LLM
        resp, err := a.llmClient.Chat(ctx, &llm.ChatRequest{
            Messages:    messages,
            Tools:       a.toolRegistry.GetToolDefinitions(),
            Temperature: input.Temperature,
            MaxTokens:   a.config.MaxTokens,
        })
        if err != nil {
            return nil, fmt.Errorf("LLM call failed: %w", err)
        }

        // 添加助手消息
        messages = append(messages, resp.Message)

        // 检查是否完成
        if resp.StopReason == llm.StopReasonStop {
            return &Output{
                Messages:  messages,
                Result:    resp.Message.Content,
                ToolCalls: allToolCalls,
            }, nil
        }

        // 处理工具调用
        if resp.StopReason == llm.StopReasonToolCalls {
            toolResults, err := a.executeTools(ctx, resp.Message.ToolCalls)
            if err != nil {
                return nil, fmt.Errorf("tool execution failed: %w", err)
            }

            allToolCalls = append(allToolCalls, toolResults...)

            // 添加工具结果消息
            for _, tr := range toolResults {
                messages = append(messages, llm.Message{
                    Role:       llm.RoleTool,
                    ToolCallID: tr.CallID,
                    Content:    tr.Result.Output,
                    Name:       tr.ToolName,
                    Timestamp:  tr.EndTime,
                })
            }

            continue
        }

        // 如果因为长度限制停止
        if resp.StopReason == llm.StopReasonLength {
            return &Output{
                Messages:  messages,
                Result:    resp.Message.Content + "\n[Response truncated due to length limit]",
                ToolCalls: allToolCalls,
            }, nil
        }
    }

    return nil, fmt.Errorf("max turns (%d) exceeded", maxTurns)
}

func (a *BaseAgent) executeTools(ctx context.Context, toolCalls []*llm.ToolCall) ([]*tool.CallResult, error) {
    results := make([]*tool.CallResult, len(toolCalls))

    for i, tc := range toolCalls {
        startTime := time.Now()

        t, err := a.toolRegistry.Get(tc.Function.Name)
        if err != nil {
            results[i] = &tool.CallResult{
                ToolName:  tc.Function.Name,
                CallID:    tc.ID,
                Result:    &tool.Result{
                    Success: false,
                    Error:   fmt.Sprintf("tool not found: %v", err),
                },
                StartTime: startTime,
                EndTime:   time.Now(),
            }
            continue
        }

        result, err := t.Execute(ctx, []byte(tc.Function.Arguments))
        if err != nil {
            results[i] = &tool.CallResult{
                ToolName:  tc.Function.Name,
                CallID:    tc.ID,
                Result:    &tool.Result{
                    Success: false,
                    Error:   fmt.Sprintf("execution error: %v", err),
                },
                StartTime: startTime,
                EndTime:   time.Now(),
            }
            continue
        }

        results[i] = &tool.CallResult{
            ToolName:  tc.Function.Name,
            CallID:    tc.ID,
            Params:    []byte(tc.Function.Arguments),
            Result:    result,
            StartTime: startTime,
            EndTime:   time.Now(),
        }
    }

    return results, nil
}
```

#### 1.7 日志和输出展示系统

这是 Phase 1 中非常重要的一部分，让用户能够清楚看到 Agent 做了什么。

**文件**: `pkg/logger/logger.go`

```go
package logger

import (
    "fmt"
    "io"
    "os"
    "strings"
    "time"
)

type Level int

const (
    LevelDebug Level = iota
    LevelInfo
    LevelTool
    LevelAgent
    LevelError
)

type Logger struct {
    writer    io.Writer
    level     Level
    showTime  bool
    colorMode bool
}

func NewLogger(w io.Writer, level Level) *Logger {
    if w == nil {
        w = os.Stdout
    }
    return &Logger{
        writer:    w,
        level:     level,
        showTime:  true,
        colorMode: true,
    }
}

// ANSI 颜色代码
const (
    ColorReset   = "\033[0m"
    ColorRed     = "\033[31m"
    ColorGreen   = "\033[32m"
    ColorYellow  = "\033[33m"
    ColorBlue    = "\033[34m"
    ColorMagenta = "\033[35m"
    ColorCyan    = "\033[36m"
    ColorGray    = "\033[90m"
    ColorBold    = "\033[1m"
)

func (l *Logger) Debug(format string, args ...any) {
    if l.level <= LevelDebug {
        l.log(ColorGray, "DEBUG", format, args...)
    }
}

func (l *Logger) Info(format string, args ...any) {
    if l.level <= LevelInfo {
        l.log(ColorBlue, "INFO", format, args...)
    }
}

func (l *Logger) Error(format string, args ...any) {
    l.log(ColorRed, "ERROR", format, args...)
}

func (l *Logger) AgentThinking(content string) {
    if l.level <= LevelAgent {
        l.printSection(ColorMagenta, "🤔 Agent Thinking", content)
    }
}

func (l *Logger) AgentResponse(content string) {
    if l.level <= LevelAgent {
        l.printSection(ColorGreen, "💬 Agent Response", content)
    }
}

func (l *Logger) ToolCall(toolName string, params string) {
    if l.level <= LevelTool {
        l.printSection(ColorCyan, fmt.Sprintf("🔧 Tool Call: %s", toolName), params)
    }
}

func (l *Logger) ToolResult(toolName string, success bool, output string, duration time.Duration) {
    if l.level <= LevelTool {
        status := "✅ Success"
        color := ColorGreen
        if !success {
            status = "❌ Failed"
            color = ColorRed
        }

        header := fmt.Sprintf("📊 Tool Result: %s [%s] (%s)", toolName, status, duration)
        l.printSection(color, header, output)
    }
}

func (l *Logger) SessionStart(task string) {
    l.printBanner(ColorCyan, "🚀 Session Started", task)
}

func (l *Logger) SessionEnd(duration time.Duration, toolCallCount int) {
    summary := fmt.Sprintf("Duration: %s | Tool Calls: %d", duration, toolCallCount)
    l.printBanner(ColorGreen, "✨ Session Completed", summary)
}

func (l *Logger) log(color, level, format string, args ...any) {
    timestamp := ""
    if l.showTime {
        timestamp = time.Now().Format("15:04:05") + " "
    }

    msg := fmt.Sprintf(format, args...)

    if l.colorMode {
        fmt.Fprintf(l.writer, "%s%s[%s]%s %s\n",
            color, timestamp, level, ColorReset, msg)
    } else {
        fmt.Fprintf(l.writer, "%s[%s] %s\n", timestamp, level, msg)
    }
}

func (l *Logger) printSection(color, header, content string) {
    separator := strings.Repeat("─", 60)

    if l.colorMode {
        fmt.Fprintf(l.writer, "\n%s%s%s%s\n", ColorBold, color, header, ColorReset)
        fmt.Fprintf(l.writer, "%s%s%s\n", color, separator, ColorReset)
        fmt.Fprintf(l.writer, "%s\n", content)
        fmt.Fprintf(l.writer, "%s%s%s\n\n", color, separator, ColorReset)
    } else {
        fmt.Fprintf(l.writer, "\n%s\n%s\n%s\n%s\n\n", header, separator, content, separator)
    }
}

func (l *Logger) printBanner(color, title, subtitle string) {
    separator := strings.Repeat("═", 70)

    if l.colorMode {
        fmt.Fprintf(l.writer, "\n%s%s%s%s\n", ColorBold, color, separator, ColorReset)
        fmt.Fprintf(l.writer, "%s%s  %s%s\n", ColorBold, color, title, ColorReset)
        if subtitle != "" {
            fmt.Fprintf(l.writer, "%s  %s%s\n", color, subtitle, ColorReset)
        }
        fmt.Fprintf(l.writer, "%s%s%s%s\n\n", ColorBold, color, separator, ColorReset)
    } else {
        fmt.Fprintf(l.writer, "\n%s\n  %s\n  %s\n%s\n\n", separator, title, subtitle, separator)
    }
}

func (l *Logger) Progress(current, total int, message string) {
    if l.level <= LevelInfo {
        bar := l.progressBar(current, total, 30)
        fmt.Fprintf(l.writer, "\r%s[%d/%d] %s", bar, current, total, message)
        if current == total {
            fmt.Fprintln(l.writer)
        }
    }
}

func (l *Logger) progressBar(current, total, width int) string {
    if total == 0 {
        return ""
    }

    percent := float64(current) / float64(total)
    filled := int(percent * float64(width))

    bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

    if l.colorMode {
        return fmt.Sprintf("%s%s%s %.0f%%", ColorCyan, bar, ColorReset, percent*100)
    }
    return fmt.Sprintf("%s %.0f%%", bar, percent*100)
}
```

**文件**: `pkg/agent/context.go`

添加执行上下文，用于记录和展示执行过程：

```go
package agent

import (
    "time"
    "finta/internal/logger"
)

type ExecutionContext struct {
    Logger        *logger.Logger
    StartTime     time.Time
    CurrentTurn   int
    TotalTurns    int
    ToolCallCount int
}

func NewExecutionContext(log *logger.Logger) *ExecutionContext {
    return &ExecutionContext{
        Logger:    log,
        StartTime: time.Now(),
    }
}

func (ctx *ExecutionContext) LogToolCall(toolName, params string) {
    ctx.ToolCallCount++
    ctx.Logger.ToolCall(toolName, params)
}

func (ctx *ExecutionContext) LogToolResult(toolName string, success bool, output string, duration time.Duration) {
    ctx.Logger.ToolResult(toolName, success, output, duration)
}

func (ctx *ExecutionContext) LogThinking(content string) {
    ctx.Logger.AgentThinking(content)
}

func (ctx *ExecutionContext) LogResponse(content string) {
    ctx.Logger.AgentResponse(content)
}

func (ctx *ExecutionContext) LogProgress() {
    ctx.Logger.Progress(ctx.CurrentTurn, ctx.TotalTurns,
        fmt.Sprintf("Turn %d/%d", ctx.CurrentTurn, ctx.TotalTurns))
}
```

**更新**: `pkg/agent/base.go`

集成日志系统：

```go
func (a *BaseAgent) Run(ctx context.Context, input *Input) (*Output, error) {
    // 创建执行上下文
    execCtx := NewExecutionContext(input.Logger)

    // 记录会话开始
    execCtx.Logger.SessionStart(input.Task)

    // ... 初始化消息列表 ...

    maxTurns := input.MaxTurns
    if maxTurns == 0 {
        maxTurns = a.config.MaxTurns
    }
    execCtx.TotalTurns = maxTurns

    allToolCalls := make([]*tool.CallResult, 0)

    // Agent 运行循环
    for turn := 0; turn < maxTurns; turn++ {
        execCtx.CurrentTurn = turn + 1
        execCtx.LogProgress()

        execCtx.Logger.Info("Turn %d: Calling LLM...", turn+1)

        // 调用 LLM
        resp, err := a.llmClient.Chat(ctx, &llm.ChatRequest{
            Messages:    messages,
            Tools:       a.toolRegistry.GetToolDefinitions(),
            Temperature: input.Temperature,
            MaxTokens:   a.config.MaxTokens,
        })
        if err != nil {
            execCtx.Logger.Error("LLM call failed: %v", err)
            return nil, fmt.Errorf("LLM call failed: %w", err)
        }

        // 记录 Agent 响应
        if resp.Message.Content != "" {
            execCtx.LogResponse(resp.Message.Content)
        }

        // 添加助手消息
        messages = append(messages, resp.Message)

        // 检查是否完成
        if resp.StopReason == llm.StopReasonStop {
            execCtx.Logger.SessionEnd(
                time.Since(execCtx.StartTime),
                execCtx.ToolCallCount,
            )
            return &Output{
                Messages:  messages,
                Result:    resp.Message.Content,
                ToolCalls: allToolCalls,
            }, nil
        }

        // 处理工具调用
        if resp.StopReason == llm.StopReasonToolCalls {
            execCtx.Logger.Info("Executing %d tool call(s)...", len(resp.Message.ToolCalls))

            toolResults, err := a.executeToolsWithLogging(ctx, resp.Message.ToolCalls, execCtx)
            if err != nil {
                execCtx.Logger.Error("Tool execution failed: %v", err)
                return nil, fmt.Errorf("tool execution failed: %w", err)
            }

            allToolCalls = append(allToolCalls, toolResults...)

            // 添加工具结果消息
            for _, tr := range toolResults {
                messages = append(messages, llm.Message{
                    Role:       llm.RoleTool,
                    ToolCallID: tr.CallID,
                    Content:    tr.Result.Output,
                    Name:       tr.ToolName,
                    Timestamp:  tr.EndTime,
                })
            }

            continue
        }

        // ... 处理其他停止原因 ...
    }

    execCtx.Logger.Error("Max turns exceeded")
    return nil, fmt.Errorf("max turns (%d) exceeded", maxTurns)
}

func (a *BaseAgent) executeToolsWithLogging(
    ctx context.Context,
    toolCalls []*llm.ToolCall,
    execCtx *ExecutionContext,
) ([]*tool.CallResult, error) {
    results := make([]*tool.CallResult, len(toolCalls))

    for i, tc := range toolCalls {
        // 记录工具调用
        execCtx.LogToolCall(tc.Function.Name, tc.Function.Arguments)

        startTime := time.Now()

        t, err := a.toolRegistry.Get(tc.Function.Name)
        if err != nil {
            duration := time.Since(startTime)
            errorMsg := fmt.Sprintf("tool not found: %v", err)
            execCtx.LogToolResult(tc.Function.Name, false, errorMsg, duration)

            results[i] = &tool.CallResult{
                ToolName:  tc.Function.Name,
                CallID:    tc.ID,
                Result:    &tool.Result{Success: false, Error: errorMsg},
                StartTime: startTime,
                EndTime:   time.Now(),
            }
            continue
        }

        result, err := t.Execute(ctx, []byte(tc.Function.Arguments))
        duration := time.Since(startTime)

        if err != nil {
            errorMsg := fmt.Sprintf("execution error: %v", err)
            execCtx.LogToolResult(tc.Function.Name, false, errorMsg, duration)

            results[i] = &tool.CallResult{
                ToolName:  tc.Function.Name,
                CallID:    tc.ID,
                Result:    &tool.Result{Success: false, Error: errorMsg},
                StartTime: startTime,
                EndTime:   time.Now(),
            }
            continue
        }

        // 记录成功的工具结果
        execCtx.LogToolResult(tc.Function.Name, result.Success, result.Output, duration)

        results[i] = &tool.CallResult{
            ToolName:  tc.Function.Name,
            CallID:    tc.ID,
            Params:    []byte(tc.Function.Arguments),
            Result:    result,
            StartTime: startTime,
            EndTime:   time.Now(),
        }
    }

    return results, nil
}
```

**更新**: `pkg/agent/agent.go`

在 Input 中添加 Logger：

```go
type Input struct {
    Messages    []llm.Message
    Task        string
    MaxTurns    int
    Temperature float32
    Logger      *logger.Logger  // 新增
}
```

#### 1.8 基础 CLI

**文件**: `cmd/finta/main.go`

```go
package main

import (
    "context"
    "fmt"
    "os"

    "finta/internal/agent"
    "finta/internal/llm/openai"
    "finta/internal/logger"
    "finta/internal/tool"
    "finta/internal/tool/builtin"

    "github.com/spf13/cobra"
)

var (
    apiKey      string
    model       string
    temperature float32
    maxTurns    int
    verbose     bool
    noColor     bool
)

func main() {
    rootCmd := &cobra.Command{
        Use:   "finta",
        Short: "Finta AI Agent Framework",
        Long:  "A flexible AI agent framework inspired by ClaudeCode",
    }

    chatCmd := &cobra.Command{
        Use:   "chat [task]",
        Short: "Chat with an AI agent",
        Args:  cobra.MinimumNArgs(1),
        RunE:  runChat,
    }

    chatCmd.Flags().StringVar(&apiKey, "api-key", os.Getenv("OPENAI_API_KEY"), "OpenAI API key")
    chatCmd.Flags().StringVar(&model, "model", "gpt-4-turbo", "Model to use")
    chatCmd.Flags().Float32Var(&temperature, "temperature", 0.7, "Temperature")
    chatCmd.Flags().IntVar(&maxTurns, "max-turns", 10, "Maximum conversation turns")
    chatCmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose output (debug mode)")
    chatCmd.Flags().BoolVar(&noColor, "no-color", false, "Disable colored output")

    rootCmd.AddCommand(chatCmd)

    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}

func runChat(cmd *cobra.Command, args []string) error {
    if apiKey == "" {
        return fmt.Errorf("OpenAI API key required (set OPENAI_API_KEY or use --api-key)")
    }

    task := args[0]

    // 创建 Logger
    logLevel := logger.LevelInfo
    if verbose {
        logLevel = logger.LevelDebug
    }
    log := logger.NewLogger(os.Stdout, logLevel)
    if noColor {
        log.SetColorMode(false)
    }

    // 创建 LLM 客户端
    log.Debug("Creating LLM client (model: %s)", model)
    llmClient := openai.NewClient(apiKey, model)

    // 创建工具注册表
    log.Debug("Registering built-in tools")
    registry := tool.NewRegistry()
    registry.Register(builtin.NewReadTool())
    registry.Register(builtin.NewBashTool())

    log.Info("Registered %d tools: read, bash", 2)

    // 创建 Agent
    systemPrompt := `You are a helpful AI assistant with access to tools.
You can read files and execute bash commands.
Always provide clear, concise responses.`

    ag := agent.NewBaseAgent("general", systemPrompt, llmClient, registry, &agent.Config{
        Model:       model,
        Temperature: temperature,
        MaxTurns:    maxTurns,
    })

    log.Debug("Agent created with max_turns=%d, temperature=%.2f", maxTurns, temperature)

    // 运行 Agent
    output, err := ag.Run(context.Background(), &agent.Input{
        Task:        task,
        Temperature: temperature,
        Logger:      log,
    })
    if err != nil {
        log.Error("Agent execution failed: %v", err)
        return err
    }

    // 最终输出已经通过 logger 展示，这里不需要再打印
    log.Debug("Agent completed successfully")

    return nil
}
```

#### 1.9 测试运行

创建简单的测试：

```bash
# 设置 API key
export OPENAI_API_KEY="your-api-key"

# 构建
go build -o finta cmd/finta/main.go

# 测试基础功能（普通模式）
./finta chat "List files in the current directory"

# 测试详细输出（verbose 模式）
./finta chat --verbose "Read the go.mod file and tell me what it contains"

# 测试无颜色模式（适合日志文件）
./finta chat --no-color "Check if there are any .go files"
```

**期望的输出示例**:

```
══════════════════════════════════════════════════════════════════════
🚀 Session Started
  List files in the current directory
══════════════════════════════════════════════════════════════════════

15:30:45 [INFO] Registered 2 tools: read, bash
15:30:45 [INFO] Turn 1: Calling LLM...

🔧 Tool Call: bash
────────────────────────────────────────────────────────
{
  "command": "ls -la"
}
────────────────────────────────────────────────────────

📊 Tool Result: bash [✅ Success] (234ms)
────────────────────────────────────────────────────────
total 48
drwxr-xr-x  6 user user 4096 Dec 20 15:30 .
drwxr-xr-x 20 user user 4096 Dec 20 15:25 ..
-rw-r--r--  1 user user  156 Dec 20 15:20 go.mod
-rw-r--r--  1 user user  892 Dec 20 15:22 go.sum
drwxr-xr-x  3 user user 4096 Dec 20 15:30 cmd
drwxr-xr-x  8 user user 4096 Dec 20 15:30 pkg
────────────────────────────────────────────────────────

15:30:46 [INFO] Turn 2: Calling LLM...

💬 Agent Response
────────────────────────────────────────────────────────
I've listed the files in the current directory. Here's what I found:

The directory contains:
- `go.mod` and `go.sum`: Go module files
- `cmd/`: Directory containing command-line applications
- `pkg/`: Directory containing package code

This appears to be a Go project with a standard project structure.
────────────────────────────────────────────────────────

══════════════════════════════════════════════════════════════════════
✨ Session Completed
  Duration: 1.234s | Tool Calls: 1
══════════════════════════════════════════════════════════════════════
```

**添加 Logger 的辅助方法**：

在 `pkg/logger/logger.go` 中补充：

```go
func (l *Logger) SetColorMode(enabled bool) {
    l.colorMode = enabled
}

func (l *Logger) SetShowTime(enabled bool) {
    l.showTime = enabled
}
```

### Phase 1 完成标准

- ✅ 基础项目结构搭建完成
- ✅ LLM 客户端（OpenAI）可以正常调用
- ✅ 工具系统可以注册和执行工具
- ✅ Agent 可以运行 LLM + 工具的循环
- ✅ **日志系统完整实现，支持彩色输出和分级日志**
- ✅ **执行过程可视化，用户能清楚看到每一步**
- ✅ **工具调用参数、结果、耗时都有展示**
- ✅ CLI 可以接受任务并输出结果
- ✅ 至少有 2 个工具可用（Read, Bash）
- ✅ **支持 verbose 和 no-color 模式**

---

## Phase 2: 高级工具系统 (2-3 天)

### 目标

实现完整的工具能力，包括并行执行、更多内置工具、流式输出等。

### 实现步骤

#### 2.1 并行工具执行器

**文件**: `pkg/tool/executor.go`

```go
package tool

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
    "time"

    "finta/internal/llm"
)

type Executor struct {
    registry *Registry
}

func NewExecutor(registry *Registry) *Executor {
    return &Executor{registry: registry}
}

// ExecuteSequential 顺序执行工具
func (e *Executor) ExecuteSequential(ctx context.Context, toolCalls []*llm.ToolCall) ([]*CallResult, error) {
    results := make([]*CallResult, len(toolCalls))

    for i, tc := range toolCalls {
        result, err := e.executeOne(ctx, tc)
        if err != nil {
            return nil, err
        }
        results[i] = result
    }

    return results, nil
}

// ExecuteParallel 并行执行所有工具
func (e *Executor) ExecuteParallel(ctx context.Context, toolCalls []*llm.ToolCall) ([]*CallResult, error) {
    results := make([]*CallResult, len(toolCalls))
    errs := make([]error, len(toolCalls))

    var wg sync.WaitGroup
    for i, tc := range toolCalls {
        wg.Add(1)
        go func(idx int, call *llm.ToolCall) {
            defer wg.Done()

            result, err := e.executeOne(ctx, call)
            if err != nil {
                errs[idx] = err
                return
            }
            results[idx] = result
        }(i, tc)
    }

    wg.Wait()

    // 检查错误
    for _, err := range errs {
        if err != nil {
            return nil, err
        }
    }

    return results, nil
}

// ExecuteMixed 智能混合执行（根据依赖关系）
func (e *Executor) ExecuteMixed(ctx context.Context, toolCalls []*llm.ToolCall) ([]*CallResult, error) {
    // 分析依赖关系
    deps := e.analyzeDependencies(toolCalls)

    // 如果没有依赖，全部并行
    if len(deps) == 0 {
        return e.ExecuteParallel(ctx, toolCalls)
    }

    // 构建执行批次
    batches := e.buildExecutionBatches(toolCalls, deps)

    allResults := make([]*CallResult, 0, len(toolCalls))

    // 按批次执行
    for _, batch := range batches {
        batchCalls := make([]*llm.ToolCall, len(batch))
        for i, idx := range batch {
            batchCalls[i] = toolCalls[idx]
        }

        results, err := e.ExecuteParallel(ctx, batchCalls)
        if err != nil {
            return nil, err
        }

        allResults = append(allResults, results...)
    }

    return allResults, nil
}

func (e *Executor) executeOne(ctx context.Context, tc *llm.ToolCall) (*CallResult, error) {
    startTime := time.Now()

    t, err := e.registry.Get(tc.Function.Name)
    if err != nil {
        return &CallResult{
            ToolName:  tc.Function.Name,
            CallID:    tc.ID,
            Result:    &Result{Success: false, Error: err.Error()},
            StartTime: startTime,
            EndTime:   time.Now(),
        }, nil
    }

    result, err := t.Execute(ctx, []byte(tc.Function.Arguments))
    if err != nil {
        return &CallResult{
            ToolName:  tc.Function.Name,
            CallID:    tc.ID,
            Result:    &Result{Success: false, Error: err.Error()},
            StartTime: startTime,
            EndTime:   time.Now(),
        }, nil
    }

    return &CallResult{
        ToolName:  tc.Function.Name,
        CallID:    tc.ID,
        Params:    []byte(tc.Function.Arguments),
        Result:    result,
        StartTime: startTime,
        EndTime:   time.Now(),
    }, nil
}

// 简单的依赖分析：基于工具名称的启发式规则
func (e *Executor) analyzeDependencies(toolCalls []*llm.ToolCall) map[int][]int {
    deps := make(map[int][]int)

    // 规则：write 必须在 read 之前，bash 可能依赖 write
    for i, tc := range toolCalls {
        if tc.Function.Name == "read" || tc.Function.Name == "bash" {
            // 检查之前是否有 write
            for j := 0; j < i; j++ {
                if toolCalls[j].Function.Name == "write" {
                    deps[i] = append(deps[i], j)
                }
            }
        }
    }

    return deps
}

// 构建执行批次（拓扑排序的简化版本）
func (e *Executor) buildExecutionBatches(toolCalls []*llm.ToolCall, deps map[int][]int) [][]int {
    batches := make([][]int, 0)
    executed := make(map[int]bool)

    for len(executed) < len(toolCalls) {
        batch := make([]int, 0)

        for i := range toolCalls {
            if executed[i] {
                continue
            }

            // 检查依赖是否都已执行
            canExecute := true
            for _, dep := range deps[i] {
                if !executed[dep] {
                    canExecute = false
                    break
                }
            }

            if canExecute {
                batch = append(batch, i)
            }
        }

        if len(batch) == 0 {
            // 检测到循环依赖，强制执行剩余的
            for i := range toolCalls {
                if !executed[i] {
                    batch = append(batch, i)
                }
            }
        }

        for _, idx := range batch {
            executed[idx] = true
        }

        batches = append(batches, batch)
    }

    return batches
}
```

#### 2.2 更多内置工具

**文件**: `pkg/tool/builtin/write.go`

```go
package builtin

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"

    "finta/internal/tool"
)

type WriteTool struct{}

func NewWriteTool() *WriteTool {
    return &WriteTool{}
}

func (t *WriteTool) Name() string {
    return "write"
}

func (t *WriteTool) Description() string {
    return "Write content to a file (creates or overwrites)"
}

func (t *WriteTool) Parameters() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "file_path": map[string]any{
                "type":        "string",
                "description": "Path to the file to write",
            },
            "content": map[string]any{
                "type":        "string",
                "description": "Content to write to the file",
            },
        },
        "required": []string{"file_path", "content"},
    }
}

func (t *WriteTool) Execute(ctx context.Context, params json.RawMessage) (*tool.Result, error) {
    var p struct {
        FilePath string `json:"file_path"`
        Content  string `json:"content"`
    }

    if err := json.Unmarshal(params, &p); err != nil {
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("invalid parameters: %v", err),
        }, nil
    }

    // 确保目录存在
    dir := filepath.Dir(p.FilePath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("failed to create directory: %v", err),
        }, nil
    }

    // 写入文件
    if err := os.WriteFile(p.FilePath, []byte(p.Content), 0644); err != nil {
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("failed to write file: %v", err),
        }, nil
    }

    return &tool.Result{
        Success: true,
        Output:  fmt.Sprintf("Successfully wrote %d bytes to %s", len(p.Content), p.FilePath),
    }, nil
}
```

**文件**: `pkg/tool/builtin/glob.go`

```go
package builtin

import (
    "context"
    "encoding/json"
    "fmt"
    "path/filepath"
    "strings"

    "finta/internal/tool"
)

type GlobTool struct{}

func NewGlobTool() *GlobTool {
    return &GlobTool{}
}

func (t *GlobTool) Name() string {
    return "glob"
}

func (t *GlobTool) Description() string {
    return "Find files matching a glob pattern"
}

func (t *GlobTool) Parameters() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "pattern": map[string]any{
                "type":        "string",
                "description": "Glob pattern (e.g., '**/*.go', 'src/**/*.ts')",
            },
            "path": map[string]any{
                "type":        "string",
                "description": "Base path to search (default: current directory)",
            },
        },
        "required": []string{"pattern"},
    }
}

func (t *GlobTool) Execute(ctx context.Context, params json.RawMessage) (*tool.Result, error) {
    var p struct {
        Pattern string `json:"pattern"`
        Path    string `json:"path"`
    }

    if err := json.Unmarshal(params, &p); err != nil {
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("invalid parameters: %v", err),
        }, nil
    }

    basePath := p.Path
    if basePath == "" {
        basePath = "."
    }

    // 使用 filepath.Glob
    fullPattern := filepath.Join(basePath, p.Pattern)
    matches, err := filepath.Glob(fullPattern)
    if err != nil {
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("glob failed: %v", err),
        }, nil
    }

    if len(matches) == 0 {
        return &tool.Result{
            Success: true,
            Output:  "No files found",
        }, nil
    }

    return &tool.Result{
        Success: true,
        Output:  strings.Join(matches, "\n"),
        Data: map[string]any{
            "count": len(matches),
            "files": matches,
        },
    }, nil
}
```

#### 2.3 流式输出支持

**文件**: `pkg/llm/openai/streaming.go`

```go
package openai

import (
    "context"
    "fmt"
    "io"

    "finta/internal/llm"

    openai "github.com/sashabaranov/go-openai"
)

type StreamReader struct {
    stream *openai.ChatCompletionStream
}

func (c *Client) ChatStream(ctx context.Context, req *llm.ChatRequest) (llm.StreamReader, error) {
    messages := c.convertMessages(req.Messages)
    tools := c.convertTools(req.Tools)

    stream, err := c.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
        Model:       c.model,
        Messages:    messages,
        Tools:       tools,
        Temperature: req.Temperature,
        MaxTokens:   req.MaxTokens,
        Stream:      true,
    })
    if err != nil {
        return nil, err
    }

    return &StreamReader{stream: stream}, nil
}

func (s *StreamReader) Recv() (*llm.Delta, error) {
    resp, err := s.stream.Recv()
    if err == io.EOF {
        return &llm.Delta{Done: true}, nil
    }
    if err != nil {
        return nil, err
    }

    if len(resp.Choices) == 0 {
        return nil, fmt.Errorf("no choices in stream response")
    }

    delta := resp.Choices[0].Delta

    result := &llm.Delta{
        Role:    llm.Role(delta.Role),
        Content: delta.Content,
        Done:    false,
    }

    // 处理 tool calls
    if len(delta.ToolCalls) > 0 {
        result.ToolCalls = make([]*llm.ToolCall, len(delta.ToolCalls))
        for i, tc := range delta.ToolCalls {
            result.ToolCalls[i] = &llm.ToolCall{
                ID:   tc.ID,
                Type: string(tc.Type),
                Function: &llm.FunctionCall{
                    Name:      tc.Function.Name,
                    Arguments: tc.Function.Arguments,
                },
            }
        }
    }

    return result, nil
}

func (s *StreamReader) Close() error {
    s.stream.Close()
    return nil
}
```

**更新 Agent**: `pkg/agent/base.go` 添加流式方法

```go
func (a *BaseAgent) RunStreaming(ctx context.Context, input *Input, streamChan chan<- string) (*Output, error) {
    // 类似 Run，但使用 ChatStream 并将内容发送到 channel
    // 实现细节省略，参考 Run 方法的结构
}
```

#### 2.4 更新 CLI 支持流式输出

**文件**: `pkg/cli/streaming.go`

```go
package cli

import (
    "fmt"
    "io"
)

type StreamingWriter struct {
    writer io.Writer
}

func NewStreamingWriter(w io.Writer) *StreamingWriter {
    return &StreamingWriter{writer: w}
}

func (sw *StreamingWriter) Write(content string) {
    fmt.Fprint(sw.writer, content)
}

func (sw *StreamingWriter) WriteLine(content string) {
    fmt.Fprintln(sw.writer, content)
}
```

### Phase 2 完成标准

- ✅ 并行工具执行器实现
- ✅ 依赖分析和批次执行
- ✅ 至少 5 个内置工具（Read, Write, Bash, Glob, 再加一个）
- ✅ 流式输出支持
- ✅ CLI 支持流式显示

---

## Phase 3: 专门化 Agent (2-3 天)

### 目标

实现不同类型的专门化 Agent，支持 Agent 嵌套和任务分发。

### 实现步骤

#### 3.1 Agent 类型系统

**文件**: `pkg/agent/types.go`

```go
package agent

type AgentType string

const (
    AgentTypeGeneral AgentType = "general"
    AgentTypeExplore AgentType = "explore"
    AgentTypePlan    AgentType = "plan"
    AgentTypeExecute AgentType = "execute"
)

type Factory interface {
    CreateAgent(agentType AgentType) (Agent, error)
}

type DefaultFactory struct {
    llmClient    llm.Client
    toolRegistry *tool.Registry
}

func NewDefaultFactory(client llm.Client, registry *tool.Registry) *DefaultFactory {
    return &DefaultFactory{
        llmClient:    client,
        toolRegistry: registry,
    }
}

func (f *DefaultFactory) CreateAgent(agentType AgentType) (Agent, error) {
    switch agentType {
    case AgentTypeGeneral:
        return NewGeneralAgent(f.llmClient, f.toolRegistry), nil
    case AgentTypeExplore:
        return NewExploreAgent(f.llmClient, f.toolRegistry), nil
    case AgentTypePlan:
        return NewPlanAgent(f.llmClient, f.toolRegistry), nil
    default:
        return nil, fmt.Errorf("unknown agent type: %s", agentType)
    }
}
```

#### 3.2 Explore Agent

**文件**: `pkg/agent/specialized/explore.go`

```go
package specialized

import (
    "finta/internal/agent"
    "finta/internal/llm"
    "finta/internal/tool"
)

func NewExploreAgent(client llm.Client, registry *tool.Registry) agent.Agent {
    // 只允许只读工具
    readOnlyRegistry := tool.NewRegistry()
    readOnlyRegistry.Register(registry.Get("read"))
    readOnlyRegistry.Register(registry.Get("glob"))
    readOnlyRegistry.Register(registry.Get("grep"))
    readOnlyRegistry.Register(registry.Get("bash")) // 限制为只读命令

    systemPrompt := `You are an expert codebase exploration agent.

Your goal is to efficiently explore and understand codebases. You have access to read-only tools:
- read: Read file contents
- glob: Find files matching patterns
- grep: Search for content in files
- bash: Execute read-only commands (ls, find, etc.)

Best practices:
1. Start with glob to find relevant files
2. Use grep to search for specific patterns
3. Read files to understand implementation details
4. Be thorough but efficient

Always provide clear summaries of your findings.`

    return agent.NewBaseAgent(
        "explore",
        systemPrompt,
        client,
        readOnlyRegistry,
        &agent.Config{
            Model:       "gpt-4-turbo",
            Temperature: 0.3,
            MaxTurns:    15,
        },
    )
}
```

#### 3.3 Plan Agent

**文件**: `pkg/agent/specialized/plan.go`

```go
package specialized

import (
    "finta/internal/agent"
    "finta/internal/llm"
    "finta/internal/tool"
)

func NewPlanAgent(client llm.Client, registry *tool.Registry) agent.Agent {
    // 计划 Agent 可以读取但不修改
    planRegistry := tool.NewRegistry()
    planRegistry.Register(registry.Get("read"))
    planRegistry.Register(registry.Get("glob"))

    systemPrompt := `You are an expert software architect and planning agent.

Your goal is to create detailed, actionable implementation plans. You can read files to understand the current codebase state.

When creating plans:
1. Break down tasks into clear steps
2. Identify critical files to be modified
3. Consider architectural trade-offs
4. Suggest best practices
5. Anticipate potential issues

Output your plan in a structured markdown format with:
- Overview
- Implementation steps
- Files to modify
- Testing strategy
- Potential risks`

    return agent.NewBaseAgent(
        "plan",
        systemPrompt,
        client,
        planRegistry,
        &agent.Config{
            Model:       "gpt-4-turbo",
            Temperature: 0.5,
            MaxTurns:    10,
        },
    )
}
```

#### 3.4 Sub-Agent 工具

**文件**: `pkg/tool/builtin/task.go`

```go
package builtin

import (
    "context"
    "encoding/json"
    "fmt"

    "finta/internal/agent"
    "finta/internal/tool"
)

type TaskTool struct {
    factory agent.Factory
}

func NewTaskTool(factory agent.Factory) *TaskTool {
    return &TaskTool{factory: factory}
}

func (t *TaskTool) Name() string {
    return "task"
}

func (t *TaskTool) Description() string {
    return "Launch a specialized sub-agent to handle a specific task"
}

func (t *TaskTool) Parameters() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "agent_type": map[string]any{
                "type":        "string",
                "description": "Type of agent to spawn (explore, plan, execute)",
                "enum":        []string{"explore", "plan", "execute"},
            },
            "task": map[string]any{
                "type":        "string",
                "description": "Task description for the sub-agent",
            },
            "description": map[string]any{
                "type":        "string",
                "description": "Short description of what this sub-agent will do (3-5 words)",
            },
        },
        "required": []string{"agent_type", "task", "description"},
    }
}

func (t *TaskTool) Execute(ctx context.Context, params json.RawMessage) (*tool.Result, error) {
    var p struct {
        AgentType   string `json:"agent_type"`
        Task        string `json:"task"`
        Description string `json:"description"`
    }

    if err := json.Unmarshal(params, &p); err != nil {
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("invalid parameters: %v", err),
        }, nil
    }

    // 创建子 Agent
    subAgent, err := t.factory.CreateAgent(agent.AgentType(p.AgentType))
    if err != nil {
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("failed to create agent: %v", err),
        }, nil
    }

    // 运行子 Agent
    output, err := subAgent.Run(ctx, &agent.Input{
        Task:     p.Task,
        MaxTurns: 10,
    })
    if err != nil {
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("sub-agent failed: %v", err),
        }, nil
    }

    return &tool.Result{
        Success: true,
        Output:  fmt.Sprintf("[%s agent: %s]\n\n%s", p.AgentType, p.Description, output.Result),
        Data: map[string]any{
            "agent_type": p.AgentType,
            "tool_calls": len(output.ToolCalls),
        },
    }, nil
}
```

### Phase 3 完成标准

- ✅ Agent 类型系统和工厂模式
- ✅ Explore Agent 实现
- ✅ Plan Agent 实现
- ✅ Task 工具支持子 Agent 调用
- ✅ 不同 Agent 有不同的工具集和提示词

---

## Phase 4: MCP 集成 (3-4 天)

### 目标

完整实现 MCP (Model Context Protocol) 支持，能够加载和使用 MCP 服务器。

### 实现步骤

#### 4.1 MCP 客户端基础

**文件**: `pkg/mcp/client.go`

参考 Go MCP SDK，实现基础的 JSON-RPC 2.0 客户端。

核心方法：

- Initialize
- ListTools
- CallTool
- ListResources
- ReadResource

#### 4.2 Stdio Transport

**文件**: `pkg/mcp/transport/stdio.go`

实现通过 stdio 与 MCP 服务器通信。

#### 4.3 MCP Tool Adapter

**文件**: `pkg/mcp/adapter.go`

将 MCP 工具适配为 Finta 工具接口。

#### 4.4 Plugin Manager

**文件**: `pkg/mcp/manager.go`

管理多个 MCP 服务器，统一工具注册。

#### 4.5 配置支持

**文件**: `configs/default.yaml`

```yaml
mcp:
  servers:
    - name: filesystem
      transport: stdio
      command: npx
      args:
        - "-y"
        - "@modelcontextprotocol/server-filesystem"
        - "/home/user/projects"

    - name: github
      transport: stdio
      command: npx
      args:
        - "-y"
        - "@modelcontextprotocol/server-github"
      env:
        GITHUB_TOKEN: ${GITHUB_TOKEN}
```

### Phase 4 完成标准

- ✅ MCP JSON-RPC 2.0 客户端实现
- ✅ Stdio transport 工作
- ✅ MCP 工具可以适配为 Finta 工具
- ✅ 可以从配置加载多个 MCP 服务器
- ✅ MCP 工具与内置工具无缝集成

---

## Phase 4.5: Skills 技能库系统 (3-4 天)

### 目标

构建可复用的 AI 技能库系统（类似 Claude Skills 和组织过程资产 OPA），让 Agent 能够重用经过验证的工作流程和最佳实践。

### 背景

在项目管理中，**组织过程资产（OPA - Organizational Process Assets）** 是宝贵的知识库，包括：
- 经过验证的流程模板
- 最佳实践文档
- 历史项目的经验教训

Skills 系统将这一概念应用到 AI Agent 中：
- **复用性**: 一次定义，多次使用
- **标准化**: 确保 Agent 遵循最佳实践
- **可共享**: 团队成员可以共享技能定义
- **版本控制**: YAML 格式便于 Git 管理

### 实现步骤

#### 4.5.1 Skill 接口设计

**文件**: `internal/skill/skill.go`

```go
package skill

import (
    "context"
    "time"

    "finta/internal/agent"
    "finta/internal/llm"
)

// Skill 代表一个可复用的 AI 能力
type Skill interface {
    // 基础元数据
    Name() string
    Description() string
    Version() string
    Tags() []string // 用于分类和搜索

    // 执行技能
    Execute(ctx context.Context, input *SkillInput) (*SkillOutput, error)

    // 可选：技能依赖
    Dependencies() []string // 依赖的其他技能
}

// SkillInput 技能执行的输入
type SkillInput struct {
    Task    string         // 具体任务描述
    Context map[string]any // 上下文数据（文件列表、代码片段等）
    AgentFactory agent.Factory // Agent 工厂（用于 WorkflowSkill）
    Logger  interface{}    // Logger 实例
}

// SkillOutput 技能执行的输出
type SkillOutput struct {
    Result      string         // 执行结果（文本）
    Data        map[string]any // 结构化数据
    Messages    []llm.Message  // LLM 对话历史
    ToolCalls   int            // 使用的工具调用次数
    Duration    time.Duration  // 执行耗时
}

// Metadata 技能元数据
type Metadata struct {
    Name        string            `yaml:"name"`
    Version     string            `yaml:"version"`
    Description string            `yaml:"description"`
    Tags        []string          `yaml:"tags"`
    Author      string            `yaml:"author"`
    CreatedAt   time.Time         `yaml:"created_at"`
    UpdatedAt   time.Time         `yaml:"updated_at"`
    Dependencies []string         `yaml:"dependencies,omitempty"`
    Examples    []string          `yaml:"examples,omitempty"`
}
```

**设计要点**：
1. **接口抽象**: 支持多种技能实现方式
2. **上下文传递**: 允许技能间共享数据
3. **元数据丰富**: 便于发现和管理

#### 4.5.2 两种 Skill 实现类型

**文件**: `internal/skill/prompt_skill.go`

```go
package skill

import (
    "context"
    "fmt"
    "time"

    "finta/internal/agent"
)

// PromptSkill 基于提示词的简单技能（占 80%）
// 适用场景：单一任务，明确的输入输出
type PromptSkill struct {
    metadata     Metadata
    systemPrompt string      // Agent 的系统提示词
    agentType    string      // 使用的 Agent 类型
    maxTurns     int         // 最大轮次
    temperature  float32     // 温度参数
    examples     []Example   // 示例（few-shot learning）
}

type Example struct {
    Input  string `yaml:"input"`
    Output string `yaml:"output"`
}

func NewPromptSkill(meta Metadata, systemPrompt, agentType string) *PromptSkill {
    return &PromptSkill{
        metadata:     meta,
        systemPrompt: systemPrompt,
        agentType:    agentType,
        maxTurns:     10,
        temperature:  0.7,
    }
}

func (s *PromptSkill) Name() string        { return s.metadata.Name }
func (s *PromptSkill) Description() string { return s.metadata.Description }
func (s *PromptSkill) Version() string     { return s.metadata.Version }
func (s *PromptSkill) Tags() []string      { return s.metadata.Tags }
func (s *PromptSkill) Dependencies() []string { return s.metadata.Dependencies }

func (s *PromptSkill) Execute(ctx context.Context, input *SkillInput) (*SkillOutput, error) {
    startTime := time.Now()

    // 创建专门的 Agent
    ag, err := input.AgentFactory.CreateAgent(agent.AgentType(s.agentType))
    if err != nil {
        return nil, fmt.Errorf("failed to create agent: %w", err)
    }

    // 运行 Agent（使用自定义的 system prompt）
    agentInput := &agent.Input{
        Task:        input.Task,
        MaxTurns:    s.maxTurns,
        Temperature: s.temperature,
        Logger:      input.Logger.(*logger.Logger),
    }

    output, err := ag.Run(ctx, agentInput)
    if err != nil {
        return nil, fmt.Errorf("skill execution failed: %w", err)
    }

    return &SkillOutput{
        Result:    output.Result,
        Messages:  output.Messages,
        ToolCalls: len(output.ToolCalls),
        Duration:  time.Since(startTime),
    }, nil
}
```

**文件**: `internal/skill/workflow_skill.go`

```go
package skill

import (
    "context"
    "fmt"
    "time"
)

// WorkflowSkill 多步骤工作流技能（占 20%）
// 适用场景：复杂任务，需要多个 Agent 协作
type WorkflowSkill struct {
    metadata Metadata
    steps    []WorkflowStep
}

type WorkflowStep struct {
    Name        string `yaml:"name"`
    AgentType   string `yaml:"agent_type"`
    Task        string `yaml:"task_template"` // 支持模板变量
    Description string `yaml:"description"`
    ContinueOnError bool `yaml:"continue_on_error"`
}

func NewWorkflowSkill(meta Metadata, steps []WorkflowStep) *WorkflowSkill {
    return &WorkflowSkill{
        metadata: meta,
        steps:    steps,
    }
}

func (s *WorkflowSkill) Name() string        { return s.metadata.Name }
func (s *WorkflowSkill) Description() string { return s.metadata.Description }
func (s *WorkflowSkill) Version() string     { return s.metadata.Version }
func (s *WorkflowSkill) Tags() []string      { return s.metadata.Tags }
func (s *WorkflowSkill) Dependencies() []string { return s.metadata.Dependencies }

func (s *WorkflowSkill) Execute(ctx context.Context, input *SkillInput) (*SkillOutput, error) {
    startTime := time.Now()
    var allMessages []llm.Message
    totalToolCalls := 0
    results := make([]string, 0, len(s.steps))

    for i, step := range s.steps {
        // 创建 Agent
        ag, err := input.AgentFactory.CreateAgent(agent.AgentType(step.AgentType))
        if err != nil {
            if step.ContinueOnError {
                results = append(results, fmt.Sprintf("[Step %d FAILED: %v]", i+1, err))
                continue
            }
            return nil, fmt.Errorf("step %d failed: %w", i+1, err)
        }

        // 替换模板变量（简单实现）
        task := replaceTemplateVars(step.Task, input.Context)

        // 执行步骤
        agentInput := &agent.Input{
            Task:     task,
            MaxTurns: 10,
            Logger:   input.Logger.(*logger.Logger),
        }

        output, err := ag.Run(ctx, agentInput)
        if err != nil {
            if step.ContinueOnError {
                results = append(results, fmt.Sprintf("[Step %d FAILED: %v]", i+1, err))
                continue
            }
            return nil, fmt.Errorf("step %d execution failed: %w", i+1, err)
        }

        // 累积结果
        results = append(results, fmt.Sprintf("[Step %d: %s]\n%s", i+1, step.Name, output.Result))
        allMessages = append(allMessages, output.Messages...)
        totalToolCalls += len(output.ToolCalls)

        // 将结果添加到上下文供后续步骤使用
        input.Context[fmt.Sprintf("step_%d_result", i+1)] = output.Result
    }

    finalResult := strings.Join(results, "\n\n")

    return &SkillOutput{
        Result:    finalResult,
        Data:      input.Context,
        Messages:  allMessages,
        ToolCalls: totalToolCalls,
        Duration:  time.Since(startTime),
    }, nil
}

func replaceTemplateVars(template string, context map[string]any) string {
    result := template
    for key, value := range context {
        placeholder := fmt.Sprintf("{{.%s}}", key)
        result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
    }
    return result
}
```

#### 4.5.3 Skill Registry

**文件**: `internal/skill/registry.go`

```go
package skill

import (
    "fmt"
    "strings"
    "sync"
)

// Registry 技能注册表
type Registry struct {
    skills map[string]Skill
    mu     sync.RWMutex
}

func NewRegistry() *Registry {
    return &Registry{
        skills: make(map[string]Skill),
    }
}

// Register 注册技能
func (r *Registry) Register(skill Skill) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    name := skill.Name()
    if _, exists := r.skills[name]; exists {
        return fmt.Errorf("skill %s already registered", name)
    }

    r.skills[name] = skill
    return nil
}

// Get 获取技能
func (r *Registry) Get(name string) (Skill, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    skill, exists := r.skills[name]
    if !exists {
        return nil, fmt.Errorf("skill %s not found", name)
    }

    return skill, nil
}

// List 列出所有技能
func (r *Registry) List() []Skill {
    r.mu.RLock()
    defer r.mu.RUnlock()

    skills := make([]Skill, 0, len(r.skills))
    for _, skill := range r.skills {
        skills = append(skills, skill)
    }

    return skills
}

// Search 按标签搜索技能
func (r *Registry) Search(tags []string) []Skill {
    r.mu.RLock()
    defer r.mu.RUnlock()

    results := make([]Skill, 0)

    for _, skill := range r.skills {
        if hasAnyTag(skill.Tags(), tags) {
            results = append(results, skill)
        }
    }

    return results
}

func hasAnyTag(skillTags, searchTags []string) bool {
    for _, searchTag := range searchTags {
        for _, skillTag := range skillTags {
            if strings.EqualFold(skillTag, searchTag) {
                return true
            }
        }
    }
    return false
}
```

#### 4.5.4 YAML Storage

**文件**: `internal/skill/storage.go`

```go
package skill

import (
    "fmt"
    "os"
    "path/filepath"
    "gopkg.in/yaml.v3"
)

// SkillDefinition YAML 技能定义
type SkillDefinition struct {
    Metadata     Metadata       `yaml:"metadata"`
    Type         string         `yaml:"type"` // "prompt" or "workflow"
    SystemPrompt string         `yaml:"system_prompt,omitempty"`
    AgentType    string         `yaml:"agent_type,omitempty"`
    MaxTurns     int            `yaml:"max_turns,omitempty"`
    Temperature  float32        `yaml:"temperature,omitempty"`
    Examples     []Example      `yaml:"examples,omitempty"`
    Steps        []WorkflowStep `yaml:"steps,omitempty"`
}

// LoadFromYAML 从 YAML 文件加载技能
func LoadFromYAML(filePath string) (Skill, error) {
    data, err := os.ReadFile(filePath)
    if err != nil {
        return nil, fmt.Errorf("failed to read file: %w", err)
    }

    var def SkillDefinition
    if err := yaml.Unmarshal(data, &def); err != nil {
        return nil, fmt.Errorf("failed to parse YAML: %w", err)
    }

    // 验证
    if err := validateDefinition(&def); err != nil {
        return nil, fmt.Errorf("invalid skill definition: %w", err)
    }

    // 根据类型创建技能
    switch def.Type {
    case "prompt":
        skill := NewPromptSkill(def.Metadata, def.SystemPrompt, def.AgentType)
        if def.MaxTurns > 0 {
            skill.maxTurns = def.MaxTurns
        }
        if def.Temperature > 0 {
            skill.temperature = def.Temperature
        }
        skill.examples = def.Examples
        return skill, nil

    case "workflow":
        return NewWorkflowSkill(def.Metadata, def.Steps), nil

    default:
        return nil, fmt.Errorf("unknown skill type: %s", def.Type)
    }
}

// LoadAllFromDirectory 加载目录中所有 YAML 技能
func LoadAllFromDirectory(dirPath string) ([]Skill, error) {
    files, err := filepath.Glob(filepath.Join(dirPath, "*.yaml"))
    if err != nil {
        return nil, err
    }

    skills := make([]Skill, 0, len(files))

    for _, file := range files {
        skill, err := LoadFromYAML(file)
        if err != nil {
            // 记录错误但继续加载其他技能
            fmt.Fprintf(os.Stderr, "Warning: failed to load skill from %s: %v\n", file, err)
            continue
        }
        skills = append(skills, skill)
    }

    return skills, nil
}

func validateDefinition(def *SkillDefinition) error {
    if def.Metadata.Name == "" {
        return fmt.Errorf("skill name is required")
    }
    if def.Type == "" {
        return fmt.Errorf("skill type is required")
    }
    if def.Type == "prompt" && def.SystemPrompt == "" {
        return fmt.Errorf("system_prompt is required for prompt skills")
    }
    if def.Type == "workflow" && len(def.Steps) == 0 {
        return fmt.Errorf("steps are required for workflow skills")
    }
    return nil
}
```

#### 4.5.5 Skill Tool

**文件**: `internal/tool/builtin/skill.go`

```go
package builtin

import (
    "context"
    "encoding/json"
    "fmt"

    "finta/internal/skill"
    "finta/internal/tool"
)

type SkillTool struct {
    registry *skill.Registry
    factory  agent.Factory
}

func NewSkillTool(registry *skill.Registry, factory agent.Factory) *SkillTool {
    return &SkillTool{
        registry: registry,
        factory:  factory,
    }
}

func (t *SkillTool) Name() string {
    return "skill"
}

func (t *SkillTool) Description() string {
    return "Execute a registered skill (reusable AI capability)"
}

func (t *SkillTool) Parameters() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "name": map[string]any{
                "type":        "string",
                "description": "Name of the skill to execute",
            },
            "task": map[string]any{
                "type":        "string",
                "description": "Task description for the skill",
            },
            "context": map[string]any{
                "type":        "object",
                "description": "Additional context data (optional)",
            },
        },
        "required": []string{"name", "task"},
    }
}

func (t *SkillTool) Execute(ctx context.Context, params json.RawMessage) (*tool.Result, error) {
    var p struct {
        Name    string         `json:"name"`
        Task    string         `json:"task"`
        Context map[string]any `json:"context"`
    }

    if err := json.Unmarshal(params, &p); err != nil {
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("invalid parameters: %v", err),
        }, nil
    }

    // 获取技能
    sk, err := t.registry.Get(p.Name)
    if err != nil {
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("skill not found: %v", err),
        }, nil
    }

    // 获取 logger from context
    logger := agent.GetLoggerFromContext(ctx)

    // 执行技能
    input := &skill.SkillInput{
        Task:         p.Task,
        Context:      p.Context,
        AgentFactory: t.factory,
        Logger:       logger,
    }

    output, err := sk.Execute(ctx, input)
    if err != nil {
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("skill execution failed: %v", err),
        }, nil
    }

    return &tool.Result{
        Success: true,
        Output:  output.Result,
        Data: map[string]any{
            "skill_name":  sk.Name(),
            "tool_calls":  output.ToolCalls,
            "duration_ms": output.Duration.Milliseconds(),
        },
    }, nil
}
```

#### 4.5.6 内置技能示例

**文件**: `~/.finta/skills/code_review.yaml`

```yaml
metadata:
  name: code_review
  version: 1.0.0
  description: 系统化的代码审查流程
  tags: [code-quality, review, best-practices]
  author: finta-team

type: workflow

steps:
  - name: 代码发现
    agent_type: explore
    task_template: "分析 {{.file_path}} 的代码结构"
    description: 探索代码文件并理解其结构

  - name: 质量检查
    agent_type: general
    task_template: "审查代码质量，检查：1) 命名规范 2) 代码重复 3) 错误处理 4) 性能问题"
    description: 执行质量检查

  - name: 安全审计
    agent_type: general
    task_template: "检查安全问题：1) SQL 注入 2) XSS 3) CSRF 4) 敏感数据泄露"
    description: 安全漏洞扫描

  - name: 生成报告
    agent_type: general
    task_template: "基于前述分析，生成 Markdown 格式的代码审查报告"
    description: 汇总并生成审查报告
```

**文件**: `~/.finta/skills/commit.yaml`

```yaml
metadata:
  name: commit
  version: 1.0.0
  description: Git 提交信息规范化
  tags: [git, commit, best-practices]
  author: finta-team

type: prompt

agent_type: general
max_turns: 5
temperature: 0.5

system_prompt: |
  你是一个 Git 提交信息专家。根据代码变更生成符合约定式提交规范的提交信息。

  格式：
  <type>(<scope>): <subject>

  <body>

  <footer>

  类型（type）：
  - feat: 新功能
  - fix: 修复
  - docs: 文档
  - style: 格式
  - refactor: 重构
  - test: 测试
  - chore: 构建/工具

  示例：
  feat(auth): add OAuth2 login support

  - Implement OAuth2 flow
  - Add token refresh mechanism
  - Update user model

  Closes #123

examples:
  - input: "添加了用户登录功能，包括密码加密和会话管理"
    output: "feat(auth): implement user login with password encryption\n\n- Add bcrypt password hashing\n- Implement session management\n- Add login endpoint"

  - input: "修复了空指针异常的 bug"
    output: "fix(core): prevent nil pointer dereference\n\nFixed panic in user handler when email is nil\n\nCloses #456"
```

**文件**: `~/.finta/skills/debug.yaml`

```yaml
metadata:
  name: debug
  version: 1.0.0
  description: 系统化的调试流程
  tags: [debug, troubleshooting]
  author: finta-team

type: workflow

steps:
  - name: 问题复现
    agent_type: general
    task_template: "分析错误信息：{{.error_message}}，尝试理解问题原因"
    description: 理解和复现问题

  - name: 代码追踪
    agent_type: explore
    task_template: "查找相关代码文件，定位问题可能出现的位置"
    description: 追踪代码路径

  - name: 根因分析
    agent_type: general
    task_template: "基于代码分析，确定根本原因"
    description: 识别根本原因

  - name: 修复建议
    agent_type: general
    task_template: "提供修复方案和预防措施"
    description: 生成修复建议
```

**更多内置技能**：

- `refactor.yaml`: 重构工作流
- `test_plan.yaml`: 测试计划生成
- `documentation.yaml`: 文档生成

#### 4.5.7 CLI 集成

**文件**: `cmd/finta/main.go`

添加技能相关命令：

```go
func main() {
    rootCmd := &cobra.Command{
        Use:   "finta",
        Short: "Finta AI Agent Framework",
    }

    // 现有的 chat 命令
    chatCmd := &cobra.Command{...}

    // 新增：skill 命令组
    skillCmd := &cobra.Command{
        Use:   "skill",
        Short: "Manage and execute skills",
    }

    // skill list - 列出所有技能
    skillListCmd := &cobra.Command{
        Use:   "list",
        Short: "List all available skills",
        RunE:  runSkillList,
    }

    // skill run - 执行技能
    skillRunCmd := &cobra.Command{
        Use:   "run <skill-name> <task>",
        Short: "Execute a skill",
        Args:  cobra.MinimumNArgs(2),
        RunE:  runSkillRun,
    }

    // skill info - 查看技能详情
    skillInfoCmd := &cobra.Command{
        Use:   "info <skill-name>",
        Short: "Show skill information",
        Args:  cobra.ExactArgs(1),
        RunE:  runSkillInfo,
    }

    skillCmd.AddCommand(skillListCmd, skillRunCmd, skillInfoCmd)
    rootCmd.AddCommand(chatCmd, skillCmd)

    rootCmd.Execute()
}

func runSkillList(cmd *cobra.Command, args []string) error {
    // 加载技能
    skillsDir := filepath.Join(os.Getenv("HOME"), ".finta", "skills")
    skills, err := skill.LoadAllFromDirectory(skillsDir)
    if err != nil {
        return err
    }

    // 显示技能列表
    fmt.Println("Available Skills:")
    fmt.Println(strings.Repeat("=", 60))

    for _, sk := range skills {
        fmt.Printf("\n📦 %s (v%s)\n", sk.Name(), sk.Version())
        fmt.Printf("   %s\n", sk.Description())
        if len(sk.Tags()) > 0 {
            fmt.Printf("   Tags: %s\n", strings.Join(sk.Tags(), ", "))
        }
    }

    return nil
}

func runSkillRun(cmd *cobra.Command, args []string) error {
    skillName := args[0]
    task := args[1]

    // 加载技能
    skillsDir := filepath.Join(os.Getenv("HOME"), ".finta", "skills")
    skills, err := skill.LoadAllFromDirectory(skillsDir)
    if err != nil {
        return err
    }

    // 注册技能
    registry := skill.NewRegistry()
    for _, sk := range skills {
        registry.Register(sk)
    }

    // 获取技能
    sk, err := registry.Get(skillName)
    if err != nil {
        return fmt.Errorf("skill not found: %s", skillName)
    }

    // 创建 LLM 客户端和工具
    llmClient := openai.NewClient(os.Getenv("OPENAI_API_KEY"), "gpt-4-turbo")
    toolRegistry := tool.NewRegistry()
    // ... 注册基础工具

    factory := agent.NewDefaultFactory(llmClient, toolRegistry)
    log := logger.NewLogger(os.Stdout, logger.LevelInfo)

    // 执行技能
    ctx := context.Background()
    output, err := sk.Execute(ctx, &skill.SkillInput{
        Task:         task,
        AgentFactory: factory,
        Logger:       log,
    })
    if err != nil {
        return fmt.Errorf("skill execution failed: %w", err)
    }

    // 显示结果
    fmt.Println("\n" + output.Result)
    fmt.Printf("\n✨ Completed in %s (%d tool calls)\n", output.Duration, output.ToolCalls)

    return nil
}
```

### 使用示例

```bash
# 列出所有技能
$ finta skill list

Available Skills:
============================================================

📦 code_review (v1.0.0)
   系统化的代码审查流程
   Tags: code-quality, review, best-practices

📦 commit (v1.0.0)
   Git 提交信息规范化
   Tags: git, commit, best-practices

📦 debug (v1.0.0)
   系统化的调试流程
   Tags: debug, troubleshooting

# 执行技能
$ finta skill run code_review "审查 internal/agent/base.go"

[Step 1: 代码发现]
文件 internal/agent/base.go 包含 BaseAgent 的核心实现...

[Step 2: 质量检查]
✅ 命名规范良好
⚠️ 发现重复代码：executeToolsWithLogging 和 executeTools 有相似逻辑
✅ 错误处理完善

[Step 3: 安全审计]
✅ 未发现安全问题

[Step 4: 生成报告]
# 代码审查报告：internal/agent/base.go

## 总体评分：8/10

## 优点
- 清晰的接口设计
- 完善的错误处理

## 改进建议
1. 考虑将重复代码提取为辅助函数
2. 添加单元测试

✨ Completed in 12.5s (8 tool calls)

# 查看技能详情
$ finta skill info commit

📦 commit (v1.0.0)
Author: finta-team
Description: Git 提交信息规范化
Tags: git, commit, best-practices
Type: Prompt Skill
Agent: general

Examples:
1. Input: "添加了用户登录功能"
   Output: "feat(auth): implement user login..."
```

### 完成标准

- ✅ Skill 接口定义（支持 PromptSkill 和 WorkflowSkill）
- ✅ Skill Registry 实现（注册、获取、搜索）
- ✅ YAML 存储和加载
- ✅ Skill Tool 集成到工具系统
- ✅ 6 个内置技能示例
- ✅ CLI 支持 `skill list/run/info` 命令
- ✅ 技能可以嵌套调用（通过 AgentFactory）
- ✅ YAML 文件可以版本控制
- ✅ 技能加载时间 < 100ms
- ✅ 用户可以在 30 分钟内创建自定义技能

### 后续优化方向

1. **技能市场**: 支持从远程仓库下载技能
2. **技能测试**: 添加技能的单元测试框架
3. **参数验证**: 为技能添加 JSON Schema 验证
4. **技能依赖**: 自动解析和加载依赖技能
5. **性能优化**: 技能执行结果缓存

---

## Phase 5: Hook 系统 (2 天)

### 目标

实现生命周期 Hook 系统，支持用户自定义脚本在特定事件时执行。

### 实现步骤

#### 5.1 Hook 接口和注册表

**文件**: `pkg/hook/hook.go`

```go
package hook

import (
    "context"
    "time"
)

type LifecycleEvent string

const (
    EventSessionStart     LifecycleEvent = "session.start"
    EventSessionEnd       LifecycleEvent = "session.end"
    EventAgentStart       LifecycleEvent = "agent.start"
    EventAgentComplete    LifecycleEvent = "agent.complete"
    EventToolCallBefore   LifecycleEvent = "tool.call.before"
    EventToolCallAfter    LifecycleEvent = "tool.call.after"
)

type Event struct {
    Type      LifecycleEvent
    Data      map[string]any
    Timestamp time.Time
}

type Feedback struct {
    Continue bool
    Message  string
    Error    error
}

type Hook interface {
    Name() string
    Events() []LifecycleEvent
    Execute(ctx context.Context, event *Event) (*Feedback, error)
    Priority() int
}
```

#### 5.2 Shell Hook 实现

**文件**: `pkg/hook/shell.go`

```go
package hook

import (
    "context"
    "encoding/json"
    "os/exec"
)

type ShellHook struct {
    name     string
    events   []LifecycleEvent
    command  string
    args     []string
    priority int
}

func NewShellHook(name string, events []LifecycleEvent, command string, args []string, priority int) *ShellHook {
    return &ShellHook{
        name:     name,
        events:   events,
        command:  command,
        args:     args,
        priority: priority,
    }
}

func (h *ShellHook) Execute(ctx context.Context, event *Event) (*Feedback, error) {
    // 将事件数据作为 JSON 传递给命令
    eventJSON, _ := json.Marshal(event)

    cmd := exec.CommandContext(ctx, h.command, h.args...)
    cmd.Env = append(cmd.Env, "FINTA_EVENT="+string(eventJSON))

    output, err := cmd.CombinedOutput()
    if err != nil {
        return &Feedback{
            Continue: true,
            Error:    err,
        }, nil
    }

    return &Feedback{
        Continue: true,
        Message:  string(output),
    }, nil
}
```

#### 5.3 Hook Registry

**文件**: `pkg/hook/registry.go`

```go
package hook

import (
    "context"
    "sort"
    "sync"
)

type Registry struct {
    hooks map[LifecycleEvent][]Hook
    mu    sync.RWMutex
}

func NewRegistry() *Registry {
    return &Registry{
        hooks: make(map[LifecycleEvent][]Hook),
    }
}

func (r *Registry) Register(hook Hook) {
    r.mu.Lock()
    defer r.mu.Unlock()

    for _, event := range hook.Events() {
        r.hooks[event] = append(r.hooks[event], hook)
    }

    // 按优先级排序
    for event := range r.hooks {
        sort.Slice(r.hooks[event], func(i, j int) bool {
            return r.hooks[event][i].Priority() > r.hooks[event][j].Priority()
        })
    }
}

func (r *Registry) Trigger(ctx context.Context, event *Event) ([]*Feedback, error) {
    r.mu.RLock()
    hooks := r.hooks[event.Type]
    r.mu.RUnlock()

    feedbacks := make([]*Feedback, 0, len(hooks))

    for _, hook := range hooks {
        feedback, err := hook.Execute(ctx, event)
        if err != nil {
            return nil, err
        }

        feedbacks = append(feedbacks, feedback)

        // 如果 hook 要求停止，则不继续
        if !feedback.Continue {
            break
        }
    }

    return feedbacks, nil
}
```

#### 5.4 集成到 Agent

在 Agent 的关键位置触发 Hook：

- Run 开始时：`EventAgentStart`
- Run 结束时：`EventAgentComplete`
- 工具调用前后：`EventToolCallBefore`, `EventToolCallAfter`

### Phase 5 完成标准

- ✅ Hook 接口和注册表
- ✅ Shell Hook 实现
- ✅ Agent 集成 Hook 触发
- ✅ 配置文件支持定义 Hook
- ✅ Hook 反馈可以影响流程

---

## Phase 5.5: WBS 任务管理 (2-3 天)

### 目标

实现基于工作分解结构（WBS）的任务管理系统，让 Agent 能够系统化地分解和执行复杂任务，追踪任务状态和依赖关系。

### 背景

**工作分解结构（WBS - Work Breakdown Structure）** 是项目管理中的核心概念：
- **层次化**: 将大任务分解为可管理的小任务
- **可追踪**: 每个任务有明确的状态和完成标准
- **依赖管理**: 任务间有明确的先后关系
- **进度可视化**: 可以清晰看到整体进度

在 AI Agent 环境中，WBS 使得：
1. **Plan Agent** 可以输出结构化的任务分解
2. **Execute Agent** 可以按依赖关系执行任务
3. **General Agent** 可以查询和更新任务状态
4. 用户可以清楚看到 Agent 的工作进度

### 实现步骤

#### 5.5.1 Task 模型

**文件**: `internal/task/task.go`

```go
package task

import (
    "fmt"
    "time"
)

// TaskStatus 任务状态生命周期
type TaskStatus string

const (
    StatusPending    TaskStatus = "pending"     // 待执行
    StatusInProgress TaskStatus = "in_progress" // 执行中
    StatusBlocked    TaskStatus = "blocked"     // 被阻塞
    StatusCompleted  TaskStatus = "completed"   // 已完成
    StatusFailed     TaskStatus = "failed"      // 失败
)

// Task 任务模型
type Task struct {
    ID           string         `json:"id"`
    ParentID     string         `json:"parent_id,omitempty"`     // 父任务 ID（用于层次结构）
    Title        string         `json:"title"`
    Description  string         `json:"description"`
    Status       TaskStatus     `json:"status"`
    Priority     int            `json:"priority"`                 // 1-5 (1=最高)
    Dependencies []string       `json:"dependencies,omitempty"`   // 依赖的任务 IDs
    Assignee     string         `json:"assignee,omitempty"`       // Agent 类型或名称
    Metadata     map[string]any `json:"metadata,omitempty"`       // 附加数据
    CreatedAt    time.Time      `json:"created_at"`
    UpdatedAt    time.Time      `json:"updated_at"`
    StartedAt    *time.Time     `json:"started_at,omitempty"`
    CompletedAt  *time.Time     `json:"completed_at,omitempty"`
}

// NewTask 创建新任务
func NewTask(title, description string) *Task {
    now := time.Now()
    return &Task{
        ID:          generateID(),
        Title:       title,
        Description: description,
        Status:      StatusPending,
        Priority:    3, // 默认中等优先级
        Metadata:    make(map[string]any),
        CreatedAt:   now,
        UpdatedAt:   now,
    }
}

// CanStart 检查任务是否可以开始（依赖都已完成）
func (t *Task) CanStart(registry *Registry) bool {
    if t.Status != StatusPending {
        return false
    }

    for _, depID := range t.Dependencies {
        dep, err := registry.Get(depID)
        if err != nil || dep.Status != StatusCompleted {
            return false
        }
    }

    return true
}

// Start 开始任务
func (t *Task) Start() error {
    if t.Status != StatusPending {
        return fmt.Errorf("task %s is not pending", t.ID)
    }

    now := time.Now()
    t.Status = StatusInProgress
    t.StartedAt = &now
    t.UpdatedAt = now

    return nil
}

// Complete 完成任务
func (t *Task) Complete() error {
    if t.Status != StatusInProgress {
        return fmt.Errorf("task %s is not in progress", t.ID)
    }

    now := time.Now()
    t.Status = StatusCompleted
    t.CompletedAt = &now
    t.UpdatedAt = now

    return nil
}

// Fail 标记任务失败
func (t *Task) Fail(reason string) error {
    if t.Status == StatusCompleted {
        return fmt.Errorf("cannot fail completed task %s", t.ID)
    }

    t.Status = StatusFailed
    t.Metadata["failure_reason"] = reason
    t.UpdatedAt = time.Now()

    return nil
}

// Block 标记任务被阻塞
func (t *Task) Block(reason string) {
    t.Status = StatusBlocked
    t.Metadata["block_reason"] = reason
    t.UpdatedAt = time.Now()
}

func generateID() string {
    // 简单实现：使用时间戳 + 随机数
    return fmt.Sprintf("task-%d-%04d", time.Now().Unix(), time.Now().Nanosecond()%10000)
}
```

#### 5.5.2 Task Registry

**文件**: `internal/task/registry.go`

```go
package task

import (
    "fmt"
    "sort"
    "sync"
)

// Registry 任务注册表
type Registry struct {
    tasks map[string]*Task
    mu    sync.RWMutex
}

func NewRegistry() *Registry {
    return &Registry{
        tasks: make(map[string]*Task),
    }
}

// Create 创建任务
func (r *Registry) Create(task *Task) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if task.ID == "" {
        task.ID = generateID()
    }

    if _, exists := r.tasks[task.ID]; exists {
        return fmt.Errorf("task %s already exists", task.ID)
    }

    r.tasks[task.ID] = task
    return nil
}

// Get 获取任务
func (r *Registry) Get(id string) (*Task, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    task, exists := r.tasks[id]
    if !exists {
        return nil, fmt.Errorf("task %s not found", id)
    }

    return task, nil
}

// Update 更新任务
func (r *Registry) Update(task *Task) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if _, exists := r.tasks[task.ID]; !exists {
        return fmt.Errorf("task %s not found", task.ID)
    }

    task.UpdatedAt = time.Now()
    r.tasks[task.ID] = task

    return nil
}

// Delete 删除任务
func (r *Registry) Delete(id string) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if _, exists := r.tasks[id]; !exists {
        return fmt.Errorf("task %s not found", id)
    }

    delete(r.tasks, id)
    return nil
}

// List 列出所有任务
func (r *Registry) List() []*Task {
    r.mu.RLock()
    defer r.mu.RUnlock()

    tasks := make([]*Task, 0, len(r.tasks))
    for _, task := range r.tasks {
        tasks = append(tasks, task)
    }

    return tasks
}

// GetByStatus 按状态查询任务
func (r *Registry) GetByStatus(status TaskStatus) []*Task {
    r.mu.RLock()
    defer r.mu.RUnlock()

    results := make([]*Task, 0)

    for _, task := range r.tasks {
        if task.Status == status {
            results = append(results, task)
        }
    }

    return results
}

// GetByParent 获取子任务
func (r *Registry) GetByParent(parentID string) []*Task {
    r.mu.RLock()
    defer r.mu.RUnlock()

    results := make([]*Task, 0)

    for _, task := range r.tasks {
        if task.ParentID == parentID {
            results = append(results, task)
        }
    }

    return results
}

// GetRootTasks 获取根任务（没有父任务的任务）
func (r *Registry) GetRootTasks() []*Task {
    r.mu.RLock()
    defer r.mu.RUnlock()

    results := make([]*Task, 0)

    for _, task := range r.tasks {
        if task.ParentID == "" {
            results = append(results, task)
        }
    }

    return results
}

// GetNextTasks 获取可以开始的任务（依赖已满足，按优先级排序）
func (r *Registry) GetNextTasks() []*Task {
    r.mu.RLock()
    defer r.mu.RUnlock()

    results := make([]*Task, 0)

    for _, task := range r.tasks {
        if task.CanStart(r) {
            results = append(results, task)
        }
    }

    // 按优先级排序（优先级高的在前）
    sort.Slice(results, func(i, j int) bool {
        return results[i].Priority < results[j].Priority // 1 > 5
    })

    return results
}

// AddDependency 添加依赖关系
func (r *Registry) AddDependency(taskID, dependsOnID string) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    task, exists := r.tasks[taskID]
    if !exists {
        return fmt.Errorf("task %s not found", taskID)
    }

    if _, exists := r.tasks[dependsOnID]; !exists {
        return fmt.Errorf("dependency task %s not found", dependsOnID)
    }

    // 检查循环依赖
    if r.hasCircularDependency(taskID, dependsOnID) {
        return fmt.Errorf("circular dependency detected")
    }

    // 添加依赖
    for _, dep := range task.Dependencies {
        if dep == dependsOnID {
            return nil // 已存在
        }
    }

    task.Dependencies = append(task.Dependencies, dependsOnID)
    task.UpdatedAt = time.Now()

    return nil
}

// hasCircularDependency 检测循环依赖（深度优先搜索）
func (r *Registry) hasCircularDependency(taskID, newDepID string) bool {
    visited := make(map[string]bool)
    return r.dfsCircular(newDepID, taskID, visited)
}

func (r *Registry) dfsCircular(current, target string, visited map[string]bool) bool {
    if current == target {
        return true
    }

    if visited[current] {
        return false
    }

    visited[current] = true

    task, exists := r.tasks[current]
    if !exists {
        return false
    }

    for _, dep := range task.Dependencies {
        if r.dfsCircular(dep, target, visited) {
            return true
        }
    }

    return false
}

// GetProgress 获取整体进度
func (r *Registry) GetProgress() (completed, total int) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    total = len(r.tasks)
    for _, task := range r.tasks {
        if task.Status == StatusCompleted {
            completed++
        }
    }

    return
}
```

#### 5.5.3 WBS Tool

**文件**: `internal/tool/builtin/wbs.go`

```go
package builtin

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"

    "finta/internal/task"
    "finta/internal/tool"
)

type WBSTool struct {
    registry *task.Registry
}

func NewWBSTool(registry *task.Registry) *WBSTool {
    return &WBSTool{
        registry: registry,
    }
}

func (t *WBSTool) Name() string {
    return "wbs"
}

func (t *WBSTool) Description() string {
    return "Work Breakdown Structure (WBS) task management tool"
}

func (t *WBSTool) Parameters() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "action": map[string]any{
                "type": "string",
                "enum": []string{"create", "update", "get", "list", "next", "add_dependency"},
                "description": "Action to perform",
            },
            "task_id": map[string]any{
                "type":        "string",
                "description": "Task ID (for update/get/add_dependency)",
            },
            "title": map[string]any{
                "type":        "string",
                "description": "Task title (for create)",
            },
            "description": map[string]any{
                "type":        "string",
                "description": "Task description (for create)",
            },
            "status": map[string]any{
                "type":        "string",
                "enum":        []string{"pending", "in_progress", "blocked", "completed", "failed"},
                "description": "Task status (for update)",
            },
            "priority": map[string]any{
                "type":        "number",
                "description": "Priority 1-5, 1=highest (for create)",
            },
            "parent_id": map[string]any{
                "type":        "string",
                "description": "Parent task ID (for create)",
            },
            "depends_on": map[string]any{
                "type":        "string",
                "description": "Dependency task ID (for add_dependency)",
            },
        },
        "required": []string{"action"},
    }
}

func (t *WBSTool) Execute(ctx context.Context, params json.RawMessage) (*tool.Result, error) {
    var p struct {
        Action      string `json:"action"`
        TaskID      string `json:"task_id"`
        Title       string `json:"title"`
        Description string `json:"description"`
        Status      string `json:"status"`
        Priority    int    `json:"priority"`
        ParentID    string `json:"parent_id"`
        DependsOn   string `json:"depends_on"`
    }

    if err := json.Unmarshal(params, &p); err != nil {
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("invalid parameters: %v", err),
        }, nil
    }

    switch p.Action {
    case "create":
        return t.handleCreate(p)
    case "update":
        return t.handleUpdate(p)
    case "get":
        return t.handleGet(p.TaskID)
    case "list":
        return t.handleList(p.ParentID)
    case "next":
        return t.handleNext()
    case "add_dependency":
        return t.handleAddDependency(p.TaskID, p.DependsOn)
    default:
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("unknown action: %s", p.Action),
        }, nil
    }
}

func (t *WBSTool) handleCreate(p struct {
    Action      string `json:"action"`
    TaskID      string `json:"task_id"`
    Title       string `json:"title"`
    Description string `json:"description"`
    Status      string `json:"status"`
    Priority    int    `json:"priority"`
    ParentID    string `json:"parent_id"`
    DependsOn   string `json:"depends_on"`
}) (*tool.Result, error) {
    if p.Title == "" {
        return &tool.Result{
            Success: false,
            Error:   "title is required",
        }, nil
    }

    newTask := task.NewTask(p.Title, p.Description)
    if p.Priority > 0 && p.Priority <= 5 {
        newTask.Priority = p.Priority
    }
    if p.ParentID != "" {
        newTask.ParentID = p.ParentID
    }

    if err := t.registry.Create(newTask); err != nil {
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("failed to create task: %v", err),
        }, nil
    }

    return &tool.Result{
        Success: true,
        Output:  fmt.Sprintf("✓ Created task %s: %s (priority: %d)", newTask.ID, newTask.Title, newTask.Priority),
        Data: map[string]any{
            "task_id":  newTask.ID,
            "title":    newTask.Title,
            "priority": newTask.Priority,
        },
    }, nil
}

func (t *WBSTool) handleUpdate(p struct {
    Action      string `json:"action"`
    TaskID      string `json:"task_id"`
    Title       string `json:"title"`
    Description string `json:"description"`
    Status      string `json:"status"`
    Priority    int    `json:"priority"`
    ParentID    string `json:"parent_id"`
    DependsOn   string `json:"depends_on"`
}) (*tool.Result, error) {
    if p.TaskID == "" {
        return &tool.Result{
            Success: false,
            Error:   "task_id is required",
        }, nil
    }

    tsk, err := t.registry.Get(p.TaskID)
    if err != nil {
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("task not found: %v", err),
        }, nil
    }

    // 更新状态
    if p.Status != "" {
        switch task.TaskStatus(p.Status) {
        case task.StatusInProgress:
            if err := tsk.Start(); err != nil {
                return &tool.Result{Success: false, Error: err.Error()}, nil
            }
        case task.StatusCompleted:
            if err := tsk.Complete(); err != nil {
                return &tool.Result{Success: false, Error: err.Error()}, nil
            }
        case task.StatusFailed:
            tsk.Fail("Manual update")
        case task.StatusBlocked:
            tsk.Block("Manual update")
        default:
            tsk.Status = task.TaskStatus(p.Status)
        }
    }

    if err := t.registry.Update(tsk); err != nil {
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("failed to update: %v", err),
        }, nil
    }

    return &tool.Result{
        Success: true,
        Output:  fmt.Sprintf("✓ Updated task %s to status: %s", tsk.ID, tsk.Status),
        Data: map[string]any{
            "task_id": tsk.ID,
            "status":  string(tsk.Status),
        },
    }, nil
}

func (t *WBSTool) handleGet(taskID string) (*tool.Result, error) {
    if taskID == "" {
        return &tool.Result{
            Success: false,
            Error:   "task_id is required",
        }, nil
    }

    tsk, err := t.registry.Get(taskID)
    if err != nil {
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("task not found: %v", err),
        }, nil
    }

    output := fmt.Sprintf(`Task %s:
  Title: %s
  Status: %s
  Priority: %d
  Description: %s
  Dependencies: %v`,
        tsk.ID, tsk.Title, tsk.Status, tsk.Priority, tsk.Description, tsk.Dependencies)

    return &tool.Result{
        Success: true,
        Output:  output,
        Data: map[string]any{
            "task": tsk,
        },
    }, nil
}

func (t *WBSTool) handleList(parentID string) (*tool.Result, error) {
    var tasks []*task.Task

    if parentID != "" {
        tasks = t.registry.GetByParent(parentID)
    } else {
        tasks = t.registry.GetRootTasks()
    }

    if len(tasks) == 0 {
        return &tool.Result{
            Success: true,
            Output:  "No tasks found",
        }, nil
    }

    var lines []string
    for _, tsk := range tasks {
        statusEmoji := getStatusEmoji(tsk.Status)
        lines = append(lines, fmt.Sprintf("%s [P%d] %s - %s", statusEmoji, tsk.Priority, tsk.ID, tsk.Title))
    }

    completed, total := t.registry.GetProgress()
    output := fmt.Sprintf("Tasks (%d/%d completed):\n%s", completed, total, strings.Join(lines, "\n"))

    return &tool.Result{
        Success: true,
        Output:  output,
        Data: map[string]any{
            "tasks":     tasks,
            "total":     total,
            "completed": completed,
        },
    }, nil
}

func (t *WBSTool) handleNext() (*tool.Result, error) {
    tasks := t.registry.GetNextTasks()

    if len(tasks) == 0 {
        return &tool.Result{
            Success: true,
            Output:  "No tasks ready to start",
        }, nil
    }

    var lines []string
    for _, tsk := range tasks {
        lines = append(lines, fmt.Sprintf("[P%d] %s - %s", tsk.Priority, tsk.ID, tsk.Title))
    }

    output := fmt.Sprintf("Ready to start (%d tasks):\n%s", len(tasks), strings.Join(lines, "\n"))

    return &tool.Result{
        Success: true,
        Output:  output,
        Data: map[string]any{
            "tasks": tasks,
            "count": len(tasks),
        },
    }, nil
}

func (t *WBSTool) handleAddDependency(taskID, dependsOn string) (*tool.Result, error) {
    if taskID == "" || dependsOn == "" {
        return &tool.Result{
            Success: false,
            Error:   "task_id and depends_on are required",
        }, nil
    }

    if err := t.registry.AddDependency(taskID, dependsOn); err != nil {
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("failed to add dependency: %v", err),
        }, nil
    }

    return &tool.Result{
        Success: true,
        Output:  fmt.Sprintf("✓ Added dependency: %s depends on %s", taskID, dependsOn),
    }, nil
}

func getStatusEmoji(status task.TaskStatus) string {
    switch status {
    case task.StatusPending:
        return "⏸️"
    case task.StatusInProgress:
        return "▶️"
    case task.StatusCompleted:
        return "✅"
    case task.StatusFailed:
        return "❌"
    case task.StatusBlocked:
        return "🚫"
    default:
        return "❓"
    }
}
```

#### 5.5.4 集成到 Plan Agent

**文件**: `internal/agent/types.go`

更新 Plan Agent 的系统提示词：

```go
func (f *DefaultFactory) createPlanAgent() (Agent, error) {
    systemPrompt := `You are an expert software architect and planning agent.

Your goal is to create detailed, actionable implementation plans using Work Breakdown Structure (WBS).

When creating plans:
1. **Break down tasks** into clear, manageable steps
2. **Use the WBS tool** to create structured task hierarchies:
   - wbs(action="create", title="...", description="...", priority=1-5)
   - wbs(action="add_dependency", task_id="...", depends_on="...")
3. **Identify dependencies** between tasks
4. **Set priorities** (1=highest, 5=lowest)
5. **Consider architectural trade-offs**

Output structure:
- **Overview**: High-level summary
- **Task Breakdown**: Created via WBS tool
- **Execution Order**: Based on dependencies
- **Testing Strategy**: How to verify
- **Potential Risks**: Issues to watch

Always use the WBS tool to create the task structure.`

    // ... rest of implementation
}
```

### 使用示例

```bash
# Plan Agent 创建 WBS
$ finta chat --agent-type plan "Plan implementation of user authentication"

[Agent creates WBS structure]

✓ Created task task-1234: Database schema (priority: 1)
✓ Created task task-1235: API endpoints (priority: 2)
✓ Created task task-1236: Frontend integration (priority: 3)
✓ Added dependency: task-1235 depends on task-1234
✓ Added dependency: task-1236 depends on task-1235

**Plan Overview**:
Authentication system with 3-tier architecture...

[Task list shows]
Tasks (0/3 completed):
⏸️ [P1] task-1234 - Database schema
⏸️ [P2] task-1235 - API endpoints
⏸️ [P3] task-1236 - Frontend integration

# Execute Agent 查询下一步
$ finta chat --agent-type execute "What tasks are ready to start?"

[Agent uses WBS tool]
Ready to start (1 task):
[P1] task-1234 - Database schema

# Execute Agent 更新任务状态
$ finta chat --agent-type execute "Start task-1234"

✓ Updated task task-1234 to status: in_progress

# 完成任务
$ finta chat --agent-type execute "Mark task-1234 as completed"

✓ Updated task task-1234 to status: completed

# 查看进度
$ finta chat "Show all tasks"

Tasks (1/3 completed):
✅ [P1] task-1234 - Database schema
⏸️ [P2] task-1235 - API endpoints
⏸️ [P3] task-1236 - Frontend integration
```

### 完成标准

- ✅ Task 模型with 5 种状态（pending, in_progress, blocked, completed, failed）
- ✅ Task Registry 支持 CRUD 和依赖管理
- ✅ WBS Tool 实现 6 个操作（create, update, get, list, next, add_dependency）
- ✅ 循环依赖检测功能
- ✅ 状态转换验证（pending → in_progress → completed）
- ✅ Plan Agent 使用 WBS 创建任务结构
- ✅ Execute Agent 可以查询和更新任务状态
- ✅ 进度追踪（X/Y completed）
- ✅ 优先级排序

### 后续优化方向

1. **持久化**: 将 WBS 保存到数据库或文件
2. **可视化**: 生成任务树状图（ASCII art 或 GraphViz）
3. **时间估算**: 添加任务耗时估算和实际耗时记录
4. **资源分配**: 支持多 Agent 并行执行任务
5. **模板**: 预定义的任务模板（如"实现 REST API"）

---

## Phase 6: Session 管理 (2 天)

### 目标

实现会话持久化和上下文管理，支持长时间对话。

### 实现步骤

#### 6.1 Session 接口

**文件**: `pkg/session/session.go`

```go
package session

import (
    "context"
    "finta/internal/llm"
    "time"
)

type Session interface {
    ID() string
    AddMessage(msg llm.Message) error
    GetMessages() []llm.Message
    Save(ctx context.Context) error
    Load(ctx context.Context, sessionID string) error
}

type SessionData struct {
    ID        string
    Messages  []llm.Message
    Metadata  map[string]any
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

#### 6.2 SQLite 持久化

**文件**: `pkg/session/persistence.go`

使用 SQLite 存储会话数据：

```go
package session

import (
    "context"
    "database/sql"
    "encoding/json"

    _ "github.com/mattn/go-sqlite3"
)

type SQLitePersistence struct {
    db *sql.DB
}

func NewSQLitePersistence(dbPath string) (*SQLitePersistence, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, err
    }

    // 创建表
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS sessions (
            id TEXT PRIMARY KEY,
            messages TEXT,
            metadata TEXT,
            created_at DATETIME,
            updated_at DATETIME
        )
    `)
    if err != nil {
        return nil, err
    }

    return &SQLitePersistence{db: db}, nil
}

func (p *SQLitePersistence) Save(ctx context.Context, data *SessionData) error {
    messagesJSON, _ := json.Marshal(data.Messages)
    metadataJSON, _ := json.Marshal(data.Metadata)

    _, err := p.db.ExecContext(ctx, `
        INSERT OR REPLACE INTO sessions (id, messages, metadata, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?)
    `, data.ID, messagesJSON, metadataJSON, data.CreatedAt, data.UpdatedAt)

    return err
}

func (p *SQLitePersistence) Load(ctx context.Context, sessionID string) (*SessionData, error) {
    // 实现加载逻辑
}
```

#### 6.3 Context Summarization

**文件**: `pkg/session/summarizer.go`

当消息过多时，使用 LLM 生成摘要：

```go
package session

import (
    "context"
    "finta/internal/llm"
)

type Summarizer struct {
    llmClient llm.Client
}

func (s *Summarizer) Summarize(ctx context.Context, messages []llm.Message) (string, error) {
    // 使用 LLM 生成对话摘要
}
```

### Phase 6 完成标准

- ✅ Session 接口和基础实现
- ✅ SQLite 持久化
- ✅ 会话可以保存和加载
- ✅ 上下文摘要功能
- ✅ CLI 支持恢复历史会话

---

## Phase 6.5: PMP 生命周期集成 (1-2 天)

### 目标

集成项目管理专业（PMP）的 5 个过程组概念，让 Agent 能够自主进行任务阶段识别和进度跟踪，但不强制要求严格的工作流。

### 背景

**PMP 5 个过程组（Process Groups）**：
1. **Initiating (启动)**: 定义项目目标和可行性
2. **Planning (规划)**: 制定详细计划和任务分解
3. **Executing (执行)**: 实施计划中的任务
4. **Monitoring & Controlling (监控)**: 跟踪进度，处理偏差
5. **Closing (收尾)**: 验证完成，总结经验

在 Finta 中的体现：
- **概念性而非强制性**: 不是状态机，而是提示引导
- **AI 自主决策**: Agent 根据任务上下文自行过渡阶段
- **与 WBS 集成**: Lifecycle 阶段 + WBS 任务 = 完整项目管理
- **可视化引导**: 为用户和 Agent 提供当前阶段信息

### 实现步骤

#### 6.5.1 Lifecycle 模型

**文件**: `internal/lifecycle/lifecycle.go`

```go
package lifecycle

import (
    "fmt"
    "time"
)

// Phase PMP 过程组阶段
type Phase string

const (
    PhaseInitiate Phase = "initiate" // 🎯 启动：理解需求
    PhasePlan     Phase = "plan"     // 📋 规划：任务分解
    PhaseExecute  Phase = "execute"  // ⚙️ 执行：实施任务
    PhaseMonitor  Phase = "monitor"  // 📊 监控：跟踪进度
    PhaseClose    Phase = "close"    // ✅ 收尾：验证完成
)

// PhaseTransition 阶段转换记录
type PhaseTransition struct {
    FromPhase Phase     `json:"from_phase"`
    ToPhase   Phase     `json:"to_phase"`
    Timestamp time.Time `json:"timestamp"`
    Trigger   string    `json:"trigger"` // 触发原因
}

// Lifecycle 生命周期管理
type Lifecycle struct {
    CurrentPhase Phase             `json:"current_phase"`
    PhaseHistory []PhaseTransition `json:"phase_history"`
    Metadata     map[string]any    `json:"metadata,omitempty"`
    CreatedAt    time.Time         `json:"created_at"`
    UpdatedAt    time.Time         `json:"updated_at"`
}

// NewLifecycle 创建新的生命周期（默认从 Initiate 开始）
func NewLifecycle() *Lifecycle {
    now := time.Now()
    return &Lifecycle{
        CurrentPhase: PhaseInitiate,
        PhaseHistory: make([]PhaseTransition, 0),
        Metadata:     make(map[string]any),
        CreatedAt:    now,
        UpdatedAt:    now,
    }
}

// Transition 转换到新阶段
func (lc *Lifecycle) Transition(toPhase Phase, trigger string) {
    transition := PhaseTransition{
        FromPhase: lc.CurrentPhase,
        ToPhase:   toPhase,
        Timestamp: time.Now(),
        Trigger:   trigger,
    }

    lc.CurrentPhase = toPhase
    lc.PhaseHistory = append(lc.PhaseHistory, transition)
    lc.UpdatedAt = time.Now()
}

// GetPhaseEmoji 获取阶段对应的 Emoji
func (lc *Lifecycle) GetPhaseEmoji() string {
    switch lc.CurrentPhase {
    case PhaseInitiate:
        return "🎯"
    case PhasePlan:
        return "📋"
    case PhaseExecute:
        return "⚙️"
    case PhaseMonitor:
        return "📊"
    case PhaseClose:
        return "✅"
    default:
        return "❓"
    }
}

// GetPhaseGuidance 获取当前阶段的引导信息
func (lc *Lifecycle) GetPhaseGuidance() string {
    switch lc.CurrentPhase {
    case PhaseInitiate:
        return `**Current Phase: Initiating** 🎯
- Understand the requirements and objectives
- Identify stakeholders and constraints
- Assess feasibility
- Define success criteria`

    case PhasePlan:
        return `**Current Phase: Planning** 📋
- Break down work into tasks (use WBS tool)
- Identify dependencies
- Estimate effort and resources
- Create detailed execution plan`

    case PhaseExecute:
        return `**Current Phase: Executing** ⚙️
- Implement tasks according to plan
- Query next tasks from WBS
- Update task status as you progress
- Document changes and decisions`

    case PhaseMonitor:
        return `**Current Phase: Monitoring** 📊
- Check WBS progress (X/Y completed)
- Identify blockers and resolve them
- Adjust plan if needed
- Communicate status`

    case PhaseClose:
        return `**Current Phase: Closing** ✅
- Verify all tasks completed
- Test and validate deliverables
- Document lessons learned
- Prepare final summary`

    default:
        return "Unknown phase"
    }
}

// SuggestNextPhase 根据上下文建议下一个阶段（不强制）
func (lc *Lifecycle) SuggestNextPhase(tasksCompleted, tasksTotal int) Phase {
    switch lc.CurrentPhase {
    case PhaseInitiate:
        // 需求已理解 → 进入规划
        return PhasePlan

    case PhasePlan:
        // 计划已制定（WBS 已创建）→ 进入执行
        if tasksTotal > 0 {
            return PhaseExecute
        }
        return PhasePlan

    case PhaseExecute:
        // 任务进行中 → 监控
        if tasksCompleted > 0 && tasksCompleted < tasksTotal {
            return PhaseMonitor
        }
        // 所有任务完成 → 收尾
        if tasksCompleted == tasksTotal && tasksTotal > 0 {
            return PhaseClose
        }
        return PhaseExecute

    case PhaseMonitor:
        // 持续监控，可回到执行
        if tasksCompleted == tasksTotal && tasksTotal > 0 {
            return PhaseClose
        }
        return PhaseExecute

    case PhaseClose:
        // 已收尾，保持不变
        return PhaseClose

    default:
        return PhaseInitiate
    }
}
```

#### 6.5.2 集成到 Session

**文件**: `internal/session/session.go`

在现有 SessionData 结构中添加 Lifecycle 字段：

```go
package session

import (
    "time"

    "finta/internal/lifecycle"
    "finta/internal/llm"
)

type SessionData struct {
    ID           string         `json:"id"`
    Messages     []llm.Message  `json:"messages"`
    StartTime    time.Time      `json:"start_time"`
    UpdatedTime  time.Time      `json:"updated_time"`
    Metadata     map[string]any `json:"metadata"`

    // 🆕 新增：PMP 生命周期
    Lifecycle    *lifecycle.Lifecycle `json:"lifecycle,omitempty"`
}

func NewSession(id string) *SessionData {
    return &SessionData{
        ID:          id,
        Messages:    make([]llm.Message, 0),
        StartTime:   time.Now(),
        UpdatedTime: time.Now(),
        Metadata:    make(map[string]any),
        Lifecycle:   lifecycle.NewLifecycle(), // 🆕 初始化生命周期
    }
}
```

#### 6.5.3 集成到 Agent 提示词

**文件**: `internal/agent/types.go`

更新各 Agent 的 system prompt 包含生命周期信息：

```go
func (f *DefaultFactory) createGeneralAgent() (Agent, error) {
    systemPrompt := `You are a helpful AI assistant with access to tools.
You can read files, execute bash commands, write files, find files with glob
patterns, and search files with grep.

当前项目阶段信息（如果有）将在任务描述中提供。
根据当前阶段，调整你的工作方式：
- **启动阶段** (🎯): 重点理解需求，询问澄清问题
- **规划阶段** (📋): 使用 WBS 工具创建任务结构
- **执行阶段** (⚙️): 查询 WBS 获取下一个任务并执行
- **监控阶段** (📊): 检查进度，处理阻塞任务
- **收尾阶段** (✅): 验证完成，生成总结

When solving tasks, follow the ReAct pattern:
1. **Think**: Explain your reasoning before taking action
2. **Act**: Use tools to gather information or make changes
3. **Observe**: Analyze the results and plan next steps

Always provide clear, concise responses.`

    return NewBaseAgent(
        "general",
        systemPrompt,
        f.llmClient,
        f.toolRegistry,
        &Config{
            Model:              "gpt-4-turbo",
            Temperature:        0.7,
            MaxTokens:          4096,
            MaxTurns:           20,
            EnableParallelTools: true,
            ToolExecutionMode:   tool.ExecutionModeMixed,
        },
    ), nil
}
```

#### 6.5.4 Lifecycle Tool（可选）

**文件**: `internal/tool/builtin/lifecycle.go`

```go
package builtin

import (
    "context"
    "encoding/json"
    "fmt"

    "finta/internal/lifecycle"
    "finta/internal/tool"
)

type LifecycleTool struct {
    lc *lifecycle.Lifecycle
}

func NewLifecycleTool(lc *lifecycle.Lifecycle) *LifecycleTool {
    return &LifecycleTool{
        lc: lc,
    }
}

func (t *LifecycleTool) Name() string {
    return "lifecycle"
}

func (t *LifecycleTool) Description() string {
    return "Query or transition project lifecycle phase (PMP process groups)"
}

func (t *LifecycleTool) Parameters() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "action": map[string]any{
                "type": "string",
                "enum": []string{"query", "transition"},
                "description": "Action: query current phase or transition to new phase",
            },
            "to_phase": map[string]any{
                "type": "string",
                "enum": []string{"initiate", "plan", "execute", "monitor", "close"},
                "description": "Target phase (for transition action)",
            },
            "trigger": map[string]any{
                "type":        "string",
                "description": "Reason for phase transition",
            },
        },
        "required": []string{"action"},
    }
}

func (t *LifecycleTool) Execute(ctx context.Context, params json.RawMessage) (*tool.Result, error) {
    var p struct {
        Action  string `json:"action"`
        ToPhase string `json:"to_phase"`
        Trigger string `json:"trigger"`
    }

    if err := json.Unmarshal(params, &p); err != nil {
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("invalid parameters: %v", err),
        }, nil
    }

    switch p.Action {
    case "query":
        return t.handleQuery()
    case "transition":
        return t.handleTransition(p.ToPhase, p.Trigger)
    default:
        return &tool.Result{
            Success: false,
            Error:   fmt.Sprintf("unknown action: %s", p.Action),
        }, nil
    }
}

func (t *LifecycleTool) handleQuery() (*tool.Result, error) {
    emoji := t.lc.GetPhaseEmoji()
    guidance := t.lc.GetPhaseGuidance()

    output := fmt.Sprintf("%s %s\n\n%s", emoji, t.lc.CurrentPhase, guidance)

    return &tool.Result{
        Success: true,
        Output:  output,
        Data: map[string]any{
            "current_phase": string(t.lc.CurrentPhase),
            "emoji":         emoji,
        },
    }, nil
}

func (t *LifecycleTool) handleTransition(toPhase, trigger string) (*tool.Result, error) {
    if toPhase == "" {
        return &tool.Result{
            Success: false,
            Error:   "to_phase is required",
        }, nil
    }

    if trigger == "" {
        trigger = "Manual transition"
    }

    phase := lifecycle.Phase(toPhase)
    t.lc.Transition(phase, trigger)

    emoji := t.lc.GetPhaseEmoji()
    output := fmt.Sprintf("✓ Transitioned to %s %s\nReason: %s",
        emoji, toPhase, trigger)

    return &tool.Result{
        Success: true,
        Output:  output,
        Data: map[string]any{
            "phase": toPhase,
        },
    }, nil
}
```

#### 6.5.5 CLI 显示生命周期信息

**文件**: `cmd/finta/main.go`

在 session 开始时显示当前阶段：

```go
func runChat(cmd *cobra.Command, args []string) error {
    // ... 现有代码 ...

    // 如果有 session，显示生命周期信息
    if session != nil && session.Lifecycle != nil {
        emoji := session.Lifecycle.GetPhaseEmoji()
        log.Info("%s Current Phase: %s", emoji, session.Lifecycle.CurrentPhase)
        log.Debug(session.Lifecycle.GetPhaseGuidance())
    }

    // ... 继续执行 ...
}
```

### 使用示例

```bash
# 启动新会话（自动进入 Initiate 阶段）
$ finta chat "Build a user authentication system"

🎯 Current Phase: initiate

[Agent asks clarifying questions about requirements]

# Agent 自动过渡到 Plan 阶段
$ finta chat --continue "Create a detailed plan"

📋 Current Phase: plan

[Agent uses WBS tool to create task structure]

# Agent 过渡到 Execute 阶段
$ finta chat --continue "Start implementing"

⚙️ Current Phase: execute

[Agent queries WBS for next task and begins implementation]

# 手动查询当前阶段
$ finta chat "What phase are we in?"

📊 Current Phase: monitor

**Current Phase: Monitoring**
- Check WBS progress (2/5 completed)
- Identify blockers and resolve them
- Adjust plan if needed

# 完成所有任务后，Agent 自动过渡到 Close
$ finta chat "All tasks completed, verify and summarize"

✅ Current Phase: close

[Agent verifies completion and generates summary]
```

### 与其他组件的集成

**完整流程示例**：

```
🎯 Initiate Phase
  ↓
  User: "Build authentication system"
  Agent: Uses general reasoning, asks clarifying questions

📋 Plan Phase
  ↓
  Agent: Creates WBS tasks
  wbs(action="create", title="Database schema", priority=1)
  wbs(action="create", title="API endpoints", priority=2)
  wbs(action="add_dependency", ...)

⚙️ Execute Phase
  ↓
  Agent: Queries WBS for next task
  wbs(action="next") → [task-1234]
  Executes task-1234
  wbs(action="update", task_id="task-1234", status="completed")

📊 Monitor Phase
  ↓
  Agent: Checks progress
  wbs(action="list") → "Tasks (2/5 completed)"
  Identifies blockers, adjusts plan

✅ Close Phase
  ↓
  Agent: Verifies all tasks completed
  wbs(action="list") → "Tasks (5/5 completed)"
  Generates final summary and lessons learned
```

### 完成标准

- ✅ Lifecycle 阶段模型（5 个 PMP 过程组）
- ✅ Session 包含 lifecycle 字段
- ✅ 阶段过渡历史记录
- ✅ 每个阶段有对应的 Emoji 和引导信息
- ✅ Agent 提示词包含阶段信息
- ✅ 不强制工作流（AI 自主决策过渡）
- ✅ Lifecycle Tool 提供查询和手动过渡功能
- ✅ CLI 显示当前阶段

### 关键设计原则

1. **非强制性**: Lifecycle 是引导而非约束，Agent 可以自由决定何时过渡
2. **集成性**: 与 WBS、Skills、ReAct 自然配合
3. **可视化**: 清晰的阶段指示帮助用户理解进度
4. **AI 驱动**: Agent 根据任务上下文自主识别阶段

### 后续优化方向

1. **自动过渡**: 基于 WBS 进度自动建议阶段过渡
2. **阶段模板**: 每个阶段预定义的检查清单
3. **历史分析**: 分析不同项目的阶段耗时模式
4. **自定义阶段**: 允许用户定义自己的工作流阶段
5. **阶段报告**: 自动生成每个阶段的总结报告

---

## Phase 7: 配置系统 (1-2 天)

### 目标

完整的 YAML 配置支持，可配置所有组件。

### 实现步骤

#### 7.1 配置结构

**文件**: `pkg/config/config.go`

```go
package config

type Config struct {
    LLM     LLMConfig     `yaml:"llm"`
    Agent   AgentConfig   `yaml:"agent"`
    Session SessionConfig `yaml:"session"`
    Tools   ToolsConfig   `yaml:"tools"`
    MCP     MCPConfig     `yaml:"mcp"`
    Hooks   []HookConfig  `yaml:"hooks"`
    CLI     CLIConfig     `yaml:"cli"`
}

type LLMConfig struct {
    Provider    string  `yaml:"provider"`
    APIKey      string  `yaml:"api_key"`
    Model       string  `yaml:"model"`
    Temperature float32 `yaml:"temperature"`
    MaxTokens   int     `yaml:"max_tokens"`
}

type AgentConfig struct {
    Type              string `yaml:"type"`
    MaxTurns          int    `yaml:"max_turns"`
    EnableParallel    bool   `yaml:"enable_parallel_tools"`
    EnableSubAgents   bool   `yaml:"enable_sub_agents"`
    ContextWindow     int    `yaml:"context_window"`
    SummarizeAfter    int    `yaml:"summarize_after"`
}

// ... 其他配置结构
```

#### 7.2 配置加载器

**文件**: `pkg/config/loader.go`

```go
package config

import (
    "os"
    "gopkg.in/yaml.v3"
)

func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }

    // 环境变量替换
    cfg = expandEnvVars(cfg)

    return &cfg, nil
}

func expandEnvVars(cfg Config) Config {
    // 替换 ${ENV_VAR} 形式的环境变量
}
```

#### 7.3 默认配置

**文件**: `configs/default.yaml`

```yaml
llm:
  provider: openai
  api_key: ${OPENAI_API_KEY}
  model: gpt-4-turbo
  temperature: 0.7
  max_tokens: 4096

agent:
  type: general
  max_turns: 20
  enable_parallel_tools: true
  enable_sub_agents: true
  context_window: 128000
  summarize_after: 50

session:
  persistence: sqlite
  db_path: ~/.finta/sessions.db
  auto_save: true

tools:
  builtin:
    - bash
    - read
    - write
    - glob
    - grep

mcp:
  servers: []

hooks: []

cli:
  markdown: true
  streaming: true
  theme: dark
```

### Phase 7 完成标准

- ✅ 完整的配置结构
- ✅ YAML 配置加载
- ✅ 环境变量支持
- ✅ 默认配置文件
- ✅ CLI 支持 `--config` 参数

---

## Phase 8: 文档和完善 (2-3 天)

### 目标

完善文档、示例和测试，确保框架可用性。

### 实现步骤

#### 8.1 架构文档

**文件**: `docs/architecture.md`

详细说明：

- 整体架构
- 核心组件
- 数据流
- 扩展点

#### 8.2 开发指南

**文件**: `docs/development.md`

包含：

- 如何添加自定义工具
- 如何创建专门化 Agent
- 如何编写 Hook
- 如何集成 MCP 服务器

#### 8.3 示例项目

**文件**: `examples/simple_agent/main.go`

```go
package main

import (
    "context"
    "fmt"
    "os"

    "finta/internal/agent"
    "finta/internal/llm/openai"
    "finta/internal/tool"
    "finta/internal/tool/builtin"
)

func main() {
    // 创建 LLM 客户端
    client := openai.NewClient(os.Getenv("OPENAI_API_KEY"), "gpt-4-turbo")

    // 创建工具注册表
    registry := tool.NewRegistry()
    registry.Register(builtin.NewReadTool())
    registry.Register(builtin.NewBashTool())

    // 创建 Agent
    ag := agent.NewBaseAgent(
        "my-agent",
        "You are a helpful assistant",
        client,
        registry,
        nil,
    )

    // 运行
    output, err := ag.Run(context.Background(), &agent.Input{
        Task: "List files in current directory",
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(output.Result)
}
```

#### 8.4 README 更新

**文件**: `README.md`

包含：

- 项目介绍
- 快速开始
- 核心特性
- 安装说明
- 基础用法
- 配置说明
- 贡献指南

#### 8.5 单元测试

为核心组件添加测试：

- `pkg/tool/registry_test.go`
- `pkg/agent/base_test.go`
- `pkg/llm/openai/client_test.go`

### Phase 8 完成标准

- ✅ 完整的架构文档
- ✅ 开发指南和教程
- ✅ 至少 3 个示例项目
- ✅ README 更新
- ✅ 核心组件有单元测试
- ✅ 代码有适当的注释

---

## 实现优先级建议

### 必须立即实现（MVP）

**Phase 1**: 核心基础

- 这是框架能运行的最小基础

### 重要但可以分步实现

**Phase 2**: 高级工具系统
**Phase 3**: 专门化 Agent

- 这两个阶段让框架更加强大和实用

### 可以后续添加的功能

**Phase 4**: MCP 集成
**Phase 5**: Hook 系统
**Phase 6**: Session 管理

- 这些功能增强了框架的可扩展性和易用性

### 最后完善

**Phase 7**: 配置系统
**Phase 8**: 文档和完善

- 让框架更加专业和易于使用

---

## 关键技术决策

### 1. 为什么选择 Interface-based 设计？

- **优点**: 最大化扩展性，便于测试
- **缺点**: 代码略显冗长
- **决策**: 接受冗长换取灵活性

### 2. 为什么使用 OpenAI 作为主要 LLM？

- **优点**: API 成熟，工具调用支持好
- **缺点**: 依赖外部服务
- **决策**: 通过接口抽象，后续可轻松切换

### 3. 工具并行执行的复杂度如何处理？

- **方案**: 启发式依赖分析 + 拓扑排序
- **权衡**: 不追求完美的依赖检测，优先保证正确性

### 4. MCP 集成的边界在哪里？

- **决策**: 支持核心协议（工具、资源、提示）
- **暂不支持**: 采样等高级特性
- **理由**: 先保证基础功能可用

### 5. Session 持久化为什么用 SQLite？

- **优点**: 零配置，ACID 保证
- **缺点**: 不适合分布式
- **决策**: 针对本地 CLI 场景优化

---

## 开发时间估算

| 阶段                  | 时间   | 累计  |
| --------------------- | ------ | ----- |
| Phase 1: 核心基础     | 2-3 天 | 3 天  |
| Phase 2: 高级工具     | 2-3 天 | 6 天  |
| Phase 3: 专门化 Agent | 2-3 天 | 9 天  |
| Phase 4: MCP 集成     | 3-4 天 | 13 天 |
| Phase 5: Hook 系统    | 2 天   | 15 天 |
| Phase 6: Session 管理 | 2 天   | 17 天 |
| Phase 7: 配置系统     | 1-2 天 | 19 天 |
| Phase 8: 文档完善     | 2-3 天 | 22 天 |

**总计**: 约 3-4 周（全职开发）

---

## 下一步行动

1. **立即开始 Phase 1**

   - 创建目录结构
   - 实现核心接口
   - 完成 OpenAI 集成
   - 实现基础工具
   - 创建简单 CLI

2. **验证 Phase 1**

   - 能够运行一个简单的 agent
   - Agent 可以调用工具
   - 工具可以执行并返回结果

3. **迭代开发**

   - 完成一个 Phase 后测试验证
   - 及时调整设计
   - 保持代码质量

4. **持续集成**
   - 每个 Phase 完成后都要确保整体可用
   - 编写测试防止回归
   - 更新文档保持同步
