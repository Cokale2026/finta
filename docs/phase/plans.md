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
