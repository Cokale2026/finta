package tool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"finta/internal/hook"
	"finta/internal/llm"
)

type ExecutionMode string

const (
	ExecutionModeSequential ExecutionMode = "sequential"
	ExecutionModeParallel   ExecutionMode = "parallel"
	ExecutionModeMixed      ExecutionMode = "mixed"
)

type Executor struct {
	registry    *Registry
	mode        ExecutionMode
	hookManager *hook.Manager
}

func NewExecutor(registry *Registry) *Executor {
	return &Executor{
		registry: registry,
		mode:     ExecutionModeMixed, // Default to smart mixed mode
	}
}

func (e *Executor) SetMode(mode ExecutionMode) {
	e.mode = mode
}

// SetHookManager sets the hook manager for tool execution hooks
func (e *Executor) SetHookManager(manager *hook.Manager) {
	e.hookManager = manager
}

// Execute executes tool calls based on the configured mode
func (e *Executor) Execute(ctx context.Context, toolCalls []*llm.ToolCall) ([]*CallResult, error) {
	switch e.mode {
	case ExecutionModeSequential:
		return e.ExecuteSequential(ctx, toolCalls)
	case ExecutionModeParallel:
		return e.ExecuteParallel(ctx, toolCalls)
	case ExecutionModeMixed:
		return e.ExecuteMixed(ctx, toolCalls)
	default:
		return e.ExecuteSequential(ctx, toolCalls)
	}
}

// ExecuteSequential executes tools one by one in order
func (e *Executor) ExecuteSequential(ctx context.Context, toolCalls []*llm.ToolCall) ([]*CallResult, error) {
	results := make([]*CallResult, len(toolCalls))

	for i, tc := range toolCalls {
		result, err := e.executeOne(ctx, tc)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}

	return results, nil
}

