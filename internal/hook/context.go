package hook

import "context"

type contextKey string

const hookManagerKey contextKey = "hook_manager"

// WithManager adds a hook manager to the context
func WithManager(ctx context.Context, manager *Manager) context.Context {
	return context.WithValue(ctx, hookManagerKey, manager)
}

// FromContext retrieves the hook manager from context
func FromContext(ctx context.Context) *Manager {
	if manager, ok := ctx.Value(hookManagerKey).(*Manager); ok {
		return manager
	}
	return nil
}
