package agent

import (
	"context"
	"fmt"
	"time"

	"finta/internal/logger"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// LoggerContextKey is the context key for storing the logger
	LoggerContextKey ContextKey = "logger"
	// NestingDepthKey is the context key for tracking sub-agent nesting depth
	NestingDepthKey ContextKey = "nesting_depth"
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

// LogReasoning logs the agent's reasoning/thinking process
func (ctx *ExecutionContext) LogReasoning(content string) {
	ctx.Logger.AgentReasoning(content)
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

// GetLoggerFromContext retrieves the logger stored in context
func GetLoggerFromContext(ctx context.Context) *logger.Logger {
	if log, ok := ctx.Value(LoggerContextKey).(*logger.Logger); ok {
		return log
	}
	return nil
}

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, log *logger.Logger) context.Context {
	return context.WithValue(ctx, LoggerContextKey, log)
}

// GetNestingDepth retrieves the current nesting depth from context
func GetNestingDepth(ctx context.Context) int {
	if depth, ok := ctx.Value(NestingDepthKey).(int); ok {
		return depth
	}
	return 0
}

// WithNestingDepth adds nesting depth to the context
func WithNestingDepth(ctx context.Context, depth int) context.Context {
	return context.WithValue(ctx, NestingDepthKey, depth)
}
