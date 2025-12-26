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
