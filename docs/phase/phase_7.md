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
