// Compaction request detection.
//
// DESIGN: Provider-specific detectors for identifying compaction requests.
// Each provider (Anthropic, OpenAI) has its own detector implementation.
//
// Usage:
//
//	detector := GetDetector(adapters.ProviderAnthropic, config)
//	result := detector.Detect(body)
package preemptive

import (
	"encoding/json"
	"strings"

	"github.com/compresr/context-gateway/internal/adapters"
)

// =============================================================================
// DETECTOR INTERFACE
// =============================================================================

// CompactionDetector is the interface for provider-specific compaction detection.
type CompactionDetector interface {
	Detect(body []byte) DetectionResult
}

// =============================================================================
// DETECTOR FACTORY
// =============================================================================

// GetDetector returns the appropriate detector for the given provider.
func GetDetector(provider adapters.Provider, cfg DetectorsConfig) CompactionDetector {
	switch provider {
	case adapters.ProviderAnthropic:
		return &ClaudeDetector{patterns: cfg.ClaudeCode.PromptPatterns}
	case adapters.ProviderOpenAI:
		return &OpenAIDetector{patterns: cfg.Codex.PromptPatterns}
	default:
		return &ClaudeDetector{patterns: cfg.ClaudeCode.PromptPatterns}
	}
}

// =============================================================================
// CLAUDE DETECTOR (Anthropic)
// =============================================================================

// ClaudeDetector detects Claude Code compaction requests.
type ClaudeDetector struct {
	patterns []string
}

func (d *ClaudeDetector) Detect(body []byte) DetectionResult {
	var req requestBody
	if err := json.Unmarshal(body, &req); err != nil {
		return DetectionResult{}
	}

	// Check last user message
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			text := strings.ToLower(ExtractText(req.Messages[i].Content))
			for _, phrase := range d.patterns {
				if strings.Contains(text, strings.ToLower(phrase)) {
					return DetectionResult{
						IsCompactionRequest: true,
						DetectedBy:          "claude_code_prompt",
						Confidence:          0.95,
						Details:             map[string]interface{}{"matched_phrase": phrase},
					}
				}
			}
			break
		}
	}

	return DetectionResult{}
}

// =============================================================================
// OPENAI DETECTOR (Codex, GPT, etc.)
// =============================================================================

// OpenAIDetector detects OpenAI-based compaction requests.
type OpenAIDetector struct {
	patterns []string
}

func (d *OpenAIDetector) Detect(body []byte) DetectionResult {
	var req requestBody
	if err := json.Unmarshal(body, &req); err != nil {
		return DetectionResult{}
	}

	// Check last user message
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			text := strings.ToLower(ExtractText(req.Messages[i].Content))
			for _, phrase := range d.patterns {
				if strings.Contains(text, strings.ToLower(phrase)) {
					return DetectionResult{
						IsCompactionRequest: true,
						DetectedBy:          "openai_prompt",
						Confidence:          0.7,
						Details:             map[string]interface{}{"matched_phrase": phrase},
					}
				}
			}
			break
		}
	}

	return DetectionResult{}
}

// =============================================================================
// SHARED TYPES
// =============================================================================

type requestBody struct {
	Messages []struct {
		Role    string      `json:"role"`
		Content interface{} `json:"content"`
	} `json:"messages"`
}
