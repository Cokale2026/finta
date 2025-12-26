package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"finta/internal/tool"
)

// TodoStatus represents the state of a todo item
type TodoStatus string

const (
	StatusPending    TodoStatus = "pending"
	StatusInProgress TodoStatus = "in_progress"
	StatusCompleted  TodoStatus = "completed"
)

// TodoItem represents a single todo task
type TodoItem struct {
	Content    string     `json:"content"`     // Task description (imperative form: "Run tests")
	Status     TodoStatus `json:"status"`      // Current status
	ActiveForm string     `json:"activeForm"`  // Present continuous form ("Running tests")
}

// TodoStore manages the global todo list state
type TodoStore struct {
	todos []TodoItem
	mu    sync.RWMutex
}

var globalTodoStore = &TodoStore{
	todos: make([]TodoItem, 0),
}

// GetTodos returns a copy of the current todo list
func (s *TodoStore) GetTodos() []TodoItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	todos := make([]TodoItem, len(s.todos))
	copy(todos, s.todos)
	return todos
}

// SetTodos updates the entire todo list
func (s *TodoStore) SetTodos(todos []TodoItem) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.todos = todos
}

// Clear removes all todos
func (s *TodoStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.todos = make([]TodoItem, 0)
}

// TodoWriteTool manages and displays todo lists for task tracking
type TodoWriteTool struct{}

func NewTodoWriteTool() *TodoWriteTool {
	return &TodoWriteTool{}
}

func (t *TodoWriteTool) Name() string {
	return "TodoWrite"
}

func (t *TodoWriteTool) Description() string {
	return `Manage and display a todo list for tracking task progress.

Use this tool to:
- Create a todo list when starting complex multi-step tasks (3+ steps)
- Update task status as work progresses (pending -> in_progress -> completed)
- Keep exactly ONE task in_progress at a time
- Mark tasks completed IMMEDIATELY after finishing them
- Remove the entire todo list when all tasks are done

Task lifecycle:
1. Create tasks as 'pending' when identified
2. Mark as 'in_progress' when starting work (only ONE at a time)
3. Mark as 'completed' when finished
4. Remove all todos when everything is complete

Each todo requires:
- content: Imperative form ("Run tests", "Fix bug")
- status: "pending" | "in_progress" | "completed"
- activeForm: Present continuous form ("Running tests", "Fixing bug")`
}

func (t *TodoWriteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"todos": map[string]any{
				"type": "array",
				"description": "Array of todo items. Pass empty array [] to clear all todos.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"content": map[string]any{
							"type":        "string",
							"description": "Task description in imperative form (e.g., 'Run tests', 'Fix authentication bug')",
						},
						"status": map[string]any{
							"type":        "string",
							"enum":        []string{"pending", "in_progress", "completed"},
							"description": "Current status of the task. Keep exactly ONE task as 'in_progress' at a time.",
						},
						"activeForm": map[string]any{
							"type":        "string",
							"description": "Present continuous form of the task (e.g., 'Running tests', 'Fixing authentication bug')",
						},
					},
					"required": []string{"content", "status", "activeForm"},
				},
			},
		},
		"required": []string{"todos"},
	}
}

func (t *TodoWriteTool) Execute(ctx context.Context, params json.RawMessage) (*tool.Result, error) {
	var p struct {
		Todos []TodoItem `json:"todos"`
	}

	if err := json.Unmarshal(params, &p); err != nil {
		return &tool.Result{
			Success: false,
			Error:   fmt.Sprintf("invalid parameters: %v", err),
		}, nil
	}

	// Validate todos
	inProgressCount := 0
	for i, todo := range p.Todos {
		// Validate content
		if strings.TrimSpace(todo.Content) == "" {
			return &tool.Result{
				Success: false,
				Error:   fmt.Sprintf("todo #%d: content cannot be empty", i+1),
			}, nil
		}

		// Validate activeForm
		if strings.TrimSpace(todo.ActiveForm) == "" {
			return &tool.Result{
				Success: false,
				Error:   fmt.Sprintf("todo #%d: activeForm cannot be empty", i+1),
			}, nil
		}

		// Validate status
		switch todo.Status {
		case StatusPending, StatusInProgress, StatusCompleted:
			// Valid status
		default:
			return &tool.Result{
				Success: false,
				Error:   fmt.Sprintf("todo #%d: invalid status '%s' (must be 'pending', 'in_progress', or 'completed')", i+1, todo.Status),
			}, nil
		}

		// Count in_progress tasks
		if todo.Status == StatusInProgress {
			inProgressCount++
		}
	}

	// Validate: exactly 0 or 1 task should be in_progress
	if inProgressCount > 1 {
		return &tool.Result{
			Success: false,
			Error:   fmt.Sprintf("only ONE task can be 'in_progress' at a time, found %d", inProgressCount),
		}, nil
	}

	// If empty array, clear the todo list
	if len(p.Todos) == 0 {
		globalTodoStore.Clear()
		return &tool.Result{
			Success: true,
			Output:  "✨ Todo list cleared - all tasks complete!",
		}, nil
	}

	// Update the global todo store
	globalTodoStore.SetTodos(p.Todos)

	// Generate formatted output
	output := t.formatTodoList(p.Todos)

	return &tool.Result{
		Success: true,
		Output:  output,
		Data: map[string]any{
			"total":       len(p.Todos),
			"pending":     t.countByStatus(p.Todos, StatusPending),
			"in_progress": t.countByStatus(p.Todos, StatusInProgress),
			"completed":   t.countByStatus(p.Todos, StatusCompleted),
		},
	}, nil
}

func (t *TodoWriteTool) formatTodoList(todos []TodoItem) string {
	if len(todos) == 0 {
		return "No todos"
	}

	var sb strings.Builder

	// Summary line
	completed := t.countByStatus(todos, StatusCompleted)
	total := len(todos)
	sb.WriteString(fmt.Sprintf("📋 Todo List: %d/%d completed\n", completed, total))
	sb.WriteString(strings.Repeat("─", 50) + "\n")

	// List each todo
	for i, todo := range todos {
		var icon string
		var text string

		switch todo.Status {
		case StatusCompleted:
			icon = "✅"
			text = todo.Content
		case StatusInProgress:
			icon = "🔧"
			text = todo.ActiveForm
		case StatusPending:
			icon = "⏳"
			text = todo.Content
		}

		sb.WriteString(fmt.Sprintf("%d. %s %s\n", i+1, icon, text))
	}

	return sb.String()
}

func (t *TodoWriteTool) countByStatus(todos []TodoItem, status TodoStatus) int {
	count := 0
	for _, todo := range todos {
		if todo.Status == status {
			count++
		}
	}
	return count
}

// GetCurrentTodos returns the current todo list (for external access)
func GetCurrentTodos() []TodoItem {
	return globalTodoStore.GetTodos()
}
