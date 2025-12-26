package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"finta/internal/agent"
	"finta/internal/logger"
	"finta/internal/tool"
)

// MaxNestingDepth is the maximum allowed depth for sub-agent calls
const MaxNestingDepth = 3

// TaskTool launches specialized sub-agents for specific tasks.
//
// Note: While 'general' agent type is supported, spawning general agents
// as sub-agents is discouraged as it may lead to overly complex nesting.
// Prefer using specialized agents (explore, plan, execute) for focused tasks.
type TaskTool struct {
	factory agent.Factory
}

// NewTaskTool creates a new Task tool
func NewTaskTool(factory agent.Factory) *TaskTool {
	return &TaskTool{
		factory: factory,
	}
}

func (t *TaskTool) Name() string {
	return "task"
}

func (t *TaskTool) BestPractices() string {
	return `**Task Tool Best Practices**:

1. **Use specialized agents for focused tasks**:
   - explore: For codebase exploration, finding files, searching code
   - plan: For creating implementation plans, breaking down work
   - execute: For implementing changes, writing code
   - general: Use sparingly as sub-agent (prefer specialized types)

2. **Provide clear, specific task descriptions** - Be precise about what the sub-agent should do
   - Good: "Find all authentication-related files in internal/"
   - Bad: "Look at the code"

3. **Use appropriate agent types**:
   - Exploring unfamiliar code → explore agent
   - Planning implementation → plan agent
   - Writing/modifying code → execute agent
   - Complex multi-faceted task → general agent (last resort)

4. **Avoid deep nesting** - Maximum 3 levels of sub-agents
   - If you need deep nesting, reconsider your approach
   - Break the problem differently

5. **Include short descriptions** - Help users understand what each sub-agent is doing
   - Format: 3-5 words describing the sub-task
   - Example: "Explore authentication code", "Plan database migration"

6. **Don't spawn sub-agents for simple tasks** - Direct tool use is more efficient
   - Bad: Spawning sub-agent just to read one file
   - Good: Use read tool directly`
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
				"description": "Type of agent to spawn (general, explore, plan, execute)",
				"enum":        []string{"general", "explore", "plan", "execute"},
			},
			"task": map[string]any{
				"type":        "string",
				"description": "Task description for the sub-agent",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Short description of what this sub-agent will do (3-5 words)",
			},
			"max_turns": map[string]any{
				"type":        "number",
				"description": "Maximum turns for sub-agent (optional, defaults to agent type default)",
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
		MaxTurns    int    `json:"max_turns"`
	}

	if err := json.Unmarshal(params, &p); err != nil {
		return &tool.Result{
			Success: false,
			Error:   fmt.Sprintf("invalid parameters: %v", err),
		}, nil
	}

	// Validate required parameters
	if len(p.Task) == 0 {
		return &tool.Result{
			Success: false,
			Error:   "task parameter cannot be empty",
		}, nil
	}
	if len(p.Description) == 0 {
		return &tool.Result{
			Success: false,
			Error:   "description parameter cannot be empty",
		}, nil
	}
	if len(p.AgentType) == 0 {
		return &tool.Result{
			Success: false,
			Error:   "agent_type parameter cannot be empty",
		}, nil
	}

	// Check nesting depth
	depth := agent.GetNestingDepth(ctx)
	if depth >= MaxNestingDepth {
		return &tool.Result{
			Success: false,
			Error:   fmt.Sprintf("maximum nesting depth (%d) exceeded", MaxNestingDepth),
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

	// Get parent logger from context
	parentLogger := agent.GetLoggerFromContext(ctx)
	if parentLogger == nil {
		// Fallback: create a basic logger
		parentLogger = logger.NewLogger(nil, logger.LevelInfo)
	}

	// Log sub-agent start
	parentLogger.Info("Launching %s sub-agent: %s", p.AgentType, p.Description)

	// Create context with incremented depth
	subCtx := agent.WithNestingDepth(ctx, depth+1)

	// Run sub-agent
	output, err := subAgent.Run(subCtx, &agent.Input{
		Task:        p.Task,
		MaxTurns:    p.MaxTurns,   // Use provided or default (0 = agent default)
		Temperature: 0,            // Use agent default
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
