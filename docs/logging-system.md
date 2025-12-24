# Finta 日志和输出展示系统

## 概述

Finta 的日志系统是 Phase 1 的核心组件之一，旨在让你清楚地了解 Agent 的执行过程。通过分层日志、彩色输出和结构化展示，你可以实时看到：

- Agent 在做什么
- 调用了哪些工具
- 工具的参数和执行结果
- 每个操作的耗时
- 整个会话的统计信息

## 核心特性

### 1. 分级日志系统

```go
type Level int

const (
    LevelDebug Level = iota  // 调试信息
    LevelInfo                // 一般信息
    LevelTool                // 工具调用相关
    LevelAgent               // Agent 思考和响应
    LevelError               // 错误信息
)
```

### 2. 彩色输出

- 🟦 **蓝色**: 一般信息 (INFO)
- 🟩 **绿色**: 成功状态和 Agent 响应
- 🟥 **红色**: 错误信息
- 🟨 **黄色**: 警告信息（未来支持）
- 🟪 **紫色**: Agent 思考过程
- 🟦 **青色**: 工具调用
- ⬜ **灰色**: 调试信息

### 3. 结构化输出

所有重要信息都通过分隔线和标题进行结构化展示：

```
🔧 Tool Call: bash
────────────────────────────────────────────────────────
{ "command": "ls -la" }
────────────────────────────────────────────────────────
```

### 4. 会话统计

每个会话都有明确的开始和结束标记，并展示统计信息：

```
══════════════════════════════════════════════════════════════════════
🚀 Session Started
  Your task description here
══════════════════════════════════════════════════════════════════════

... 执行过程 ...

══════════════════════════════════════════════════════════════════════
✨ Session Completed
  Duration: 1.234s | Tool Calls: 3
══════════════════════════════════════════════════════════════════════
```

## 使用方法

### CLI 选项

```bash
# 普通模式（显示重要信息）
./finta chat "Your task"

# 详细模式（显示所有调试信息）
./finta chat --verbose "Your task"

# 无颜色模式（适合日志文件或不支持颜色的终端）
./finta chat --no-color "Your task"

# 组合使用
./finta chat --verbose --no-color "Your task" > agent.log
```

### 在代码中使用

```go
import "finta/internal/logger"

// 创建 logger
log := logger.NewLogger(os.Stdout, logger.LevelInfo)

// 基础日志
log.Debug("Debug message: %v", data)
log.Info("Info message: %s", info)
log.Error("Error: %v", err)

// Agent 专用日志
log.AgentThinking("Let me analyze this problem...")
log.AgentResponse("Here's my answer...")

// 工具调用日志
log.ToolCall("bash", `{"command": "ls"}`)
log.ToolResult("bash", true, "output here", 100*time.Millisecond)

// 会话管理
log.SessionStart("Task description")
log.SessionEnd(duration, toolCallCount)

// 进度条
for i := 0; i < total; i++ {
    log.Progress(i+1, total, "Processing...")
}
```

## 日志输出示例

### 完整的执行流程

```
══════════════════════════════════════════════════════════════════════
🚀 Session Started
  Read the go.mod file and tell me what dependencies we have
══════════════════════════════════════════════════════════════════════

15:30:45 [INFO] Registered 2 tools: read, bash
15:30:45 [DEBUG] Agent created with max_turns=10, temperature=0.70
15:30:45 [INFO] Turn 1: Calling LLM...

🔧 Tool Call: read
────────────────────────────────────────────────────────
{
  "file_path": "go.mod"
}
────────────────────────────────────────────────────────

📊 Tool Result: read [✅ Success] (15ms)
────────────────────────────────────────────────────────
module finta

go 1.24.5

require (
    github.com/sashabaranov/go-openai v1.35.6
    github.com/spf13/cobra v1.8.1
    github.com/charmbracelet/glamour v0.8.0
    gopkg.in/yaml.v3 v3.0.1
)
────────────────────────────────────────────────────────

15:30:46 [INFO] Turn 2: Calling LLM...

💬 Agent Response
────────────────────────────────────────────────────────
Based on the go.mod file, this project has the following dependencies:

1. **github.com/sashabaranov/go-openai** (v1.35.6)
   - OpenAI API client for Go

2. **github.com/spf13/cobra** (v1.8.1)
   - CLI framework

3. **github.com/charmbracelet/glamour** (v0.8.0)
   - Markdown rendering library

4. **gopkg.in/yaml.v3** (v3.0.1)
   - YAML parsing library

The project is using Go 1.24.5.
────────────────────────────────────────────────────────

══════════════════════════════════════════════════════════════════════
✨ Session Completed
  Duration: 1.523s | Tool Calls: 1
══════════════════════════════════════════════════════════════════════

15:30:46 [DEBUG] Agent completed successfully
```

