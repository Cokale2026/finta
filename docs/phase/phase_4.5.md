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
