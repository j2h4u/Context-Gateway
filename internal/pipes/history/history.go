// Package history compresses conversation history.
//
// DESIGN: Compresses old conversation messages to reduce token count
// while preserving semantic meaning. Keeps recent messages uncompressed.
//
// FLOW:
//  1. Receives adapter via PipeContext
//  2. Calls adapter.ExtractHistory() to get messages for compression
//  3. Compresses messages (keeps recent N unchanged)
//  4. Calls adapter.ApplyHistory() to patch compressed history back
//
// STATUS: Stub implementation - Process() is a no-op.
package history

import (
	"github.com/compresr/context-gateway/internal/config"
	"github.com/compresr/context-gateway/internal/pipes"
)

// Pipe compresses conversation history.
type Pipe struct {
	enabled  bool
	strategy string
}

// New creates a new history compression pipe.
func New(cfg *config.Config) *Pipe {
	return &Pipe{
		enabled:  cfg.Pipes.History.Enabled,
		strategy: cfg.Pipes.History.Strategy,
	}
}

// Name returns the pipe name.
func (p *Pipe) Name() string {
	return "history"
}

// Strategy returns the processing strategy.
func (p *Pipe) Strategy() string {
	return p.strategy
}

// Enabled returns whether the pipe is active.
func (p *Pipe) Enabled() bool {
	return p.enabled
}

// Process compresses history on requests before sending to LLM.
//
// DESIGN: Pipes ALWAYS delegate extraction to adapters. Pipes contain NO
// provider-specific logic - they only implement compression/filtering logic.
func (p *Pipe) Process(ctx *pipes.PipeContext) ([]byte, error) {
	if !p.enabled || p.strategy == config.StrategyPassthrough {
		return ctx.OriginalRequest, nil
	}

	// TODO: Implement history compression
	// 1. extracted, err := ctx.Adapter.ExtractHistory(ctx.OriginalRequest, &adapters.HistoryOptions{KeepRecent: 5})
	// 2. Compress extracted messages
	// 3. results := compress(extracted)
	// 4. return ctx.Adapter.ApplyHistory(ctx.OriginalRequest, results)

	return ctx.OriginalRequest, nil
}
