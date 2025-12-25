● 完美！我已经完成了探索阶段，对 Phase 3 的需求和当前架构有了深入理解。现在让我先澄清几个问题，然后进入设计阶段。

● User answered Claude's questions:
⎿  · Execute Agent 是否需要在 Phase 3 实现？plans.md 中提到了它但没有详细说明。 →
是，实现 Execute Agent（需要定义其工具集和系统提示）
· Task 工具的嵌套深度限制应该是多少？ → 最大深度 3 层（推荐）
· 子 Agent 的日志应该如何处理？ → 合并到父 Agent（推荐）
· Task 工具是否应该支持自定义 Agent 参数（temperature、model 等）？ → 只支持
max_turns（推荐）

● 完美！现在我了解了所有需求。让我进入设计阶段，启动 Plan agent 来设计详细的实现方案。

● Plan(Design Phase 3 implementation)
⎿  Done (8 tool uses · 0 tokens · 14s)

● 我遇到了 agent 调用限制，但没关系，我已经从探索阶段获得了足够的信息。让我直接基于探索结果和你的反馈创建详细的实现计划。

Current Plan
/home/voker/.claude/plans/curried-dreaming-boot.md

     # Phase 3: Specialized Agents - Implementation Plan

     ## Overview

     Implement a multi-agent system with 4 specialized agent types (General, Explore,
      Plan, Execute), a factory pattern for agent creation, and a Task tool for agent
      nesting with maximum 3-level depth.

     ## Architecture Summary

     ```
     Factory Pattern:
       AgentFactory → CreateAgent(type) → {Explore, Plan, Execute, General}Agent
                                                 ↓
                                           Each with specific:
                                           - Tool subset
                                           - System prompt
                                           - Temperature
                                           - MaxTurns

     Task Tool:
       Agent A → Task("explore", "find files") → Explore Agent
                                                      ↓
                                                Shared Logger
                                                Depth tracking
                                                Result formatting
     ```

     ## Implementation Steps

     ### Step 1: Create Agent Types System

     **File**: `/home/voker/data/code/finta/internal/agent/types.go` (NEW)

     **Content**:
     ```go
     package agent

     import (
         "fmt"
         "finta/internal/llm"
         "finta/internal/tool"
         "finta/internal/tool/builtin"
     )

     // AgentType defines the type of specialized agent
     type AgentType string

     const (
         AgentTypeGeneral AgentType = "general"
         AgentTypeExplore AgentType = "explore"
         AgentTypePlan    AgentType = "plan"
         AgentTypeExecute AgentType = "execute"
     )

     // Factory creates agents of different types
     type Factory interface {
         CreateAgent(agentType AgentType) (Agent, error)
     }

     // DefaultFactory is the standard agent factory
     type DefaultFactory struct {
         llmClient    llm.Client
         toolRegistry *tool.Registry
     }

     func NewDefaultFactory(client llm.Client, registry *tool.Registry)
     *DefaultFactory {
         return &DefaultFactory{
             llmClient:    client,
             toolRegistry: registry,
         }
     }

     func (f *DefaultFactory) CreateAgent(agentType AgentType) (Agent, error) {
         switch agentType {
         case AgentTypeGeneral:
             return f.createGeneralAgent(), nil
         case AgentTypeExplore:
             return f.createExploreAgent(), nil
         case AgentTypePlan:
             return f.createPlanAgent(), nil
         case AgentTypeExecute:
             return f.createExecuteAgent(), nil
         default:
             return nil, fmt.Errorf("unknown agent type: %s", agentType)
         }
     }

     // Agent-specific creation methods (implemented in Step 2)
     ```

     **Key Points**:
     - Simple factory interface
     - DefaultFactory holds shared LLM client and full tool registry
     - Each createXXXAgent() method returns a BaseAgent with specific config
     - Error handling for invalid agent types

     ---

     ### Step 2: Implement Specialized Agents

     #### 2.1 General Agent

     **File**: `/home/voker/data/code/finta/internal/agent/types.go` (add method)

     ```go
     func (f *DefaultFactory) createGeneralAgent() Agent {
         // Use full tool registry
         return NewBaseAgent(
             "general",
             `You are a helpful AI assistant with access to tools.
     You can read files, execute bash commands, write files, find files with glob
     patterns, and search files with grep.
     Always provide clear, concise responses.`,
             f.llmClient,
             f.toolRegistry, // All tools available
             &Config{
                 Model:              "gpt-4-turbo",
                 Temperature:        0.7,
                 MaxTokens:          4096,
                 MaxTurns:           20,
                 EnableParallelTools: true,
                 ToolExecutionMode:   tool.ExecutionModeMixed,
             },
         )
     }
     ```

     #### 2.2 Explore Agent

     **File**: `/home/voker/data/code/finta/internal/agent/types.go` (add method)

     ```go
     func (f *DefaultFactory) createExploreAgent() Agent {
         // Create filtered registry with read-only tools
         exploreRegistry := tool.NewRegistry()

         // Get tools from main registry
         if readTool, err := f.toolRegistry.Get("read"); err == nil {
             exploreRegistry.Register(readTool)
         }
         if globTool, err := f.toolRegistry.Get("glob"); err == nil {
             exploreRegistry.Register(globTool)
         }
         if grepTool, err := f.toolRegistry.Get("grep"); err == nil {
             exploreRegistry.Register(grepTool)
         }
         if bashTool, err := f.toolRegistry.Get("bash"); err == nil {
             exploreRegistry.Register(bashTool)
         }

         systemPrompt := `You are an expert codebase exploration agent.

     Your goal is to efficiently explore and understand codebases. You have access to
      read-only tools:
     - read: Read file contents
     - glob: Find files matching patterns
     - grep: Search for content in files
     - bash: Execute read-only commands (ls, find, cat, etc.)

     Best practices:
     1. Start with glob to find relevant files
     2. Use grep to search for specific patterns
     3. Read files to understand implementation details
     4. Be thorough but efficient
     5. Use bash for directory listings and simple queries

     Always provide clear summaries of your findings.`

         return NewBaseAgent(
             "explore",
             systemPrompt,
             f.llmClient,
             exploreRegistry,
             &Config{
                 Model:              "gpt-4-turbo",
                 Temperature:        0.3, // Lower for focused exploration
                 MaxTokens:          4096,
                 MaxTurns:           15,
                 EnableParallelTools: true,
                 ToolExecutionMode:   tool.ExecutionModeMixed,
             },
         )
     }
     ```

     #### 2.3 Plan Agent

     **File**: `/home/voker/data/code/finta/internal/agent/types.go` (add method)

     ```go
     func (f *DefaultFactory) createPlanAgent() Agent {
         // Create filtered registry with read and glob only
         planRegistry := tool.NewRegistry()

         if readTool, err := f.toolRegistry.Get("read"); err == nil {
             planRegistry.Register(readTool)
         }
         if globTool, err := f.toolRegistry.Get("glob"); err == nil {
             planRegistry.Register(globTool)
         }

         systemPrompt := `You are an expert software architect and planning agent.

     Your goal is to create detailed, actionable implementation plans. You can read
     files to understand the current codebase state.

     When creating plans:
     1. Break down tasks into clear steps
     2. Identify critical files to be modified
     3. Consider architectural trade-offs
     4. Suggest best practices
     5. Anticipate potential issues

     Output your plan in a structured markdown format with:
     - **Overview**: High-level summary
     - **Implementation Steps**: Numbered, actionable steps
     - **Files to Modify**: List with descriptions
     - **Testing Strategy**: How to verify the implementation
     - **Potential Risks**: Issues to watch out for

     Be thorough and consider edge cases.`

         return NewBaseAgent(
             "plan",
             systemPrompt,
             f.llmClient,
             planRegistry,
             &Config{
                 Model:              "gpt-4-turbo",
                 Temperature:        0.5, // Balanced
                 MaxTokens:          4096,
                 MaxTurns:           10,
                 EnableParallelTools: true,
                 ToolExecutionMode:   tool.ExecutionModeMixed,
             },
         )
     }
     ```

     #### 2.4 Execute Agent

     **File**: `/home/voker/data/code/finta/internal/agent/types.go` (add method)

     ```go
     func (f *DefaultFactory) createExecuteAgent() Agent {
         // Use full tool registry
         systemPrompt := `You are an expert implementation agent focused on careful,
     precise code execution.

     Your goal is to implement changes accurately and safely. You have access to all
     tools:
     - read: Read file contents
     - write: Create or overwrite files
     - bash: Execute bash commands
     - glob: Find files matching patterns
     - grep: Search for content in files

     Best practices:
     1. Always read files before modifying them
     2. Make incremental changes and verify
     3. Use glob/grep to understand context
     4. Test changes when possible
     5. Be careful with destructive operations
     6. Explain what you're doing and why

     Focus on correctness and safety over speed.`

         return NewBaseAgent(
             "execute",
             systemPrompt,
             f.llmClient,
             f.toolRegistry, // All tools available
             &Config{
                 Model:              "gpt-4-turbo",
                 Temperature:        0.5,
                 MaxTokens:          4096,
                 MaxTurns:           20,
                 EnableParallelTools: true,
                 ToolExecutionMode:   tool.ExecutionModeMixed,
             },
         )
     }
     ```

     ---

     ### Step 3: Implement Task Tool with Nesting Control

     **File**: `/home/voker/data/code/finta/internal/tool/builtin/task.go` (NEW)

     **Content**:
     ```go
     package builtin

     import (
         "context"
         "encoding/json"
         "fmt"

         "finta/internal/agent"
         "finta/internal/logger"
         "finta/internal/tool"
     )

     // Context key for nesting depth tracking
     type contextKey string
     const nestingDepthKey contextKey = "nesting_depth"

     // MaxNestingDepth is the maximum allowed depth for sub-agent calls
     const MaxNestingDepth = 3

     type TaskTool struct {
         factory agent.Factory
     }

     func NewTaskTool(factory agent.Factory) *TaskTool {
         return &TaskTool{
             factory: factory,
         }
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
                     "description": "Type of agent to spawn (explore, plan,
     execute)",
                     "enum":        []string{"explore", "plan", "execute"},
                 },
                 "task": map[string]any{
                     "type":        "string",
                     "description": "Task description for the sub-agent",
                 },
                 "description": map[string]any{
                     "type":        "string",
                     "description": "Short description of what this sub-agent will do
      (3-5 words)",
                 },
                 "max_turns": map[string]any{
                     "type":        "number",
                     "description": "Maximum turns for sub-agent (optional, defaults
     to agent type default)",
                 },
             },
             "required": []string{"agent_type", "task", "description"},
         }
     }

     func (t *TaskTool) Execute(ctx context.Context, params json.RawMessage)
     (*tool.Result, error) {
         var p struct {
             AgentType   string `json:"agent_type"`
             Task        string `json:"task"`
             Description string `json:"description"`
             MaxTurns    int    `json:"max_turns"`
         }

         if err := json.Unmarshal(params, &p); err != nil {
             return &tool.Result{
                 Success: false,
                 Error:   fmt.Sprintf("invalid parameters: %v", err),
             }, nil
         }

         // Check nesting depth
         depth := getNestingDepth(ctx)
         if depth >= MaxNestingDepth {
             return &tool.Result{
                 Success: false,
                 Error:   fmt.Sprintf("maximum nesting depth (%d) exceeded",
     MaxNestingDepth),
             }, nil
         }

         // Create sub-agent
         subAgent, err := t.factory.CreateAgent(agent.AgentType(p.AgentType))
         if err != nil {
             return &tool.Result{
                 Success: false,
                 Error:   fmt.Sprintf("failed to create agent: %v", err),
             }, nil
         }

         // Get parent logger from context (if available)
         // NOTE: Parent agent should store logger in context
         parentLogger := getLoggerFromContext(ctx)
         if parentLogger == nil {
             // Fallback: create a basic logger
             parentLogger = logger.NewLogger(nil, logger.LevelInfo)
         }

         // Log sub-agent start
         parentLogger.Info("Launching %s sub-agent: %s", p.AgentType, p.Description)

         // Create context with incremented depth
         subCtx := context.WithValue(ctx, nestingDepthKey, depth+1)

         // Run sub-agent
         output, err := subAgent.Run(subCtx, &agent.Input{
             Task:        p.Task,
             MaxTurns:    p.MaxTurns, // Use provided or default (0 = agent default)
             Temperature: 0,          // Use agent default
             Logger:      parentLogger, // Share parent logger
         })
         if err != nil {
             return &tool.Result{
                 Success: false,
                 Error:   fmt.Sprintf("sub-agent failed: %v", err),
             }, nil
         }

         // Log sub-agent completion
         parentLogger.Info("Sub-agent completed: %s", p.Description)

         // Format result
         resultText := fmt.Sprintf("[%s agent: %s]\n\n%s",
             p.AgentType, p.Description, output.Result)

         return &tool.Result{
             Success: true,
             Output:  resultText,
             Data: map[string]any{
                 "agent_type": p.AgentType,
                 "tool_calls": len(output.ToolCalls),
                 "turns":      len(output.Messages),
             },
         }, nil
     }

     // Helper functions
     func getNestingDepth(ctx context.Context) int {
         if depth, ok := ctx.Value(nestingDepthKey).(int); ok {
             return depth
         }
         return 0
     }

     func getLoggerFromContext(ctx context.Context) *logger.Logger {
         // This requires adding logger to context in agent/base.go
         // For now, return nil and let caller create default
         return nil
     }
     ```

     **Key Design Points**:
     - Uses context to track nesting depth (prevents infinite recursion)
     - Shares parent logger (merged logging)
     - Supports optional max_turns parameter
     - Returns structured result with agent identification
     - Limits to 3 levels of nesting

     ---

     ### Step 4: Update Agent to Store Logger in Context

     **File**: `/home/voker/data/code/finta/internal/agent/base.go` (MODIFY)

     **Changes needed**:

     1. Add context key at package level:
     ```go
     type contextKey string
     const loggerContextKey contextKey = "logger"
     ```

     2. In `Run()` method, add logger to context after line 43:
     ```go
     // Create execution context
     execCtx := NewExecutionContext(input.Logger)

     // Add logger to context for sub-agents (NEW)
     ctx = context.WithValue(ctx, loggerContextKey, input.Logger)

     // Log session start
     execCtx.Logger.SessionStart(input.Task)
     ```

     3. In `RunStreaming()` method, add same modification after line 202:
     ```go
     // Create execution context
     execCtx := NewExecutionContext(input.Logger)

     // Add logger to context for sub-agents (NEW)
     ctx = context.WithValue(ctx, loggerContextKey, input.Logger)

     // Log session start
     execCtx.Logger.SessionStart(input.Task)
     ```

     4. Add helper function to retrieve logger from context:
     ```go
     // GetLoggerFromContext retrieves the logger stored in context
     func GetLoggerFromContext(ctx context.Context) *logger.Logger {
         if log, ok := ctx.Value(loggerContextKey).(*logger.Logger); ok {
             return log
         }
         return nil
     }
     ```

     5. Update task.go to use this helper:
     ```go
     func getLoggerFromContext(ctx context.Context) *logger.Logger {
         return agent.GetLoggerFromContext(ctx)
     }
     ```

     **Why**: Allows Task tool to retrieve and share parent's logger with sub-agents.

     ---

     ### Step 5: Update CLI to Support Specialized Agents

     **File**: `/home/voker/data/code/finta/cmd/finta/main.go` (MODIFY)

     **Changes**:

     1. Add new flag for agent type:
     ```go
     var (
         // ... existing flags ...
         agentType   string  // NEW
     )
     ```

     2. Register flag:
     ```go
     chatCmd.Flags().StringVar(&agentType, "agent-type", "general",
         "Agent type to use (general, explore, plan, execute)")
     ```

     3. Update agent creation (replace lines 93-112):
     ```go
     // Create agent factory
     factory := agent.NewDefaultFactory(llmClient, registry)

     // Register Task tool with factory
     taskTool := builtin.NewTaskTool(factory)
     registry.Register(taskTool)

     log.Info("Registered %d tools: read, bash, write, glob, grep, task", 6)

     // Create agent based on type
     var ag agent.Agent
     var err error
     ag, err = factory.CreateAgent(agent.AgentType(agentType))
     if err != nil {
         log.Error("Failed to create agent: %v", err)
         return err
     }

     log.Debug("Created %s agent", agentType)
     ```

     **Note**: Task tool is registered AFTER factory creation but BEFORE any agent
     uses it.

     ---

     ### Step 6: Update Tool Count

     **File**: `/home/voker/data/code/finta/cmd/finta/main.go` (MODIFY)

     Change line 91 from:
     ```go
     log.Info("Registered %d tools: read, bash, write, glob, grep", 5)
     ```

     To:
     ```go
     log.Info("Registered %d tools: read, bash, write, glob, grep, task", 6)
     ```

     ---

     ## File Summary

     ### New Files (2):
     1. `/home/voker/data/code/finta/internal/agent/types.go` - Agent types, factory
     2. `/home/voker/data/code/finta/internal/tool/builtin/task.go` - Task tool

     ### Modified Files (2):
     1. `/home/voker/data/code/finta/internal/agent/base.go` - Add logger to context
     2. `/home/voker/data/code/finta/cmd/finta/main.go` - Add factory, task tool,
     agent-type flag

     ---

     ## Testing Strategy

     ### Unit Testing

     1. **Factory Tests**:
        - Test CreateAgent() for each type
        - Verify error on invalid type
        - Check that each agent has correct config

     2. **Task Tool Tests**:
        - Test depth limiting (should fail at depth 4)
        - Test parameter validation
        - Test agent spawning and result formatting
        - Test logger sharing

     3. **Agent Type Tests**:
        - Verify each agent has correct tool subset
        - Test that Explore can't write
        - Test that Plan can't bash
        - Test that Execute has all tools

     ### Integration Testing

     1. **Simple Task Flow**:
        ```bash
        ./finta chat "Use the task tool to explore internal/agent/"
        ```
        Expected: General agent spawns Explore sub-agent

     2. **Nested Tasks**:
        ```bash
        ./finta chat "Plan an implementation, then explore the code to understand
     current patterns"
        ```
        Expected: Multiple sub-agent calls work correctly

     3. **Depth Limit**:
        Create scenario where agent tries to nest 4+ levels
        Expected: Error message about max depth

     4. **Specialized Agents Direct**:
        ```bash
        ./finta chat --agent-type explore "Find all .go files in internal/"
        ./finta chat --agent-type plan "Plan how to add a new tool"
        ./finta chat --agent-type execute "Create a test file"
        ```

     ---

     ## Edge Cases and Error Handling

     ### Task Tool Errors:
     1. **Invalid agent type**: Return Result with Success=false and clear error
     2. **Max depth exceeded**: Return descriptive error, don't crash
     3. **Sub-agent execution fails**: Capture error in Result, return to parent
     4. **Missing logger**: Create default logger as fallback

     ### Factory Errors:
     1. **Invalid agent type string**: Return error from CreateAgent()
     2. **Nil LLM client**: Will panic (document as required)
     3. **Nil tool registry**: Will panic (document as required)

     ### Context Handling:
     1. **Context cancellation**: Propagates to sub-agents automatically
     2. **Missing logger in context**: Task tool creates default
     3. **Nesting depth key missing**: Assumes depth 0

     ---

     ## Migration Path

     ### Phase 3 → Phase 4 (Future):
     - Add more specialized agents (e.g., Researcher, Debugger)
     - Add agent-specific metrics in Config
     - Support agent communication via message passing
     - Add agent pooling/reuse

     ---

     ## Completion Criteria

     - ✅ Agent type system with 4 types (General, Explore, Plan, Execute)
     - ✅ Factory pattern implementation
     - ✅ Each agent has correct tool subset
     - ✅ Each agent has appropriate system prompt
     - ✅ Task tool spawns sub-agents correctly
     - ✅ Maximum nesting depth of 3 enforced
     - ✅ Sub-agent logs merged with parent
     - ✅ CLI supports `--agent-type` flag
     - ✅ Build succeeds without errors
     - ✅ Basic integration test passes

     ---

     ## Implementation Order

     1. Create `types.go` with factory (30 min)
     2. Create `task.go` with Task tool (45 min)
     3. Modify `base.go` to add logger to context (15 min)
     4. Update `main.go` with factory and flags (20 min)
     5. Build and fix any compilation errors (15 min)
     6. Test basic functionality (30 min)

     **Total estimated time**: ~2.5 hours
