package builtin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestTodoWriteTool_Basic(t *testing.T) {
	tool := NewTodoWriteTool()

	// Test basic todo creation
	params := `{
		"todos": [
			{
				"content": "Fix bug in authentication",
				"status": "pending",
				"activeForm": "Fixing bug in authentication"
			},
			{
				"content": "Write tests",
				"status": "in_progress",
				"activeForm": "Writing tests"
			},
			{
				"content": "Update documentation",
				"status": "pending",
				"activeForm": "Updating documentation"
			}
		]
	}`

	result, err := tool.Execute(context.Background(), []byte(params))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("Expected success, got error: %s", result.Error)
	}

	// Check output format
	if !strings.Contains(result.Output, "📋 Todo List") {
		t.Errorf("Output should contain todo list header")
	}

	if !strings.Contains(result.Output, "0/3 completed") {
		t.Errorf("Output should show progress (0/3 completed): got %s", result.Output)
	}

	// Check data
	if result.Data["total"] != 3 {
		t.Errorf("Expected 3 total todos, got %v", result.Data["total"])
	}

	if result.Data["pending"] != 2 {
		t.Errorf("Expected 2 pending todos, got %v", result.Data["pending"])
	}

	if result.Data["in_progress"] != 1 {
		t.Errorf("Expected 1 in_progress todo, got %v", result.Data["in_progress"])
	}
}

func TestTodoWriteTool_StatusValidation(t *testing.T) {
	tool := NewTodoWriteTool()

	// Test invalid status
	params := `{
		"todos": [
			{
				"content": "Invalid task",
				"status": "invalid_status",
				"activeForm": "Invalid task"
			}
		]
	}`

	result, err := tool.Execute(context.Background(), []byte(params))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Success {
		t.Fatal("Expected failure for invalid status")
	}

	if !strings.Contains(result.Error, "invalid status") {
		t.Errorf("Error should mention invalid status: %s", result.Error)
	}
}

func TestTodoWriteTool_MultipleInProgress(t *testing.T) {
	tool := NewTodoWriteTool()

	// Test multiple in_progress tasks (should fail)
	params := `{
		"todos": [
			{
				"content": "Task 1",
				"status": "in_progress",
				"activeForm": "Task 1"
			},
			{
				"content": "Task 2",
				"status": "in_progress",
				"activeForm": "Task 2"
			}
		]
	}`

	result, err := tool.Execute(context.Background(), []byte(params))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Success {
		t.Fatal("Expected failure for multiple in_progress tasks")
	}

	if !strings.Contains(result.Error, "only ONE task") {
		t.Errorf("Error should mention one task limit: %s", result.Error)
	}
}

func TestTodoWriteTool_EmptyContent(t *testing.T) {
	tool := NewTodoWriteTool()

	// Test empty content
	params := `{
		"todos": [
			{
				"content": "",
				"status": "pending",
				"activeForm": "Empty task"
			}
		]
	}`

	result, err := tool.Execute(context.Background(), []byte(params))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Success {
		t.Fatal("Expected failure for empty content")
	}

	if !strings.Contains(result.Error, "content cannot be empty") {
		t.Errorf("Error should mention empty content: %s", result.Error)
	}
}

func TestTodoWriteTool_ClearTodos(t *testing.T) {
	tool := NewTodoWriteTool()

	// First create some todos
	params1 := `{
		"todos": [
			{
				"content": "Task 1",
				"status": "completed",
				"activeForm": "Task 1"
			}
		]
	}`

	result1, err := tool.Execute(context.Background(), []byte(params1))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result1.Success {
		t.Fatalf("Expected success: %s", result1.Error)
	}

	// Now clear with empty array
	params2 := `{"todos": []}`

	result2, err := tool.Execute(context.Background(), []byte(params2))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result2.Success {
		t.Fatalf("Expected success: %s", result2.Error)
	}

	if !strings.Contains(result2.Output, "cleared") {
		t.Errorf("Output should mention cleared: %s", result2.Output)
	}

	// Verify todos are cleared
	todos := GetCurrentTodos()
	if len(todos) != 0 {
		t.Errorf("Expected 0 todos after clear, got %d", len(todos))
	}
}

func TestTodoWriteTool_IconDisplay(t *testing.T) {
	tool := NewTodoWriteTool()

	params := `{
		"todos": [
			{
				"content": "Pending task",
				"status": "pending",
				"activeForm": "Pending task"
			},
			{
				"content": "Active task",
				"status": "in_progress",
				"activeForm": "Working on active task"
			},
			{
				"content": "Done task",
				"status": "completed",
				"activeForm": "Done task"
			}
		]
	}`

	result, err := tool.Execute(context.Background(), []byte(params))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("Expected success: %s", result.Error)
	}

	// Check icons
	if !strings.Contains(result.Output, "⏳") {
		t.Errorf("Output should contain pending icon ⏳")
	}

	if !strings.Contains(result.Output, "🔧") {
		t.Errorf("Output should contain in_progress icon 🔧")
	}

	if !strings.Contains(result.Output, "✅") {
		t.Errorf("Output should contain completed icon ✅")
	}

	// Check that active form is shown for in_progress
	if !strings.Contains(result.Output, "Working on active task") {
		t.Errorf("Output should show activeForm for in_progress task")
	}

	// Check that content is shown for completed
	if !strings.Contains(result.Output, "Done task") {
		t.Errorf("Output should show content for completed task")
	}
}

func TestTodoWriteTool_Parameters(t *testing.T) {
	tool := NewTodoWriteTool()

	params := tool.Parameters()

	// Verify structure
	if params["type"] != "object" {
		t.Errorf("Expected object type")
	}

	props := params["properties"].(map[string]any)
	todosParam := props["todos"].(map[string]any)

	if todosParam["type"] != "array" {
		t.Errorf("Expected todos to be array type")
	}

	// Check required fields
	required := params["required"].([]string)
	if len(required) != 1 || required[0] != "todos" {
		t.Errorf("Expected 'todos' to be required")
	}
}

func TestTodoStore_Concurrent(t *testing.T) {
	store := &TodoStore{
		todos: make([]TodoItem, 0),
	}

	// Test concurrent read/write
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			store.SetTodos([]TodoItem{
				{Content: "Task", Status: StatusPending, ActiveForm: "Task"},
			})
		}
		done <- true
	}()

	// Reader goroutines
	for j := 0; j < 10; j++ {
		go func() {
			for i := 0; i < 100; i++ {
				_ = store.GetTodos()
			}
		}()
	}

	<-done
	// If we get here without deadlock/race, test passes
}

func TestTodoItem_JSON(t *testing.T) {
	item := TodoItem{
		Content:    "Test task",
		Status:     StatusInProgress,
		ActiveForm: "Testing task",
	}

	// Marshal to JSON
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	// Unmarshal from JSON
	var decoded TodoItem
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	// Verify fields
	if decoded.Content != item.Content {
		t.Errorf("Content mismatch: got %s, want %s", decoded.Content, item.Content)
	}

	if decoded.Status != item.Status {
		t.Errorf("Status mismatch: got %s, want %s", decoded.Status, item.Status)
	}

	if decoded.ActiveForm != item.ActiveForm {
		t.Errorf("ActiveForm mismatch: got %s, want %s", decoded.ActiveForm, item.ActiveForm)
	}
}