// ExecuteParallel executes all tools concurrently
func (e *Executor) ExecuteParallel(ctx context.Context, toolCalls []*llm.ToolCall) ([]*CallResult, error) {
	results := make([]*CallResult, len(toolCalls))
	errs := make([]error, len(toolCalls))

	var wg sync.WaitGroup
	for i, tc := range toolCalls {
		wg.Add(1)
		go func(idx int, call *llm.ToolCall) {
			defer wg.Done()

			result, err := e.executeOne(ctx, call)
			if err != nil {
				errs[idx] = err
				return
			}
			results[idx] = result
		}(i, tc)
	}

	wg.Wait()

	// Check for errors
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

// ExecuteMixed intelligently executes tools based on dependency analysis
func (e *Executor) ExecuteMixed(ctx context.Context, toolCalls []*llm.ToolCall) ([]*CallResult, error) {
	// Analyze dependencies
	deps := e.analyzeDependencies(toolCalls)

	// If no dependencies, execute all in parallel
	if len(deps) == 0 {
		return e.ExecuteParallel(ctx, toolCalls)
	}

	// Build execution batches based on dependencies
	batches := e.buildExecutionBatches(toolCalls, deps)

	// Track all results in order
	allResults := make([]*CallResult, len(toolCalls))
	resultMap := make(map[int]*CallResult)

	// Execute batch by batch
	for _, batch := range batches {
		batchCalls := make([]*llm.ToolCall, len(batch))
		for i, idx := range batch {
			batchCalls[i] = toolCalls[idx]
		}

		// Execute batch in parallel
		batchResults, err := e.ExecuteParallel(ctx, batchCalls)
		if err != nil {
			return nil, err
		}

		// Map results back to original indices
		for i, idx := range batch {
			resultMap[idx] = batchResults[i]
		}
	}

	// Reconstruct results in original order
	for i := 0; i < len(toolCalls); i++ {
		allResults[i] = resultMap[i]
	}

	return allResults, nil
}

// EmptyOutputPlaceholder is returned when a tool produces no output.
// This ensures LLM APIs (which require non-empty content) don't fail with 400 errors.
const EmptyOutputPlaceholder = "(Tool executed successfully with no output)"

func (e *Executor) executeOne(ctx context.Context, tc *llm.ToolCall) (*CallResult, error) {
	startTime := time.Now()

	t, err := e.registry.Get(tc.Function.Name)
	if err != nil {
		return &CallResult{
			ToolName:  tc.Function.Name,
			CallID:    tc.ID,
			Result:    &Result{Success: false, Error: err.Error()},
			StartTime: startTime,
			EndTime:   time.Now(),
		}, nil
	}

	// Trigger before tool execution hook
	if e.hookManager != nil {
		hookData := hook.NewHookData(hook.BeforeToolExecution, tc.Function.Name).
			Set("params", tc.Function.Arguments)

		feedback, err := e.hookManager.Trigger(ctx, hookData)
		if err != nil {
			return &CallResult{
				ToolName:  tc.Function.Name,
				CallID:    tc.ID,
				Result:    &Result{Success: false, Error: fmt.Sprintf("hook error: %v", err)},
				StartTime: startTime,
				EndTime:   time.Now(),
			}, nil
		}

		if !feedback.Allow {
			denyMsg := fmt.Sprintf("Tool execution was DENIED by user. Reason: %s. Please ask the user for guidance on how to proceed.", feedback.Message)
			return &CallResult{
				ToolName:  tc.Function.Name,
				CallID:    tc.ID,
				Result:    &Result{Success: false, Output: denyMsg, Error: denyMsg},
				StartTime: startTime,
				EndTime:   time.Now(),
			}, nil
		}

		// Add hook manager to context for tools that need it (like bash)
		ctx = hook.WithManager(ctx, e.hookManager)
	}

	result, err := t.Execute(ctx, []byte(tc.Function.Arguments))
	if err != nil {
		return &CallResult{
			ToolName:  tc.Function.Name,
			CallID:    tc.ID,
			Result:    &Result{Success: false, Error: err.Error()},
			StartTime: startTime,
			EndTime:   time.Now(),
		}, nil
	}

	// Trigger after tool execution hook
	if e.hookManager != nil {
		hookData := hook.NewHookData(hook.AfterToolExecution, tc.Function.Name).
			Set("params", tc.Function.Arguments).
			Set("result", result).
			Set("duration", time.Since(startTime))

		// After hooks don't block, just trigger
		_, _ = e.hookManager.Trigger(ctx, hookData)
	}

	// Ensure non-empty output for LLM APIs that require non-empty content
	if result.Output == "" {
		result.Output = EmptyOutputPlaceholder
	}

	return &CallResult{
		ToolName:  tc.Function.Name,
		CallID:    tc.ID,
		Params:    []byte(tc.Function.Arguments),
		Result:    result,
		StartTime: startTime,
		EndTime:   time.Now(),
	}, nil
}

// analyzeDependencies performs heuristic dependency analysis
// Rules:
// - read/bash/grep may depend on write (if write comes before them)
// - This is a simple heuristic, not perfect dependency tracking
func (e *Executor) analyzeDependencies(toolCalls []*llm.ToolCall) map[int][]int {
	deps := make(map[int][]int)

	// Track write operations
	writeIndices := []int{}
	for i, tc := range toolCalls {
		if tc.Function.Name == "write" {
			writeIndices = append(writeIndices, i)
		}
	}

	// Tools that might depend on writes
	dependentTools := map[string]bool{
		"read": true,
		"bash": true,
		"grep": true,
		"glob": true,
	}

	// For each tool, check if it depends on any previous write
	for i, tc := range toolCalls {
		if dependentTools[tc.Function.Name] {
			// Check if there are any write operations before this
			for _, writeIdx := range writeIndices {
				if writeIdx < i {
					// This tool might depend on this write
					deps[i] = append(deps[i], writeIdx)
				}
			}
		}
	}

	return deps
}

// buildExecutionBatches creates execution batches using topological sort
func (e *Executor) buildExecutionBatches(toolCalls []*llm.ToolCall, deps map[int][]int) [][]int {
	batches := make([][]int, 0)
	executed := make(map[int]bool)

	for len(executed) < len(toolCalls) {
		batch := make([]int, 0)

		// Find all tools whose dependencies have been executed
		for i := range toolCalls {
			if executed[i] {
				continue
			}

			// Check if all dependencies are satisfied
			canExecute := true
			for _, dep := range deps[i] {
				if !executed[dep] {
					canExecute = false
					break
				}
			}

			if canExecute {
				batch = append(batch, i)
			}
		}

		// If no tools can be executed, we have a circular dependency
		// Force execute remaining tools to prevent infinite loop
		if len(batch) == 0 {
			for i := range toolCalls {
				if !executed[i] {
					batch = append(batch, i)
				}
			}
		}

		// Mark batch as executed
		for _, idx := range batch {
			executed[idx] = true
		}

		batches = append(batches, batch)
	}

	return batches
}

// GetDependencyInfo returns dependency information for debugging
func (e *Executor) GetDependencyInfo(toolCalls []*llm.ToolCall) string {
	deps := e.analyzeDependencies(toolCalls)
	batches := e.buildExecutionBatches(toolCalls, deps)

	info := "Tool Call Dependency Analysis:\n"
	info += fmt.Sprintf("Total tools: %d\n", len(toolCalls))
	info += fmt.Sprintf("Dependencies found: %d\n\n", len(deps))

	for i, tc := range toolCalls {
		info += fmt.Sprintf("[%d] %s", i, tc.Function.Name)
		if depList, hasDeps := deps[i]; hasDeps {
			info += fmt.Sprintf(" (depends on: %v)", depList)
		}
		info += "\n"
	}

	info += fmt.Sprintf("\nExecution plan: %d batch(es)\n", len(batches))
	for i, batch := range batches {
		info += fmt.Sprintf("Batch %d (parallel): %v\n", i+1, batch)
	}

	return info
}
