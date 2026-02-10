// Pipes configuration - compression pipeline settings.
//
// STATUS: All compression pipes are disabled in current release.
// Only preemptive summarization is enabled.
//
// DESIGN: Three independent pipes process requests:
//   - History:       Compress conversation history
//   - ToolOutput:    Compress tool results, store originals for expand_context
//   - ToolDiscovery: Filter irrelevant tools
//
// Each pipe has a STRATEGY: "passthrough" (noop) or "api" (call compression service).
//
// NOTE: This file defines pipe-specific configuration types.
// The main Config struct in config/ imports and uses these types.
package pipes

import (
	"fmt"
	"time"
)

// =============================================================================
// STRATEGY CONSTANTS
// =============================================================================

// Strategy constants for pipe execution.
const (
	StrategyPassthrough      = "passthrough"       // Do nothing, pass through unchanged
	StrategyAPI              = "api"               // Call compresr platform API
	StrategySimple           = "simple"            // Simple compression (first N words)
	StrategyExternalProvider = "external_provider" // Call external LLM provider (OpenAI/Anthropic) directly
)

// =============================================================================
// COMPRESSION THRESHOLDS
// =============================================================================

// CompressionThreshold represents user-selectable compression trigger thresholds.
// Set via X-Compression-Threshold header.
type CompressionThreshold string

const (
	ThresholdOff  CompressionThreshold = "off"  // No compression ever
	Threshold256  CompressionThreshold = "256"  // Compress when > 256 tokens (default)
	Threshold1K   CompressionThreshold = "1k"   // Compress when > 1,000 tokens
	Threshold2K   CompressionThreshold = "2k"   // Compress when > 2,000 tokens
	Threshold4K   CompressionThreshold = "4k"   // Compress when > 4,000 tokens
	Threshold8K   CompressionThreshold = "8k"   // Compress when > 8,000 tokens
	Threshold16K  CompressionThreshold = "16k"  // Compress when > 16,000 tokens
	Threshold32K  CompressionThreshold = "32k"  // Compress when > 32,000 tokens
	Threshold64K  CompressionThreshold = "64k"  // Compress when > 64,000 tokens
	Threshold128K CompressionThreshold = "128k" // Compress when > 128,000 tokens
)

// ThresholdTokenCounts maps thresholds to token counts.
var ThresholdTokenCounts = map[CompressionThreshold]int{
	ThresholdOff: 0, Threshold256: 256, Threshold1K: 1000, Threshold2K: 2000, Threshold4K: 4000,
	Threshold8K: 8000, Threshold16K: 16000, Threshold32K: 32000, Threshold64K: 64000, Threshold128K: 128000,
}

// DefaultThreshold is the default compression threshold when none specified.
const DefaultThreshold = Threshold256

// ParseCompressionThreshold parses a threshold string from header, returns default if invalid.
func ParseCompressionThreshold(s string) CompressionThreshold {
	switch CompressionThreshold(s) {
	case ThresholdOff, Threshold256, Threshold1K, Threshold2K, Threshold4K, Threshold8K, Threshold16K, Threshold32K, Threshold64K, Threshold128K:
		return CompressionThreshold(s)
	default:
		return DefaultThreshold
	}
}

// TokenCount returns the token count for this threshold.
// Returns -1 for ThresholdOff (meaning compression disabled).
func (t CompressionThreshold) TokenCount() int {
	if t == ThresholdOff {
		return -1
	}
	if count, ok := ThresholdTokenCounts[t]; ok {
		return count
	}
	return ThresholdTokenCounts[DefaultThreshold]
}

// =============================================================================
// PIPES CONFIG - Root configuration for all pipes
// =============================================================================

// Config contains configuration for all compression pipes.
type Config struct {
	History       HistoryConfig       `yaml:"history"`        // Conversation history compression
	ToolOutput    ToolOutputConfig    `yaml:"tool_output"`    // Tool output compression
	ToolDiscovery ToolDiscoveryConfig `yaml:"tool_discovery"` // Tool filtering
}

// Validate validates pipe configurations.
func (p *Config) Validate() error {
	if err := p.History.Validate(); err != nil {
		return err
	}
	if err := p.ToolOutput.Validate(); err != nil {
		return err
	}
	if err := p.ToolDiscovery.Validate(); err != nil {
		return err
	}
	return nil
}

// =============================================================================
// HISTORY PIPE CONFIG
// =============================================================================

