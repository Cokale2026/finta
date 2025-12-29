# Finta

[![Go Version](https://img.shields.io/badge/Go-1.24.5-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

[English](README.md) | 中文

一个灵活的 AI Agent 框架，设计理念受 Claude Code 启发。Finta 提供了模块化、可扩展的基础架构，用于构建能够执行工具、与 LLM 交互并处理复杂多轮对话的 AI 代理。

## 特性

- **接口驱动架构** - 清晰的关注点分离，组件可插拔
- **专业化代理** - 四种针对不同任务优化的代理类型（通用、探索、计划、执行）
- **层级代理组合** - 可生成子代理进行复杂任务委托
- **内置工具** - Read、Write、Bash、Glob、Grep、TodoWrite 和 Task 工具
- **MCP 集成** - 通过 Model Context Protocol 服务器扩展能力
- **Hook 系统** - 执行潜在危险操作前的用户确认机制
- **并行工具执行** - 智能依赖分析实现并发工具调用
- **流式输出** - 实时响应流式传输，支持 Markdown 渲染
- **推理支持** - 扩展思维/推理过程可视化

## 安装

```bash
# 克隆仓库
git clone https://github.com/cokale/finta.git
cd finta

# 构建二进制文件
go build -o finta cmd/finta/main.go
```

## 快速开始

```bash
# 设置 OpenAI API 密钥
export OPENAI_API_KEY="your-api-key"

# 启动交互式对话
./finta chat

# 使用自定义模型
./finta chat --model gpt-4o

# 使用专业化代理
./finta chat --agent-type explore
```

## 使用方法

### CLI 参数

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `--api-key` | OpenAI API 密钥 | `$OPENAI_API_KEY` |
| `--api-base-url` | 自定义 API 端点 | `$OPENAI_API_BASE_URL` |
| `--model` | 使用的模型 | `gpt-4-turbo` |
| `--agent-type` | 代理类型 (general, explore, plan, execute) | `general` |
| `--temperature` | 温度参数 | `0.7` |
| `--max-turns` | 最大对话轮数 | `10` |
| `--verbose` | 启用调试日志 | `false` |
| `--streaming` | 启用流式输出 | `false` |
| `--parallel` | 启用并行工具执行 | `true` |
| `--config` | 配置文件路径 | 自动检测 |

### 代理类型

| 类型 | 描述 | 工具 | 温度 |
|------|------|------|------|
| **General** | 通用代理 | 所有工具 | 0.7 |
| **Explore** | 代码探索和搜索 | 只读工具 | 0.3 |
| **Plan** | 实现规划 | Read + Glob | 0.5 |
| **Execute** | 任务执行 | 所有工具 | 0.5 |

```bash
# 探索代码库
./finta chat --agent-type explore
> 找出所有处理 HTTP 请求的 Go 文件

# 规划实现
./finta chat --agent-type plan
> 规划如何添加用户认证功能
```

## 内置工具

| 工具 | 描述 |
|------|------|
| `read` | 读取文件，支持可选行范围（最多 8 个文件） |
| `write` | 创建或覆盖文件 |
| `bash` | 执行 shell 命令，支持超时 |
| `glob` | 查找匹配模式的文件（支持 `**` 递归） |
| `grep` | 使用正则表达式搜索文件内容 |
| `task` | 生成子代理进行任务委托 |
| `TodoWrite` | 跟踪多步骤任务的进度 |

## 配置

Finta 按以下顺序查找配置文件：
1. `./finta.yaml`
2. `./configs/finta.yaml`
3. `~/.config/finta/finta.yaml`
4. `/etc/finta/finta.yaml`

### 配置示例

```yaml
# MCP 服务器配置
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

# Hook 配置
hooks:
  bash_confirm: true
  tool_confirm:
    - write
    - bash
```

## MCP 集成

Finta 支持 [Model Context Protocol](https://modelcontextprotocol.io/) 服务器以扩展工具能力。

```bash
# 安装 MCP 服务器
npm install -g @modelcontextprotocol/server-filesystem
npm install -g @modelcontextprotocol/server-github

# 设置环境变量
export GITHUB_TOKEN=your_token

# 使用 MCP 运行
./finta chat --config configs/finta.yaml
```

MCP 工具以 `{服务器}_{工具}` 格式命名（例如 `filesystem_read_file`、`github_create_issue`）。

## Hook 系统

Hook 允许在执行潜在危险操作前进行用户确认：

- **bash_confirm** - 执行 shell 命令前确认
- **tool_confirm** - 执行特定工具前确认

触发 Hook 时，系统会提示您允许或拒绝该操作。

## 架构

```
┌─────────────────────────────────────────────────────────────┐
│                         CLI (main.go)                        │
├─────────────────────────────────────────────────────────────┤
│                      代理工厂 (Agent Factory)                │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐           │
│  │  通用   │ │  探索   │ │  计划   │ │  执行   │           │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘           │
├─────────────────────────────────────────────────────────────┤
│                      基础代理 (Base Agent)                   │
│         运行循环: LLM 调用 → 工具执行 → 重复                 │
├──────────────────────┬──────────────────────────────────────┤
│     工具注册表       │           LLM 客户端                  │
│  ┌────────────────┐  │  ┌────────────────────────────────┐  │
│  │  内置工具      │  │  │  OpenAI API                    │  │
│  │  + MCP 工具    │  │  │  （支持推理功能）               │  │
│  └────────────────┘  │  └────────────────────────────────┘  │
├──────────────────────┴──────────────────────────────────────┤
│                      Hook 管理器                             │
│              用户确认 & 反馈                                 │
└─────────────────────────────────────────────────────────────┘
```

## 项目结构

```
finta/
├── cmd/finta/          # CLI 入口
├── internal/
│   ├── agent/          # 代理实现和工厂
│   ├── config/         # 配置解析
│   ├── hook/           # Hook 系统
│   ├── llm/            # LLM 客户端接口和 OpenAI 实现
│   ├── logger/         # 结构化日志，支持 Markdown 渲染
│   ├── mcp/            # MCP 集成
│   └── tool/           # 工具接口、注册表和内置工具
├── configs/            # 示例配置文件
└── docs/               # 文档
```

## 贡献

欢迎贡献！请随时提交 Issue 和 Pull Request。

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启 Pull Request

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。
