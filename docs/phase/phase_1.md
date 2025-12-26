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
