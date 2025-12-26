package agent

import (
	"context"
	"fmt"
	"time"

	"finta/internal/llm"
	"finta/internal/tool"
)

type BaseAgent struct {
	name         string
	systemPrompt string
	llmClient    llm.Client
	toolRegistry *tool.Registry
	toolExecutor *tool.Executor
	config       *Config
}

func NewBaseAgent(name, systemPrompt string, client llm.Client, registry *tool.Registry, cfg *Config) *BaseAgent {
	if cfg == nil {
		cfg = &Config{
			Model:               "gpt-4-turbo",
			Temperature:         0.7,
			MaxTokens:           4096,
			MaxTurns:            20,
			EnableParallelTools: true,
			ToolExecutionMode:   tool.ExecutionModeMixed,
		}
	}

	executor := tool.NewExecutor(registry)
	if cfg.ToolExecutionMode != "" {
		executor.SetMode(cfg.ToolExecutionMode)
	}

	return &BaseAgent{
		name:         name,
		systemPrompt: systemPrompt,
		llmClient:    client,
		toolRegistry: registry,
		toolExecutor: executor,
		config:       cfg,
	}
}

func (a *BaseAgent) Name() string {
	return a.name
}

func (a *BaseAgent) Run(ctx context.Context, input *Input) (*Output, error) {
	// Create execution context
	execCtx := NewExecutionContext(input.Logger)

	// Add logger to context for sub-agents
	ctx = WithLogger(ctx, input.Logger)

	// Log session start
	execCtx.Logger.SessionStart(input.Task)

	// Initialize message list
	messages := make([]llm.Message, 0, len(input.Messages)+1)

	// Add system prompt
	if a.systemPrompt != "" {
		messages = append(messages, llm.Message{
			Role:    llm.RoleSystem,
			Content: a.systemPrompt,
		})
	}

	// Add history messages
	messages = append(messages, input.Messages...)

	// Add user task
	if input.Task != "" {
		messages = append(messages, llm.Message{
			Role:      llm.RoleUser,
			Content:   input.Task,
			Timestamp: time.Now(),
		})
	}

	maxTurns := input.MaxTurns
	if maxTurns == 0 {
		maxTurns = a.config.MaxTurns
	}
	execCtx.TotalTurns = maxTurns

	allToolCalls := make([]*tool.CallResult, 0)

	// Agent run loop
	for turn := 0; turn < maxTurns; turn++ {
		execCtx.CurrentTurn = turn + 1

		execCtx.Logger.Info("Turn %d: Calling LLM...", turn+1)

		// Call LLM
		resp, err := a.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    messages,
			Tools:       a.toolRegistry.GetToolDefinitions(),
			Temperature: input.Temperature,
			MaxTokens:   a.config.MaxTokens,
		})
		if err != nil {
			execCtx.Logger.Error("LLM call failed: %v", err)
			return nil, fmt.Errorf("LLM call failed: %w", err)
		}

		// Add assistant message
		messages = append(messages, resp.Message)

		// Save reasoning and log it
		if resp.Message.Reason != "" {
			execCtx.LogReasoning(resp.Message.Reason)
		}

		// Log agent response if present
		if resp.Message.Content != "" {
			execCtx.LogResponse(resp.Message.Content)
		}

		// Check if done
		if resp.StopReason == llm.StopReasonStop {
			execCtx.Logger.SessionEnd(
				time.Since(execCtx.StartTime),
				execCtx.ToolCallCount,
			)
			return &Output{
				Messages:  messages,
				Result:    resp.Message.Content,
				ToolCalls: allToolCalls,
			}, nil
		}

		// Handle tool calls
		if resp.StopReason == llm.StopReasonToolCalls {
			execCtx.Logger.Info("Executing %d tool call(s)...", len(resp.Message.ToolCalls))

			toolResults, err := a.executeToolsWithLogging(ctx, resp.Message.ToolCalls, execCtx)
			if err != nil {
				execCtx.Logger.Error("Tool execution failed: %v", err)
				return nil, fmt.Errorf("tool execution failed: %w", err)
			}

			allToolCalls = append(allToolCalls, toolResults...)

			// Add tool result messages
			for _, tr := range toolResults {
				messages = append(messages, llm.Message{
					Role:       llm.RoleTool,
					ToolCallID: tr.CallID,
					Content:    tr.Result.Output,
					Name:       tr.ToolName,
					Timestamp:  tr.EndTime,
				})
			}

			continue
		}

		// If stopped due to length limit
		if resp.StopReason == llm.StopReasonLength {
			execCtx.Logger.SessionEnd(
				time.Since(execCtx.StartTime),
				execCtx.ToolCallCount,
			)
			return &Output{
				Messages:  messages,
				Result:    resp.Message.Content + "\n[Response truncated due to length limit]",
				ToolCalls: allToolCalls,
			}, nil
		}
	}

	execCtx.Logger.Error("Max turns exceeded")
	return nil, fmt.Errorf("max turns (%d) exceeded", maxTurns)
}

