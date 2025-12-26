package tool

import (
	"finta/internal/llm"
	"fmt"
	"sync"
)

type Registry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.Name()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %s already registered", name)
	}

	r.tools[name] = tool
	return nil
}

func (r *Registry) Get(name string) (Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	return tool, nil
}

func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

func (r *Registry) GetToolDefinitions() []*llm.ToolDefinition {
	tools := r.List()
	defs := make([]*llm.ToolDefinition, len(tools))

	for i, t := range tools {
		defs[i] = &llm.ToolDefinition{
			Type: "function",
			Function: &llm.FunctionDef{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
			},
		}
	}

	return defs
}

// GetToolBestPractices collects best practices from all registered tools
// that implement the ToolWithBestPractices interface
func (r *Registry) GetToolBestPractices() string {
	tools := r.List()
	var practices []string

	for _, t := range tools {
		if bp := t.BestPractices(); bp != "" {
			practices = append(practices, bp)
		}
	}

	if len(practices) == 0 {
		return ""
	}

	// Join all practices with double newline separator
	result := "# Tool Usage Best Practices\n\n"
	for i, practice := range practices {
		result += practice
		if i < len(practices)-1 {
			result += "\n\n"
		}
	}

	return result
}
