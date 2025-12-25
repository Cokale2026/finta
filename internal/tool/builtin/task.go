package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"finta/internal/agent"
	"finta/internal/logger"
	"finta/internal/tool"
)

// nestingDepthKey is the context key for tracking nesting depth
type contextKey string

const nestingDepthKey contextKey = "nesting_depth"

// MaxNestingDepth is the maximum allowed depth for sub-agent calls
const MaxNestingDepth = 3

// TaskTool launches specialized sub-agents
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

	// Check nesting depth
	depth := getNestingDepth(ctx)
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

// getNestingDepth retrieves the current nesting depth from context
func getNestingDepth(ctx context.Context) int {
	if depth, ok := ctx.Value(nestingDepthKey).(int); ok {
		return depth
	}
	return 0
}

// getLoggerFromContext retrieves the logger from context using the agent's helper
func getLoggerFromContext(ctx context.Context) *logger.Logger {
	return agent.GetLoggerFromContext(ctx)
}