// HistoryConfig configures conversation history compression.
type HistoryConfig struct {
	Enabled          bool   `yaml:"enabled"`           // Enable this pipe
	Strategy         string `yaml:"strategy"`          // passthrough | api
	FallbackStrategy string `yaml:"fallback_strategy"` // Fallback when primary fails

	// API strategy config (only needed when strategy=api)
	API APIConfig `yaml:"api,omitempty"`

	// Compression settings
	DefaultThreshold string  `yaml:"default_threshold"` // Default threshold when header not provided
	TargetRatio      float64 `yaml:"target_ratio"`      // Compress to this ratio (e.g., 0.5 = 50%)
	KeepRecent       int     `yaml:"keep_recent"`       // Always keep last N messages uncompressed
}

// Validate validates history pipe config.
func (h *HistoryConfig) Validate() error {
	if !h.Enabled {
		return nil // Disabled pipes don't need strategy
	}
	if h.Strategy == "" || h.Strategy == StrategyPassthrough {
		return nil
	}
	if h.Strategy == StrategyAPI && h.API.Endpoint == "" {
		return fmt.Errorf("history: api.endpoint required when strategy=api")
	}
	return nil
}

// =============================================================================
// TOOL OUTPUT PIPE CONFIG
// =============================================================================

// ToolOutputConfig configures tool result compression.
type ToolOutputConfig struct {
	Enabled          bool   `yaml:"enabled"`           // Enable this pipe
	Strategy         string `yaml:"strategy"`          // passthrough | api | external_provider
	FallbackStrategy string `yaml:"fallback_strategy"` // Fallback when primary fails

	// API strategy config (for strategy=api or strategy=external_provider)
	API APIConfig `yaml:"api,omitempty"`

	// Compression settings
	MinBytes    int     `yaml:"min_bytes"`    // Below this size, no compression (default: 2048)
	MaxBytes    int     `yaml:"max_bytes"`    // Above this size, skip compression (V2, default: 64KB)
	TargetRatio float64 `yaml:"target_ratio"` // Compress to this ratio (e.g., 0.5 = 50%)

	// Expand context feature
	EnableExpandContext bool `yaml:"enable_expand_context"` // Inject expand_context tool
	IncludeExpandHint   bool `yaml:"include_expand_hint"`   // Add hint to compressed content
}

// Validate validates tool output pipe config.
func (t *ToolOutputConfig) Validate() error {
	if !t.Enabled {
		return nil // Disabled pipes don't need strategy
	}
	if t.Strategy == "" || t.Strategy == StrategyPassthrough {
		return nil
	}
	if t.Strategy == StrategySimple {
		return nil
	}
	if t.Strategy == StrategyAPI {
		if t.API.Endpoint == "" {
			return fmt.Errorf("tool_output: api.endpoint required")
		}
		return nil
	}
	if t.Strategy == StrategyExternalProvider {
		if t.API.Endpoint == "" {
			return fmt.Errorf("tool_output: api.endpoint required when strategy=external_provider")
		}
		return nil
	}
	return fmt.Errorf("tool_output: unknown strategy %q, must be 'passthrough', 'simple', 'api', or 'external_provider'", t.Strategy)
}

// =============================================================================
// TOOL DISCOVERY PIPE CONFIG
// =============================================================================

// ToolDiscoveryConfig configures tool filtering.
type ToolDiscoveryConfig struct {
	Enabled          bool   `yaml:"enabled"`           // Enable this pipe
	Strategy         string `yaml:"strategy"`          // passthrough | api
	FallbackStrategy string `yaml:"fallback_strategy"` // Fallback when primary fails

	// API strategy config
	API APIConfig `yaml:"api,omitempty"`

	// Filtering settings
	MinTools    int     `yaml:"min_tools"`    // Below this count, no filtering
	TargetRatio float64 `yaml:"target_ratio"` // Keep this ratio of tools
}

// Validate validates tool discovery pipe config.
func (d *ToolDiscoveryConfig) Validate() error {
	if !d.Enabled {
		return nil // Disabled pipes don't need strategy
	}
	if d.Strategy == "" || d.Strategy == StrategyPassthrough {
		return nil
	}
	if d.Strategy == StrategyAPI && d.API.Endpoint == "" {
		return fmt.Errorf("tool_discovery: api.endpoint required when strategy=api")
	}
	return nil
}

// =============================================================================
// STRATEGY-SPECIFIC CONFIGS
// =============================================================================

// APIConfig contains settings for calling compression APIs.
// Not used in current release - tool output compression is disabled.
type APIConfig struct {
	Endpoint      string        `yaml:"endpoint"`       // API endpoint URL
	APIKey        string        `yaml:"api_key"`        // API authentication key
	Model         string        `yaml:"model"`          // Compression model to use
	Timeout       time.Duration `yaml:"timeout"`        // Request timeout
	QueryAgnostic bool          `yaml:"query_agnostic"` // If true, compression is context-agnostic
}
