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
