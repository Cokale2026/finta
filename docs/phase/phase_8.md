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