## 关键方法说明

### Logger 核心方法

| 方法 | 用途 | 示例 |
|------|------|------|
| `Debug()` | 调试信息 | 只在 verbose 模式显示 |
| `Info()` | 一般信息 | 显示重要的执行步骤 |
| `Error()` | 错误信息 | 总是显示 |
| `AgentThinking()` | Agent 思考过程 | 展示 Agent 的推理 |
| `AgentResponse()` | Agent 回复 | 展示最终答案 |
| `ToolCall()` | 工具调用 | 显示工具名和参数 |
| `ToolResult()` | 工具结果 | 显示执行结果和耗时 |
| `SessionStart()` | 会话开始 | 显示任务描述 |
| `SessionEnd()` | 会话结束 | 显示统计信息 |
| `Progress()` | 进度条 | 显示执行进度 |

### ExecutionContext 方法

ExecutionContext 是对 Logger 的封装，用于在 Agent 执行过程中记录上下文信息：

```go
type ExecutionContext struct {
    Logger        *logger.Logger
    StartTime     time.Time
    CurrentTurn   int
    TotalTurns    int
    ToolCallCount int
}

// 使用方法
execCtx := NewExecutionContext(log)
execCtx.LogToolCall("bash", params)
execCtx.LogToolResult("bash", true, output, duration)
execCtx.LogProgress()
```

## 最佳实践

### 1. 选择合适的日志级别

- **开发阶段**: 使用 `--verbose` 查看所有细节
- **生产环境**: 使用默认级别，只显示重要信息
- **调试问题**: 使用 `--verbose --no-color > debug.log` 保存详细日志

### 2. 结构化输出

所有工具调用都应该展示：
- 工具名称
- 输入参数（JSON 格式化）
- 执行结果
- 执行时间

### 3. 错误处理

错误信息要清晰明确：
```go
if err != nil {
    log.Error("Failed to execute tool %s: %v", toolName, err)
    return nil, err
}
```

### 4. 进度展示

对于长时间运行的任务，使用进度条：
```go
for i, item := range items {
    log.Progress(i+1, len(items), fmt.Sprintf("Processing %s", item))
    // 处理 item
}
```

## 配置选项

### 环境变量支持（未来）

```bash
export FINTA_LOG_LEVEL=debug
export FINTA_LOG_COLOR=true
export FINTA_LOG_TIMESTAMP=true
```

### 配置文件支持（未来）

```yaml
logging:
  level: info
  color: true
  timestamp: true
  format: text  # 或 json
  output: stdout
```

## 与其他组件集成

### Agent 集成

Agent 在执行过程中自动记录：
- 每一轮的开始
- LLM 调用
- 工具调用和结果
- 最终响应

### Tool 集成

所有工具执行都会自动记录：
- 调用时间
- 参数
- 结果
- 执行时长
- 成功/失败状态

### Hook 集成（Phase 5）

Hook 系统可以监听日志事件并采取行动：
```go
type LogHook struct {
    // 当发生错误时触发
    OnError func(error)
}
```

## 性能考虑

### 日志开销

- 普通模式：日志开销 < 1% 执行时间
- Verbose 模式：日志开销 < 5% 执行时间
- 无颜色模式：略微更快

### 优化建议

1. 对于长输出，自动截断（例如 > 10000 字符）
2. 使用缓冲写入减少系统调用
3. 异步日志写入（可选）

## 扩展性

日志系统设计为可扩展的，未来可以：

1. **支持多种输出格式**
   - JSON 格式（机器可读）
   - 纯文本（人类可读）
   - 结构化日志（如 logfmt）

2. **支持多个输出目标**
   - 标准输出
   - 文件
   - 远程日志服务（如 Logstash）

3. **支持日志过滤**
   - 按组件过滤
   - 按关键字过滤
   - 按时间过滤

4. **支持日志聚合**
   - 统计信息
   - 性能分析
   - 错误聚合

## 总结

Finta 的日志系统让你能够：

✅ **实时了解 Agent 在做什么**
✅ **追踪每个工具的调用和结果**
✅ **调试问题时有完整的执行记录**
✅ **获得执行性能的洞察**
✅ **在不同场景下选择合适的日志级别**

这个系统是 Finta 框架透明度和可观察性的基础，让你在开发和调试 AI Agent 时更加高效。
