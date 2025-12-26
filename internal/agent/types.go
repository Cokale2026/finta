package agent

import (
	"fmt"

	"finta/internal/llm"
	"finta/internal/tool"
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
	llmClient           llm.Client
	toolRegistry        *tool.Registry
	includeBestPractices bool // Whether to include tool best practices in system prompts
}

// NewDefaultFactory creates a new agent factory with best practices enabled by default
func NewDefaultFactory(client llm.Client, registry *tool.Registry) *DefaultFactory {
	return &DefaultFactory{
		llmClient:            client,
		toolRegistry:         registry,
		includeBestPractices: true, // Enable by default
	}
}

// SetIncludeBestPractices enables or disables including tool best practices in system prompts
func (f *DefaultFactory) SetIncludeBestPractices(include bool) {
	f.includeBestPractices = include
}

// buildSystemPrompt constructs a system prompt with optional tool best practices
func (f *DefaultFactory) buildSystemPrompt(basePrompt string) string {
	if !f.includeBestPractices {
		return basePrompt
	}

	bestPractices := f.toolRegistry.GetToolBestPractices()
	if bestPractices == "" {
		return basePrompt
	}

	return basePrompt + "\n\n" + bestPractices
}

// CreateAgent creates an agent of the specified type
func (f *DefaultFactory) CreateAgent(agentType AgentType) (Agent, error) {
	switch agentType {
	case AgentTypeGeneral:
		return f.createGeneralAgent()
	case AgentTypeExplore:
		return f.createExploreAgent()
	case AgentTypePlan:
		return f.createPlanAgent()
	case AgentTypeExecute:
		return f.createExecuteAgent()
	default:
		return nil, fmt.Errorf("unknown agent type: %s", agentType)
	}
}

// createGeneralAgent creates a general-purpose agent with access to all tools
func (f *DefaultFactory) createGeneralAgent() (Agent, error) {
	basePrompt := `You are a helpful AI assistant with access to tools.
You can read files, execute bash commands, write files, find files with glob
patterns, and search files with grep.

When solving tasks, follow the ReAct pattern:
1. **Think**: Explain your reasoning before taking action
2. **Act**: Use tools to gather information or make changes
3. **Observe**: Analyze the results and plan next steps

Always provide clear, concise responses.`

	systemPrompt := f.buildSystemPrompt(basePrompt)

	return NewBaseAgent(
		"general",
		systemPrompt,
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
	), nil
}

// createExploreAgent creates an exploration-focused agent with read-only tools
func (f *DefaultFactory) createExploreAgent() (Agent, error) {
	// Create filtered registry with read-only tools
	exploreRegistry := tool.NewRegistry()

	// Get tools from main registry - must succeed for required tools
	requiredTools := []string{"read", "glob", "grep", "bash"}
	for _, name := range requiredTools {
		t, err := f.toolRegistry.Get(name)
		if err != nil {
			return nil, fmt.Errorf("explore agent requires tool '%s' but it's not registered: %w", name, err)
		}
		if err := exploreRegistry.Register(t); err != nil {
			return nil, fmt.Errorf("failed to register tool '%s' for explore agent: %w", name, err)
		}
	}

	basePrompt := `You are an expert codebase exploration agent.

Your goal is to efficiently explore and understand codebases using the ReAct pattern:
- **Think**: Before each action, explain what you're looking for and why
- **Act**: Use read-only tools (read, glob, grep, bash)
- **Observe**: Summarize findings and decide next exploration steps

You have access to read-only tools:
- read: Read file contents
- glob: Find files matching patterns
- grep: Search for content in files
- bash: Execute read-only commands (ls, find, cat, etc.)

Always provide clear summaries of your findings.`

	// Build system prompt with best practices from the filtered explore registry
	systemPrompt := f.buildSystemPrompt(basePrompt)

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
	), nil
}

// createPlanAgent creates a planning-focused agent with limited tools
func (f *DefaultFactory) createPlanAgent() (Agent, error) {
	// Create filtered registry with read and glob only
	planRegistry := tool.NewRegistry()

	// Get tools from main registry - must succeed for required tools
	requiredTools := []string{"read", "glob"}
	for _, name := range requiredTools {
		t, err := f.toolRegistry.Get(name)
		if err != nil {
			return nil, fmt.Errorf("plan agent requires tool '%s' but it's not registered: %w", name, err)
		}
		if err := planRegistry.Register(t); err != nil {
			return nil, fmt.Errorf("failed to register tool '%s' for plan agent: %w", name, err)
		}
	}

	basePrompt := `You are an expert software architect and planning agent.

Your goal is to create detailed, actionable implementation plans using the ReAct pattern:
- **Think**: Analyze the requirements and existing codebase
- **Act**: Read files to understand current state
- **Observe**: Synthesize findings into comprehensive plans

You can read files to understand the current codebase state.

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

	systemPrompt := f.buildSystemPrompt(basePrompt)

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
	), nil
}

// createExecuteAgent creates an execution-focused agent with all tools
func (f *DefaultFactory) createExecuteAgent() (Agent, error) {
	basePrompt := `You are an expert implementation agent focused on careful,
precise code execution.

Your goal is to implement changes accurately and safely using the ReAct pattern:
- **Think**: Plan changes carefully before executing
- **Act**: Use tools to read, modify, and verify code
- **Observe**: Check results and ensure correctness

You have access to all tools:
- read: Read file contents
- write: Create or overwrite files
- bash: Execute bash commands
- glob: Find files matching patterns
- grep: Search for content in files

Focus on correctness and safety over speed.`

	systemPrompt := f.buildSystemPrompt(basePrompt)

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
	), nil
}
