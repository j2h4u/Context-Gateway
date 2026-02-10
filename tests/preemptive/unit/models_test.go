// Package preemptive_test provides tests for preemptive summarization.
//
// Test organization follows the existing gateway test patterns:
// - models_test.go: Model context window tests
// - config_test.go: Configuration validation tests
// - session_test.go: Session management tests
// - detector_test.go: Compaction detection tests
// - worker_test.go: Background worker tests
// - manager_test.go: Integration tests
package preemptive_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/compresr/context-gateway/internal/preemptive"
)

// =============================================================================
// MODEL CONTEXT WINDOW TESTS
// =============================================================================

func TestGetModelContextWindow_KnownModel(t *testing.T) {
	tests := []struct {
		name              string
		model             string
		expectedMaxTokens int
		expectedOutput    int
	}{
		{
			name:              "claude-sonnet-4-5",
			model:             "claude-sonnet-4-5-20250929",
			expectedMaxTokens: 200000,
			expectedOutput:    64000,
		},
		{
			name:              "claude-opus-4-6",
			model:             "claude-opus-4-6",
			expectedMaxTokens: 200000,
			expectedOutput:    128000,
		},
		{
			name:              "gpt-4-turbo",
			model:             "gpt-4-turbo",
			expectedMaxTokens: 128000,
			expectedOutput:    4096,
		},
		{
			name:              "gpt-4o",
			model:             "gpt-4o",
			expectedMaxTokens: 128000,
			expectedOutput:    16384,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := preemptive.GetModelContextWindow(tt.model)
			assert.Equal(t, tt.model, mw.Model)
			assert.Equal(t, tt.expectedMaxTokens, mw.MaxTokens)
			assert.Equal(t, tt.expectedOutput, mw.OutputMax)
			assert.Equal(t, tt.expectedMaxTokens-tt.expectedOutput, mw.EffectiveMax)
		})
	}
}

func TestGetModelContextWindow_UnknownModel(t *testing.T) {
	mw := preemptive.GetModelContextWindow("unknown-model-xyz")

	// Should return conservative defaults
	assert.Equal(t, "unknown-model-xyz", mw.Model)
	assert.Equal(t, 128000, mw.MaxTokens)
	assert.Equal(t, 4096, mw.OutputMax)
	assert.Equal(t, 123904, mw.EffectiveMax)
}

func TestCalculateUsage_Normal(t *testing.T) {
	usage := preemptive.CalculateUsage(80000, 200000)

	assert.Equal(t, 80000, usage.InputTokens)
	assert.Equal(t, 200000, usage.MaxTokens)
	assert.InDelta(t, 40.0, usage.UsagePercent, 0.01)
}

func TestCalculateUsage_AtThreshold(t *testing.T) {
	usage := preemptive.CalculateUsage(160000, 200000)

	assert.InDelta(t, 80.0, usage.UsagePercent, 0.01)
}

func TestCalculateUsage_OverLimit(t *testing.T) {
	usage := preemptive.CalculateUsage(250000, 200000)

	// Should cap at 100%
	assert.InDelta(t, 100.0, usage.UsagePercent, 0.01)
}

func TestCalculateUsage_ZeroMax(t *testing.T) {
	usage := preemptive.CalculateUsage(50000, 0)

	// Should use fallback max
	assert.Equal(t, 128000, usage.MaxTokens)
	assert.InDelta(t, 39.06, usage.UsagePercent, 0.1)
}
