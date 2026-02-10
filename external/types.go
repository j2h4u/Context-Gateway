// Package external provides external/remote compression providers.
//
// STATUS: Not used in current release. This package is reserved for future
// integration with external compression services.
//
// DESIGN: This package is for EXTERNAL compression services (remote APIs).
// Internal compression strategies remain in internal/pipes/tool_output.
package external

import (
	"time"
)

// Provider defines the interface for external compression providers.
type Provider interface {
	// Name returns the provider name (e.g., "compresr", "custom")
	Name() string

	// Compress sends content to the external service for compression.
	// Returns compressed content and metadata.
	Compress(req *CompressRequest) (*CompressResponse, error)

	// HealthCheck verifies the provider is reachable.
	HealthCheck() error
}

// CompressRequest is the request sent to external compression providers.
// Format is provider-agnostic - the provider adapts it to its API.
type CompressRequest struct {
	// Content to compress
	Content string `json:"content"`

	// ToolName is the name of the tool that produced this output
	ToolName string `json:"tool_name"`

	// UserQuery is the last user message (for relevance-based compression)
	UserQuery string `json:"user_query,omitempty"`

	// SourceProvider indicates if this is from Anthropic or OpenAI format
	SourceProvider string `json:"source_provider"` // "anthropic" or "openai"

	// Model indicates which compression model to use (optional)
	Model string `json:"model,omitempty"`

	// MaxTokens is the target max tokens for compressed output (optional)
	MaxTokens int `json:"max_tokens,omitempty"`
}

// CompressResponse is the response from external compression providers.
type CompressResponse struct {
	// Compressed content
	Content string `json:"content"`

	// OriginalSize in bytes
	OriginalSize int `json:"original_size"`

	// CompressedSize in bytes
	CompressedSize int `json:"compressed_size"`

	// CompressionRatio (compressed/original)
	CompressionRatio float64 `json:"compression_ratio"`

	// Model used for compression
	Model string `json:"model,omitempty"`

	// CacheHit indicates if this was served from cache
	CacheHit bool `json:"cache_hit"`

	// ProcessingTime is how long compression took
	ProcessingTime time.Duration `json:"processing_time,omitempty"`
}

// Config holds configuration for external compression providers.
type Config struct {
	// Provider name: "compresr", "custom"
	Provider string `yaml:"provider"`

	// BaseURL of the external service
	BaseURL string `yaml:"base_url"`

	// APIKeyEnv is the environment variable name containing the API key
	APIKeyEnv string `yaml:"api_key_env"`

	// Endpoint path for compression (default: /v1/compress/tool-output)
	Endpoint string `yaml:"endpoint"`

	// Timeout for API calls
	Timeout time.Duration `yaml:"timeout"`

	// Model to request (optional, provider-specific)
	Model string `yaml:"model,omitempty"`

	// MaxRetries for failed requests
	MaxRetries int `yaml:"max_retries"`
}

// DefaultConfig returns sensible defaults for external compression.
func DefaultConfig() Config {
	return Config{
		Provider:   "compresr",
		Endpoint:   "/v1/compress/tool-output",
		Timeout:    30 * time.Second,
		MaxRetries: 2,
	}
}
