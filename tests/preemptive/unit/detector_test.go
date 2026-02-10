package preemptive_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/compresr/context-gateway/internal/adapters"
	"github.com/compresr/context-gateway/internal/preemptive"
)

// =============================================================================
// COMPACTION DETECTOR TESTS
// =============================================================================

func TestCompactionDetector_NoPatterns(t *testing.T) {
	cfg := preemptive.DetectorsConfig{
		ClaudeCode: preemptive.ClaudeCodeDetectorConfig{
			Enabled:        false,
			PromptPatterns: []string{},
		},
	}

	detector := preemptive.GetDetector(adapters.ProviderAnthropic, cfg)
	result := detector.Detect([]byte(`{"messages": []}`))

	assert.False(t, result.IsCompactionRequest)
}

// =============================================================================
// CLAUDE CODE DETECTOR TESTS
// =============================================================================

func TestClaudeCodeDetector_PromptPattern_Summarize(t *testing.T) {
	cfg := preemptive.DetectorsConfig{
		ClaudeCode: preemptive.ClaudeCodeDetectorConfig{
			Enabled: true,
			PromptPatterns: []string{
				"summarize this conversation",
				"compact the context",
			},
		},
	}

	detector := preemptive.GetDetector(adapters.ProviderAnthropic, cfg)

	body := []byte(`{
		"messages": [
			{"role": "user", "content": "Please summarize this conversation for me"}
		]
	}`)

	result := detector.Detect(body)

	assert.True(t, result.IsCompactionRequest)
	assert.Equal(t, "claude_code_prompt", result.DetectedBy)
	assert.InDelta(t, 0.95, result.Confidence, 0.01)
}

func TestClaudeCodeDetector_PromptPattern_Compact(t *testing.T) {
	cfg := preemptive.DetectorsConfig{
		ClaudeCode: preemptive.ClaudeCodeDetectorConfig{
			Enabled: true,
			PromptPatterns: []string{
				"compact the context",
			},
		},
	}

	detector := preemptive.GetDetector(adapters.ProviderAnthropic, cfg)

	body := []byte(`{
		"messages": [
			{"role": "user", "content": "Let's compact the context now"}
		]
	}`)

	result := detector.Detect(body)

	assert.True(t, result.IsCompactionRequest)
}

func TestClaudeCodeDetector_PromptPattern_CaseInsensitive(t *testing.T) {
	cfg := preemptive.DetectorsConfig{
		ClaudeCode: preemptive.ClaudeCodeDetectorConfig{
			Enabled: true,
			PromptPatterns: []string{
				"summarize this conversation",
			},
		},
	}

	detector := preemptive.GetDetector(adapters.ProviderAnthropic, cfg)

	body := []byte(`{
		"messages": [
			{"role": "user", "content": "SUMMARIZE THIS CONVERSATION please"}
		]
	}`)

	result := detector.Detect(body)

	assert.True(t, result.IsCompactionRequest)
}

func TestClaudeCodeDetector_PromptPattern_NoMatch(t *testing.T) {
	cfg := preemptive.DetectorsConfig{
		ClaudeCode: preemptive.ClaudeCodeDetectorConfig{
			Enabled: true,
			PromptPatterns: []string{
				"summarize this conversation",
			},
		},
	}

	detector := preemptive.GetDetector(adapters.ProviderAnthropic, cfg)

	body := []byte(`{
		"messages": [
			{"role": "user", "content": "Please help me with this code"}
		]
	}`)

	result := detector.Detect(body)

	assert.False(t, result.IsCompactionRequest)
}

func TestClaudeCodeDetector_OnlyLastUserMessage(t *testing.T) {
	cfg := preemptive.DetectorsConfig{
		ClaudeCode: preemptive.ClaudeCodeDetectorConfig{
			Enabled: true,
			PromptPatterns: []string{
				"summarize this conversation",
			},
		},
	}

	detector := preemptive.GetDetector(adapters.ProviderAnthropic, cfg)

	// Pattern in earlier message, not in last
	body := []byte(`{
		" messages": [
			{"role": "user", "content": "summarize this conversation"},
			{"role": "assistant", "content": "Here is the summary..."},
			{"role": "user", "content": "Thanks! Now help me with code"}
		]
	}`)

	result := detector.Detect(body)

	// Should NOT detect - only checks last user message
	assert.False(t, result.IsCompactionRequest)
}

func TestClaudeCodeDetector_ContentBlockArray(t *testing.T) {
	cfg := preemptive.DetectorsConfig{
		ClaudeCode: preemptive.ClaudeCodeDetectorConfig{
			Enabled: true,
			PromptPatterns: []string{
				"summarize this conversation",
			},
		},
	}

	detector := preemptive.GetDetector(adapters.ProviderAnthropic, cfg)

	// Anthropic format with content blocks
	body := []byte(`{
		"messages": [
			{"role": "user", "content": [
				{"type": "text", "text": "Please summarize this conversation"}
			]}
		]
	}`)

	result := detector.Detect(body)

	assert.True(t, result.IsCompactionRequest)
}

// =============================================================================
// OPENAI DETECTOR TESTS
// =============================================================================

func TestOpenAIDetector_PromptPattern(t *testing.T) {
	cfg := preemptive.DetectorsConfig{
		Codex: preemptive.CodexDetectorConfig{
			Enabled:        true,
			PromptPatterns: []string{"compact history"},
		},
	}

	detector := preemptive.GetDetector(adapters.ProviderOpenAI, cfg)

	body := []byte(`{
		"messages": [
			{"role": "user", "content": "Please compact history now"}
		]
	}`)

	result := detector.Detect(body)

	assert.True(t, result.IsCompactionRequest)
	assert.Equal(t, "openai_prompt", result.DetectedBy)
}

// =============================================================================
// MALFORMED INPUT TESTS
// =============================================================================

func TestDetector_InvalidJSON(t *testing.T) {
	cfg := preemptive.DetectorsConfig{
		ClaudeCode: preemptive.ClaudeCodeDetectorConfig{
			Enabled: true,
		},
	}

	detector := preemptive.GetDetector(adapters.ProviderAnthropic, cfg)
	result := detector.Detect([]byte(`not valid json`))

	assert.False(t, result.IsCompactionRequest)
}

func TestDetector_EmptyMessages(t *testing.T) {
	cfg := preemptive.DetectorsConfig{
		ClaudeCode: preemptive.ClaudeCodeDetectorConfig{
			Enabled:        true,
			PromptPatterns: []string{"summarize"},
		},
	}

	detector := preemptive.GetDetector(adapters.ProviderAnthropic, cfg)
	result := detector.Detect([]byte(`{"messages": []}`))

	assert.False(t, result.IsCompactionRequest)
}

func TestDetector_NoUserMessages(t *testing.T) {
	cfg := preemptive.DetectorsConfig{
		ClaudeCode: preemptive.ClaudeCodeDetectorConfig{
			Enabled:        true,
			PromptPatterns: []string{"summarize"},
		},
	}

	detector := preemptive.GetDetector(adapters.ProviderAnthropic, cfg)

	body := []byte(`{
		"messages": [
			{"role": "assistant", "content": "summarize this please"}
		]
	}`)

	result := detector.Detect(body)

	// Should not match - only checks user messages for prompts
	assert.False(t, result.IsCompactionRequest)
}
