// Package hooks provides middleware hooks for the gateway request/response pipeline.
//
// DESIGN: Hooks intercept and can modify requests/responses at various points
// in the gateway pipeline. They run INSIDE the gateway (unlike external/ which
// calls remote services).
//
// Pipeline flow:
//
//	Request → Adapter → [PRE-HOOKS] → Pipes → LLM API
//	                                            ↓
//	Response ← Adapter ← [POST-HOOKS] ← Pipes ←─┘
//
// STATUS: Not implemented in current release. This package provides interface
// definitions for future extensibility.
package hooks

// Hook defines the interface for pipeline hooks.
type Hook interface {
	// Name returns the hook identifier
	Name() string

	// Priority determines execution order (lower = earlier)
	Priority() int

	// Enabled returns whether the hook should run
	Enabled() bool
}

// PreRequestHook runs before the LLM API call.
type PreRequestHook interface {
	Hook
}

// PostResponseHook runs after the LLM API response.
type PostResponseHook interface {
	Hook
}

// ToolOutputHook intercepts tool outputs for processing.
type ToolOutputHook interface {
	Hook
}

// Registry holds registered hooks.
type Registry struct {
	preRequest   []PreRequestHook
	postResponse []PostResponseHook
	toolOutput   []ToolOutputHook
}

// NewRegistry creates a new hook registry.
func NewRegistry() *Registry {
	return &Registry{
		preRequest:   make([]PreRequestHook, 0),
		postResponse: make([]PostResponseHook, 0),
		toolOutput:   make([]ToolOutputHook, 0),
	}
}
