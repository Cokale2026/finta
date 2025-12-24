package agent

import (
	"fmt"
	"time"

	"finta/internal/logger"
)

// ExecutionContext tracks the execution state of an agent and provides logging utilities
type ExecutionContext struct {
	Logger        *logger.Logger
	StartTime     time.Time
	CurrentTurn   int
	TotalTurns    int
	ToolCallCount int
}

// NewExecutionContext creates a new execution context with the given logger
func NewExecutionContext(log *logger.Logger) *ExecutionContext {
	return &ExecutionContext{
		Logger:    log,
		StartTime: time.Now(),
	}
}

// LogToolCall logs a tool call with its parameters
func (ctx *ExecutionContext) LogToolCall(toolName, params string) {
	ctx.ToolCallCount++
	ctx.Logger.ToolCall(toolName, params)
}

// LogToolResult logs a tool execution result
func (ctx *ExecutionContext) LogToolResult(toolName string, success bool, output string, duration time.Duration) {
	ctx.Logger.ToolResult(toolName, success, output, duration)
}

// LogResponse logs the agent's response
func (ctx *ExecutionContext) LogResponse(content string) {
	ctx.Logger.AgentResponse(content)
}

// LogProgress logs the current progress (turn X of Y)
func (ctx *ExecutionContext) LogProgress() {
	ctx.Logger.Progress(ctx.CurrentTurn, ctx.TotalTurns,
		fmt.Sprintf("Turn %d/%d", ctx.CurrentTurn, ctx.TotalTurns))
}
