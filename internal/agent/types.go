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
	llmClient    llm.Client
	toolRegistry *tool.Registry
}

// NewDefaultFactory creates a new agent factory
func NewDefaultFactory(client llm.Client, registry *tool.Registry) *DefaultFactory {
	return &DefaultFactory{
		llmClient:    client,
		toolRegistry: registry,
	}
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
	systemPrompt := `You are a helpful AI assistant with access to tools.
You can read files, execute bash commands, write files, find files with glob
patterns, and search files with grep.
Always provide clear, concise responses.`

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

	systemPrompt := `You are an expert codebase exploration agent.

Your goal is to efficiently explore and understand codebases. You have access to read-only tools:
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
	), nil
}

// createExecuteAgent creates an execution-focused agent with all tools
func (f *DefaultFactory) createExecuteAgent() (Agent, error) {
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
	), nil
}
