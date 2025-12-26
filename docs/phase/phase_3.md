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
