// Package tooldiscovery filters tools dynamically based on relevance.
//
// DESIGN: Filters tool definitions based on relevance to the current
// query, reducing token overhead when many tools are registered.
//
// FLOW:
//  1. Receives adapter via PipeContext
//  2. Calls adapter.ExtractToolDiscovery() to get tool definitions
//  3. Filters tools based on relevance to query
//  4. Calls adapter.ApplyToolDiscovery() to patch filtered tools back
//
// STATUS: Stub implementation - Process() is a no-op.
package tooldiscovery

import (
	"github.com/compresr/context-gateway/internal/config"
	"github.com/compresr/context-gateway/internal/pipes"
)

// Pipe filters tools dynamically based on relevance to the current query.
type Pipe struct {
	enabled  bool
	strategy string
}

// New creates a new tool discovery pipe.
func New(cfg *config.Config) *Pipe {
	return &Pipe{
		enabled:  cfg.Pipes.ToolDiscovery.Enabled,
		strategy: cfg.Pipes.ToolDiscovery.Strategy,
	}
}

// Name returns the pipe name.
func (p *Pipe) Name() string {
	return "tool_discovery"
}

// Strategy returns the processing strategy.
func (p *Pipe) Strategy() string {
	return p.strategy
}

// Enabled returns whether the pipe is active.
func (p *Pipe) Enabled() bool {
	return p.enabled
}

// Process filters tools before sending to LLM.
//
// DESIGN: Pipes ALWAYS delegate extraction to adapters. Pipes contain NO
// provider-specific logic - they only implement compression/filtering logic.
//
// STATUS: Not implemented in current release. Returns request unchanged.
func (p *Pipe) Process(ctx *pipes.PipeContext) ([]byte, error) {
	if !p.enabled || p.strategy == config.StrategyPassthrough {
		return ctx.OriginalRequest, nil
	}

	// Tool discovery/filtering not implemented in current release
	return ctx.OriginalRequest, nil
}