// executeToolsWithLogging executes tool calls with comprehensive logging
// Uses the executor for parallel or sequential execution based on config
func (a *BaseAgent) executeToolsWithLogging(
	ctx context.Context,
	toolCalls []*llm.ToolCall,
	execCtx *ExecutionContext,
) ([]*tool.CallResult, error) {
	// Log all tool calls first
	for _, tc := range toolCalls {
		execCtx.LogToolCall(tc.Function.Name, tc.Function.Arguments)
	}

	// Execute tools using executor (handles parallel/sequential/mixed execution)
	results, err := a.toolExecutor.Execute(ctx, toolCalls)
	if err != nil {
		return nil, err
	}

	// Log all results and build ReActTrace if enabled
	for _, result := range results {
		duration := result.EndTime.Sub(result.StartTime)
		execCtx.LogToolResult(result.ToolName, result.Result.Success, result.Result.Output, duration)
	}

	return results, nil
}

// RunStreaming runs the agent with streaming output
func (a *BaseAgent) RunStreaming(ctx context.Context, input *Input, streamChan chan<- string) (*Output, error) {
	// Create execution context
	execCtx := NewExecutionContext(input.Logger)

	// Add logger to context for sub-agents
	ctx = WithLogger(ctx, input.Logger)

	// Log session start
	execCtx.Logger.SessionStart(input.Task)

	// Initialize message list
	messages := make([]llm.Message, 0, len(input.Messages)+1)

	// Add system prompt
	if a.systemPrompt != "" {
		messages = append(messages, llm.Message{
			Role:    llm.RoleSystem,
			Content: a.systemPrompt,
		})
	}

	// Add history messages
	messages = append(messages, input.Messages...)

	// Add user task
	if input.Task != "" {
		messages = append(messages, llm.Message{
			Role:      llm.RoleUser,
			Content:   input.Task,
			Timestamp: time.Now(),
		})
	}

	maxTurns := input.MaxTurns
	if maxTurns == 0 {
		maxTurns = a.config.MaxTurns
	}
	execCtx.TotalTurns = maxTurns

	allToolCalls := make([]*tool.CallResult, 0)

	// Agent run loop
	for turn := 0; turn < maxTurns; turn++ {
		execCtx.CurrentTurn = turn + 1

		execCtx.Logger.Info("Turn %d: Calling LLM (streaming)...", turn+1)

		// Call LLM with streaming
		reader, err := a.llmClient.ChatStream(ctx, &llm.ChatRequest{
			Messages:    messages,
			Tools:       a.toolRegistry.GetToolDefinitions(),
			Temperature: input.Temperature,
			MaxTokens:   a.config.MaxTokens,
		})
		if err != nil {
			execCtx.Logger.Error("LLM streaming call failed: %v", err)
			return nil, fmt.Errorf("LLM streaming call failed: %w", err)
		}

		// Stream content and accumulate message
		accumulatedMsg := llm.Message{
			Role:      llm.RoleAssistant,
			Reason:    "",
			Content:   "",
			ToolCalls: nil,
		}

		for {
			delta, err := reader.Recv()
			if err != nil {
				reader.Close()
				execCtx.Logger.Error("Stream recv failed: %v", err)
				return nil, fmt.Errorf("stream recv failed: %w", err)
			}

			if delta.Done {
				break
			}

			// Accumulate and send reasoning to stream channel
			if delta.Reason != "" {
				accumulatedMsg.Reason += delta.Reason
				select {
				case streamChan <- delta.Reason:
				case <-ctx.Done():
					reader.Close()
					return nil, ctx.Err()
				}
			}

			// Send content to stream channel
			if delta.Content != "" {
				accumulatedMsg.Content += delta.Content
				select {
				case streamChan <- delta.Content:
				case <-ctx.Done():
					reader.Close()
					return nil, ctx.Err()
				}
			}

			// Accumulate tool calls
			if len(delta.ToolCalls) > 0 {
				accumulatedMsg.ToolCalls = delta.ToolCalls
			}
		}

		reader.Close()

		// Add assistant message
		messages = append(messages, accumulatedMsg)

		// Log reasoning if present
		if accumulatedMsg.Reason != "" {
			execCtx.LogReasoning(accumulatedMsg.Reason)
		}

		// Check if done
		if len(accumulatedMsg.ToolCalls) == 0 {
			// No tool calls, we're done
			execCtx.Logger.SessionEnd(
				time.Since(execCtx.StartTime),
				execCtx.ToolCallCount,
			)
			return &Output{
				Messages:  messages,
				Result:    accumulatedMsg.Content,
				ToolCalls: allToolCalls,
			}, nil
		}

		// Handle tool calls
		execCtx.Logger.Info("Executing %d tool call(s)...", len(accumulatedMsg.ToolCalls))

		toolResults, err := a.executeToolsWithLogging(ctx, accumulatedMsg.ToolCalls, execCtx)
		if err != nil {
			execCtx.Logger.Error("Tool execution failed: %v", err)
			return nil, fmt.Errorf("tool execution failed: %w", err)
		}

		allToolCalls = append(allToolCalls, toolResults...)

		// Add tool result messages
		for _, tr := range toolResults {
			messages = append(messages, llm.Message{
				Role:       llm.RoleTool,
				ToolCallID: tr.CallID,
				Content:    tr.Result.Output,
				Name:       tr.ToolName,
				Timestamp:  tr.EndTime,
			})
		}
	}

	execCtx.Logger.Error("Max turns exceeded")
	return nil, fmt.Errorf("max turns (%d) exceeded", maxTurns)
}
